package interchaintest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// Amounts are chosen so every split is exact: 1,200,000 a day over 4,000,000
// staked is 0.3 per unit and over 6,000,000 it is 0.2, so the accumulator
// never truncates and every payout can be asserted to the unit.
const (
	drCreatorFunds = 300_000_000_000 // 300,000 BZE: three denoms, DR, two prizes, schedule
	drUserFunds    = 1_000_000_000   // 1,000 BZE for gas

	drDailyAmount  = 1_200_000
	drScheduleDays = 3
	drExtraDays    = 1
	drTotalDays    = drScheduleDays + drExtraDays

	drAliceStake = 3_000_000
	drBobJoin    = 1_000_000
	drBobTopUp   = 2_000_000
	drCarolStake = 1_000_000
	drAirdrop    = 600_000

	drRewardsModule = "rewards"
	drEventPrefix   = "bze.rewards."

	// A day epoch is 30s; waiting for the next tick never needs more than one.
	dayTickTimeout = 75 * time.Second
	// The unlock is 24 hour epochs (5s each) after the exit.
	unlockTimeout = 4 * time.Minute
)

// TestDenomRewards runs the denom rewards lifecycle on one network with fast
// epochs: real txs, the day-epoch distribution queue in EndBlock and the
// hour-epoch pending-unlock queue. Subtests run in order and share state; the
// step numbers match the scenario of the suite's design.
//
// Timeline (D = the day epoch right after the zero-staker tick):
//
//	D    carol joins and exits (10a), alice and bob join (5)
//	D+1  first payout; bob tops up (6)
//	D+2  second payout; airdrop (7), alice claims (8)
//	D+3  third payout
//	D+4  fourth payout finishes the schedule (9)
//	D+5  nothing more is paid; everyone claims (9b)
//
// Carol's join and exit happen in epoch D, before anyone else stakes and
// before the next tick, so her stake never dilutes a payout while her unlock
// (24 hour epochs) runs in parallel with the schedule.
func TestDenomRewards(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupDenomRewards(t, ctx, net.Chain)

	t.Run("1 create denom reward, duplicate rejected", s.createDenomReward)
	t.Run("2 create and extend a schedule", s.createSchedule)
	t.Run("3 airdrop with no stakers rejected", s.airdropWithoutStakers)
	t.Run("4 zero-staker day is skipped", s.zeroStakerDay)
	t.Run("10a exit queues the stake for unlock", s.exitQueuesUnlock)
	t.Run("5 join and first payout", s.joinAndFirstPayout)
	t.Run("6 top-up keeps accrued rewards", s.topUp)
	t.Run("7 airdrop in a new prize denom", s.airdropNewPrize)
	t.Run("8 multi-prize claim", s.multiPrizeClaim)
	t.Run("9 schedule finishes after its last payout", s.scheduleFinish)
	t.Run("9b nothing is paid after the finish, all claimed", s.afterFinish)
	t.Run("10b unlock returns the stake", s.unlockReturnsStake)
	t.Run("11 escrow conservation at the end", s.escrowAtEnd)
}

type denomRewardsSuite struct {
	ctx context.Context
	bze *cosmos.CosmosChain

	creator, alice, bob, carol ibc.Wallet
	stk, prz, prz2             string
	module                     string
	params                     rewardsParams

	// baseline rewards module balances, before the DR exists
	baseline map[string]sdkmath.Int

	// stakers with a live position, for the escrow check
	stakers []ibc.Wallet

	scheduleID string
	// window is the day epoch right after the zero-staker tick (D)
	window int64
	// carolUnlock is the hour epoch carol's stake unlocks at
	carolUnlock int64
	// bobSettled is the prz bob's top-up settled to his balance
	bobSettled sdkmath.Int
}

func setupDenomRewards(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *denomRewardsSuite {
	s := &denomRewardsSuite{ctx: ctx, bze: bze, baseline: map[string]sdkmath.Int{}}

	s.creator = interchaintest.GetAndFundTestUsers(t, ctx, "drcreator", sdkmath.NewInt(drCreatorFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "dr", sdkmath.NewInt(drUserFunds), bze, bze, bze)
	s.alice, s.bob, s.carol = users[0], users[1], users[2]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	factory := "factory/" + s.creator.FormattedAddress() + "/"
	s.stk, s.prz, s.prz2 = factory+"stk", factory+"prz", factory+"prz2"
	supply := map[string]int64{
		s.stk:  drAliceStake + drBobJoin + drBobTopUp + drCarolStake,
		s.prz:  drDailyAmount * drTotalDays,
		s.prz2: drAirdrop,
	}
	for _, sub := range []string{"stk", "prz", "prz2"} {
		s.mustExec(t, s.creator, "tokenfactory", "create-denom", sub)
		denom := factory + sub
		s.mustExec(t, s.creator, "tokenfactory", "mint", coinStr(supply[denom], denom))
	}
	for _, send := range []struct {
		to     ibc.Wallet
		amount int64
	}{
		{s.alice, drAliceStake},
		{s.bob, drBobJoin + drBobTopUp},
		{s.carol, drCarolStake},
	} {
		s.mustExec(t, s.creator, "bank", "send", s.creator.FormattedAddress(), send.to.FormattedAddress(), coinStr(send.amount, s.stk))
	}

	var err error
	s.module, err = helpers.ModuleAddress(ctx, bze, drRewardsModule)
	require.NoError(t, err)
	for _, denom := range []string{s.stk, s.prz, s.prz2} {
		s.baseline[denom] = s.balance(t, s.module, denom)
	}
	s.params = s.queryParams(t)
	return s
}

func (s *denomRewardsSuite) createDenomReward(t *testing.T) {
	fee := s.params.CreateDenomRewardFee
	require.Equal(t, chain.Denom, fee.Denom, "create_denom_reward_fee is not in %s", chain.Denom)
	before := s.balance(t, s.creator.FormattedAddress(), chain.Denom)

	res := s.mustExec(t, s.creator, "rewards", "create-denom-reward", s.stk)
	s.requireTxEvent(t, res, "DenomRewardCreateEvent", map[string]string{"denom": s.stk})
	after := s.balance(t, s.creator.FormattedAddress(), chain.Denom)
	require.Equal(t, before.Sub(fee.Amount).Sub(s.txFee(t, res)).String(), after.String(), "creator must pay exactly the create fee and the tx fee")

	dr := s.denomReward(t)
	require.Equal(t, s.stk, dr.StakingDenom)
	require.Equal(t, int64(chain.DenomRewardLock), int64(dr.Lock))
	require.Equal(t, int64(s.params.DenomRewardLock), int64(dr.Lock))
	require.Equal(t, int64(s.params.DenomRewardMinStake), int64(dr.MinStake))
	require.Zero(t, int64(dr.MinStake))
	require.Equal(t, "0", dr.staked().String())

	s.requireRejected(t, "a denom reward already exists", s.creator, "rewards", "create-denom-reward", s.stk)
	require.Equal(t, dr, s.denomReward(t), "a rejected duplicate must not change the denom reward")
}

func (s *denomRewardsSuite) createSchedule(t *testing.T) {
	creator := s.creator.FormattedAddress()
	ubzeBefore := s.balance(t, creator, chain.Denom)
	przBefore := s.balance(t, creator, s.prz)
	escrowBefore := s.balance(t, s.module, s.prz)

	res := s.mustExec(t, s.creator, "rewards", "create-denom-reward-schedule",
		s.stk, s.prz, strconv.Itoa(drDailyAmount), strconv.Itoa(drScheduleDays))
	s.requireTxEvent(t, res, "DenomRewardPrizeCreateEvent", map[string]string{"denom": s.stk, "prize_denom": s.prz})
	created := s.requireTxEvent(t, res, "DenomRewardScheduleCreateEvent", map[string]string{
		"denom":        s.stk,
		"prize_denom":  s.prz,
		"daily_amount": strconv.Itoa(drDailyAmount),
		"duration":     strconv.Itoa(drScheduleDays),
	})
	s.scheduleID = created.Attributes["schedule_id"]
	require.NotEmpty(t, s.scheduleID)

	fees := s.params.CreateDenomRewardPrizeFee.Amount.Add(s.params.AddDenomRewardScheduleFee.Amount)
	require.Equal(t, ubzeBefore.Sub(fees).Sub(s.txFee(t, res)).String(), s.balance(t, creator, chain.Denom).String(),
		"creator must pay exactly the prize fee, the schedule fee and the tx fee")
	budget := sdkmath.NewInt(drDailyAmount * drScheduleDays)
	require.Equal(t, przBefore.Sub(budget).String(), s.balance(t, creator, s.prz).String())
	require.Equal(t, escrowBefore.Add(budget).String(), s.balance(t, s.module, s.prz).String())
	s.requireSchedule(t, drScheduleDays, 0)

	// extend by one day: no fee, only the extra budget
	ubzeBefore = s.balance(t, creator, chain.Denom)
	res = s.mustExec(t, s.creator, "rewards", "update-denom-reward-schedule", s.stk, s.scheduleID, strconv.Itoa(drExtraDays))
	s.requireTxEvent(t, res, "DenomRewardScheduleUpdateEvent", map[string]string{
		"schedule_id": s.scheduleID,
		"duration":    strconv.Itoa(drTotalDays),
	})
	require.Equal(t, ubzeBefore.Sub(s.txFee(t, res)).String(), s.balance(t, creator, chain.Denom).String(), "extending a schedule has no fee")
	require.Equal(t, escrowBefore.Add(budget).AddRaw(drDailyAmount*drExtraDays).String(), s.balance(t, s.module, s.prz).String())
	s.requireSchedule(t, drTotalDays, 0)

	s.requireEscrow(t)
}

func (s *denomRewardsSuite) airdropWithoutStakers(t *testing.T) {
	s.requireRejected(t, "denom reward has no stakers", s.creator,
		"rewards", "distribute-denom-rewards", s.stk, s.prz, "1000")
}

func (s *denomRewardsSuite) zeroStakerDay(t *testing.T) {
	require.Equal(t, "0", s.denomReward(t).staked().String())
	day := s.waitDayTick(t)
	s.window = int64(day.CurrentEpoch)

	s.requireSchedule(t, drTotalDays, 0)
	require.Empty(t, s.distributionEvents(t, day, s.prz), "a zero-staker day must not distribute")
}

func (s *denomRewardsSuite) exitQueuesUnlock(t *testing.T) {
	carol := s.carol.FormattedAddress()
	stkBefore := s.balance(t, carol, s.stk)

	res := s.mustExec(t, s.carol, "rewards", "join-denom-reward", s.stk, strconv.Itoa(drCarolStake))
	s.requireTxEvent(t, res, "DenomRewardJoinEvent", map[string]string{"denom": s.stk, "address": carol, "amount": strconv.Itoa(drCarolStake)})

	res = s.mustExec(t, s.carol, "rewards", "exit-denom-reward", s.stk)
	s.requireTxEvent(t, res, "DenomRewardExitEvent", map[string]string{"denom": s.stk, "address": carol, "amount": strconv.Itoa(drCarolStake)})
	require.Greater(t, res.GasUsed, int64(s.params.ExtraGasForDenomExit), "exit must charge extra_gas_for_denom_exit")
	s.requireSameDay(t, s.window, "carol's join and exit")

	hour, err := helpers.Query[struct {
		CurrentEpoch helpers.Int64 `json:"current_epoch"`
	}](s.ctx, s.bze, "epochs", "identifier", "hour", "--height", strconv.FormatInt(res.Height, 10))
	require.NoError(t, err)
	s.carolUnlock = int64(hour.CurrentEpoch) + int64(s.params.DenomRewardLock)*24

	entry, found := s.pendingUnlock(t, s.carolUnlockIndex())
	require.True(t, found, "no pending unlock %s", s.carolUnlockIndex())
	require.Equal(t, carol, entry.Address)
	require.Equal(t, s.stk, entry.Denom)
	require.Equal(t, strconv.Itoa(drCarolStake), entry.Amount)

	require.Equal(t, stkBefore.SubRaw(drCarolStake).String(), s.balance(t, carol, s.stk).String(), "the stake stays locked until the unlock epoch")
	require.Equal(t, "0", s.denomReward(t).staked().String())
}

func (s *denomRewardsSuite) joinAndFirstPayout(t *testing.T) {
	alice, bob := s.alice.FormattedAddress(), s.bob.FormattedAddress()
	res := s.mustExec(t, s.alice, "rewards", "join-denom-reward", s.stk, strconv.Itoa(drAliceStake))
	s.requireTxEvent(t, res, "DenomRewardJoinEvent", map[string]string{"denom": s.stk, "address": alice, "amount": strconv.Itoa(drAliceStake)})
	res = s.mustExec(t, s.bob, "rewards", "join-denom-reward", s.stk, strconv.Itoa(drBobJoin))
	s.requireTxEvent(t, res, "DenomRewardJoinEvent", map[string]string{"denom": s.stk, "address": bob, "amount": strconv.Itoa(drBobJoin)})
	s.stakers = []ibc.Wallet{s.alice, s.bob}
	s.requireSameDay(t, s.window, "alice's and bob's joins")
	require.Equal(t, strconv.Itoa(drAliceStake+drBobJoin), s.denomReward(t).staked().String())

	day := s.waitDayTick(t)
	require.Equal(t, s.window+1, int64(day.CurrentEpoch))
	s.requireSchedule(t, drTotalDays, 1)
	// 1,200,000 over 4,000,000 staked: 0.3 per unit
	s.requirePending(t, s.alice, coins(900_000, s.prz))
	s.requirePending(t, s.bob, coins(300_000, s.prz))
	s.requireEscrow(t)

	assert.Len(t, s.distributionEvents(t, day, s.prz), 1, "the daily payout must emit DenomRewardDistributionEvent in EndBlock")
}

func (s *denomRewardsSuite) topUp(t *testing.T) {
	bob := s.bob.FormattedAddress()
	przBefore := s.balance(t, bob, s.prz)

	res := s.mustExec(t, s.bob, "rewards", "join-denom-reward", s.stk, strconv.Itoa(drBobTopUp))
	s.requireTxEvent(t, res, "DenomRewardJoinEvent", map[string]string{"denom": s.stk, "address": bob, "amount": strconv.Itoa(drBobTopUp)})
	s.requireSameDay(t, s.window+1, "bob's top-up")

	amount, pending := s.participant(t, s.bob)
	require.Equal(t, strconv.Itoa(drBobJoin+drBobTopUp), amount.String())
	s.bobSettled = s.balance(t, bob, s.prz).Sub(przBefore)
	require.Equal(t, "300000", s.bobSettled.Add(pending.AmountOf(s.prz)).String(), "the top-up must keep bob's accrued 300,000")
	require.Equal(t, strconv.Itoa(drAliceStake+drBobJoin+drBobTopUp), s.denomReward(t).staked().String())

	day := s.waitDayTick(t)
	require.Equal(t, s.window+2, int64(day.CurrentEpoch))
	s.requireSchedule(t, drTotalDays, 2)
	// 1,200,000 over 6,000,000 staked: 0.2 per unit, 600,000 each
	s.requirePending(t, s.alice, coins(1_500_000, s.prz))
	_, pending = s.participant(t, s.bob)
	require.Equal(t, "900000", s.bobSettled.Add(pending.AmountOf(s.prz)).String())
	s.requireEscrow(t)

	assert.Len(t, s.distributionEvents(t, day, s.prz), 1, "the daily payout must emit DenomRewardDistributionEvent in EndBlock")
}

func (s *denomRewardsSuite) airdropNewPrize(t *testing.T) {
	creator := s.creator.FormattedAddress()
	ubzeBefore := s.balance(t, creator, chain.Denom)
	escrowBefore := s.balance(t, s.module, s.prz2)

	res := s.mustExec(t, s.creator, "rewards", "distribute-denom-rewards", s.stk, s.prz2, strconv.Itoa(drAirdrop))
	s.requireTxEvent(t, res, "DenomRewardPrizeCreateEvent", map[string]string{"denom": s.stk, "prize_denom": s.prz2})
	s.requireTxEvent(t, res, "DenomRewardDistributionEvent", map[string]string{
		"denom": s.stk, "prize_denom": s.prz2, "amount": strconv.Itoa(drAirdrop),
	})
	require.Equal(t, ubzeBefore.Sub(s.params.CreateDenomRewardPrizeFee.Amount).Sub(s.txFee(t, res)).String(),
		s.balance(t, creator, chain.Denom).String(), "a new prize denom costs exactly the prize fee")
	require.Equal(t, escrowBefore.AddRaw(drAirdrop).String(), s.balance(t, s.module, s.prz2).String())

	// credited immediately, 3,000,000 each of 6,000,000 staked
	_, alicePending := s.participant(t, s.alice)
	require.Equal(t, "300000", alicePending.AmountOf(s.prz2).String())
	_, bobPending := s.participant(t, s.bob)
	require.Equal(t, "300000", bobPending.AmountOf(s.prz2).String())
}

func (s *denomRewardsSuite) multiPrizeClaim(t *testing.T) {
	alice := s.alice.FormattedAddress()
	przBefore, prz2Before := s.balance(t, alice, s.prz), s.balance(t, alice, s.prz2)

	res := s.mustExec(t, s.alice, "rewards", "claim-denom-rewards", s.stk)
	s.requireSameDay(t, s.window+2, "the airdrop and alice's claim")
	claim := s.requireTxEvent(t, res, "DenomRewardClaimEvent", map[string]string{"denom": s.stk, "address": alice})
	want := coins(1_500_000, s.prz).Add(coins(300_000, s.prz2)...)
	paid, err := sdk.ParseCoinsNormalized(claim.Attributes["amounts"])
	require.NoError(t, err)
	require.Equal(t, want.String(), paid.String())

	require.Equal(t, przBefore.AddRaw(1_500_000).String(), s.balance(t, alice, s.prz).String())
	require.Equal(t, prz2Before.AddRaw(300_000).String(), s.balance(t, alice, s.prz2).String())
	s.requirePending(t, s.alice, sdk.NewCoins())
	s.requireEscrow(t)
}

func (s *denomRewardsSuite) scheduleFinish(t *testing.T) {
	day := s.waitDayTick(t)
	require.Equal(t, s.window+3, int64(day.CurrentEpoch))
	s.requireSchedule(t, drTotalDays, 3)
	assert.Len(t, s.distributionEvents(t, day, s.prz), 1, "the daily payout must emit DenomRewardDistributionEvent in EndBlock")

	day = s.waitDayTick(t)
	require.Equal(t, s.window+4, int64(day.CurrentEpoch))
	require.Empty(t, s.schedules(t), "the schedule must be deleted by its last payout")
	finish, err := helpers.BlockEvents(s.ctx, s.bze, int64(day.CurrentEpochStartHeight), drEventPrefix+"DenomRewardScheduleFinishEvent")
	require.NoError(t, err)
	require.Len(t, helpers.EventsWith(finish, map[string]string{"schedule_id": s.scheduleID}), 1,
		"no DenomRewardScheduleFinishEvent for schedule %s at height %d: %+v", s.scheduleID, day.CurrentEpochStartHeight, finish)
	assert.Len(t, s.distributionEvents(t, day, s.prz), 1, "the daily payout must emit DenomRewardDistributionEvent in EndBlock")

	// alice claimed after the 2nd payout: 600,000 × 2 since; bob: 600,000 × 3
	s.requirePending(t, s.alice, coins(1_200_000, s.prz))
	s.requirePending(t, s.bob, coins(1_800_000, s.prz).Add(coins(300_000, s.prz2)...))
	s.requireEscrow(t)
}

func (s *denomRewardsSuite) afterFinish(t *testing.T) {
	day := s.waitDayTick(t)
	require.Equal(t, s.window+5, int64(day.CurrentEpoch))
	require.Empty(t, s.distributionEvents(t, day, s.prz), "a finished schedule must not distribute")
	s.requirePending(t, s.alice, coins(1_200_000, s.prz))
	s.requirePending(t, s.bob, coins(1_800_000, s.prz).Add(coins(300_000, s.prz2)...))

	for _, claim := range []struct {
		user ibc.Wallet
		want sdk.Coins
	}{
		{s.bob, coins(1_800_000, s.prz).Add(coins(300_000, s.prz2)...)},
		{s.alice, coins(1_200_000, s.prz)},
	} {
		res := s.mustExec(t, claim.user, "rewards", "claim-denom-rewards", s.stk)
		ev := s.requireTxEvent(t, res, "DenomRewardClaimEvent", map[string]string{"denom": s.stk, "address": claim.user.FormattedAddress()})
		paid, err := sdk.ParseCoinsNormalized(ev.Attributes["amounts"])
		require.NoError(t, err)
		require.Equal(t, claim.want.String(), paid.String())
		s.requirePending(t, claim.user, sdk.NewCoins())
	}

	// across every claim and settlement the stakers received the whole budget
	alice, bob := s.alice.FormattedAddress(), s.bob.FormattedAddress()
	require.Equal(t, strconv.Itoa(drDailyAmount*drTotalDays),
		s.balance(t, alice, s.prz).Add(s.balance(t, bob, s.prz)).String())
	require.Equal(t, strconv.Itoa(drAirdrop),
		s.balance(t, alice, s.prz2).Add(s.balance(t, bob, s.prz2)).String())
}

func (s *denomRewardsSuite) unlockReturnsStake(t *testing.T) {
	ctx, cancel := context.WithTimeout(s.ctx, unlockTimeout)
	defer cancel()
	// AfterEpochEnd(h) runs when the hour epoch moves to h+1; the unlock queue
	// it fills is drained in that block's EndBlock.
	require.NoError(t, helpers.WaitForEpoch(ctx, s.bze, "hour", s.carolUnlock+1))

	require.Equal(t, strconv.Itoa(drCarolStake), s.balance(t, s.carol.FormattedAddress(), s.stk).String())
	_, found := s.pendingUnlock(t, s.carolUnlockIndex())
	require.False(t, found, "the pending unlock must be removed once paid")

	s.requireRejected(t, "not a participant", s.carol, "rewards", "exit-denom-reward", s.stk)
}

func (s *denomRewardsSuite) escrowAtEnd(t *testing.T) {
	staked := sdkmath.NewInt(drAliceStake + drBobJoin + drBobTopUp)
	require.Equal(t, staked.String(), s.denomReward(t).staked().String())
	require.Equal(t, s.baseline[s.prz].String(), s.balance(t, s.module, s.prz).String(), "prz escrow must be fully paid out")
	require.Equal(t, s.baseline[s.prz2].String(), s.balance(t, s.module, s.prz2).String(), "prz2 escrow must be fully paid out")
	require.Equal(t, s.baseline[s.stk].Add(staked).String(), s.balance(t, s.module, s.stk).String(), "the module must hold exactly the staked amount")
	s.requireEscrow(t)
}

// requireEscrow asserts the rewards module holds, per prize denom, exactly the
// unpaid schedule budget plus every staker's pending amount (on top of its
// baseline). With exact splits there is no dust, so equality holds.
func (s *denomRewardsSuite) requireEscrow(t *testing.T) {
	t.Helper()
	owed := map[string]sdkmath.Int{s.prz: sdkmath.ZeroInt(), s.prz2: sdkmath.ZeroInt()}
	for _, sc := range s.schedules(t) {
		unpaid := sdkmath.NewInt(int64(sc.Duration) - int64(sc.Payouts))
		owed[sc.PrizeDenom] = owed[sc.PrizeDenom].Add(sc.daily(t).Mul(unpaid))
	}
	for _, staker := range s.stakers {
		_, pending := s.participant(t, staker)
		for denom := range owed {
			owed[denom] = owed[denom].Add(pending.AmountOf(denom))
		}
	}
	for denom, amount := range owed {
		require.Equal(t, s.baseline[denom].Add(amount).String(), s.balance(t, s.module, denom).String(),
			"rewards module %s balance must equal unpaid budget + pending", denom)
	}
}

func (s *denomRewardsSuite) mustExec(t *testing.T, user ibc.Wallet, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTx(s.ctx, s.bze, user, args...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", args, res.RawLog)
	return res
}

func (s *denomRewardsSuite) requireRejected(t *testing.T, reason string, user ibc.Wallet, args ...string) {
	t.Helper()
	res, err := helpers.ExecTx(s.ctx, s.bze, user, args...)
	msg, err := helpers.RejectionReason(res, err)
	require.NoError(t, err, "tx %v", args)
	require.Contains(t, msg, reason)
}

func (s *denomRewardsSuite) requireTxEvent(t *testing.T, res helpers.TxResult, name string, attrs map[string]string) helpers.Event {
	t.Helper()
	events := helpers.EventsWith(helpers.TxEvents(res, drEventPrefix+name), attrs)
	require.Len(t, events, 1, "want one %s with %v in tx %s, got %+v", name, attrs, res.TxHash, helpers.TxEvents(res, drEventPrefix+name))
	return events[0]
}

func (s *denomRewardsSuite) txFee(t *testing.T, res helpers.TxResult) sdkmath.Int {
	t.Helper()
	fee, err := helpers.TxFee(res, chain.Denom)
	require.NoError(t, err)
	return fee
}

// waitDayTick waits for the next day epoch and returns it. The tick's block
// already holds the payouts: the epoch hook queues them and the rewards
// EndBlock pays them in the same block.
func (s *denomRewardsSuite) waitDayTick(t *testing.T) helpers.EpochInfo {
	t.Helper()
	cur, err := helpers.CurrentEpoch(s.ctx, s.bze, "day")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(s.ctx, dayTickTimeout)
	defer cancel()
	require.NoError(t, helpers.WaitForEpoch(ctx, s.bze, "day", cur+1))
	day, err := helpers.Epoch(s.ctx, s.bze, "day")
	require.NoError(t, err)
	return day
}

// requireSameDay fails when a day tick fell inside a window the scenario needs
// tick-free (a payout would then split differently than asserted).
func (s *denomRewardsSuite) requireSameDay(t *testing.T, want int64, what string) {
	t.Helper()
	cur, err := helpers.CurrentEpoch(s.ctx, s.bze, "day")
	require.NoError(t, err)
	require.Equal(t, want, cur, "a day tick fell during %s; the scenario needs them in one day epoch", what)
}

func (s *denomRewardsSuite) distributionEvents(t *testing.T, day helpers.EpochInfo, prize string) []helpers.Event {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, int64(day.CurrentEpochStartHeight), drEventPrefix+"DenomRewardDistributionEvent")
	require.NoError(t, err)
	return helpers.EventsWith(events, map[string]string{"denom": s.stk, "prize_denom": prize})
}

func (s *denomRewardsSuite) balance(t *testing.T, addr, denom string) sdkmath.Int {
	t.Helper()
	b, err := helpers.Balance(s.ctx, s.bze, addr, denom)
	require.NoError(t, err)
	return b
}

type denomRewardInfo struct {
	StakingDenom string        `json:"staking_denom"`
	Lock         helpers.Int64 `json:"lock"`
	MinStake     helpers.Int64 `json:"min_stake"`
	StakedAmount string        `json:"staked_amount"`
}

func (d denomRewardInfo) staked() sdkmath.Int {
	return parseInt(d.StakedAmount)
}

func (s *denomRewardsSuite) denomReward(t *testing.T) denomRewardInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		DenomReward denomRewardInfo `json:"denom_reward"`
	}](s.ctx, s.bze, "rewards", "denom-reward", s.stk)
	require.NoError(t, err)
	return res.DenomReward
}

type scheduleInfo struct {
	ScheduleID  string        `json:"schedule_id"`
	PrizeDenom  string        `json:"prize_denom"`
	DailyAmount string        `json:"daily_amount"`
	Duration    helpers.Int64 `json:"duration"`
	Payouts     helpers.Int64 `json:"payouts"`
}

func (sc scheduleInfo) daily(t *testing.T) sdkmath.Int {
	t.Helper()
	v, ok := sdkmath.NewIntFromString(sc.DailyAmount)
	require.True(t, ok, "daily_amount %q", sc.DailyAmount)
	return v
}

func (s *denomRewardsSuite) schedules(t *testing.T) []scheduleInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		List []scheduleInfo `json:"list"`
	}](s.ctx, s.bze, "rewards", "denom-reward-schedules", s.stk)
	require.NoError(t, err)
	return res.List
}

func (s *denomRewardsSuite) requireSchedule(t *testing.T, duration, payouts int64) {
	t.Helper()
	list := s.schedules(t)
	require.Len(t, list, 1)
	sc := list[0]
	require.Equal(t, s.scheduleID, sc.ScheduleID)
	require.Equal(t, s.prz, sc.PrizeDenom)
	require.Equal(t, strconv.Itoa(drDailyAmount), sc.DailyAmount)
	require.Equal(t, duration, int64(sc.Duration), "duration")
	require.Equal(t, payouts, int64(sc.Payouts), "payouts")
}

func (s *denomRewardsSuite) participant(t *testing.T, user ibc.Wallet) (sdkmath.Int, sdk.Coins) {
	t.Helper()
	res, err := helpers.Query[struct {
		Participant struct {
			Amount string `json:"amount"`
		} `json:"participant"`
		Pending sdk.Coins `json:"pending"`
	}](s.ctx, s.bze, "rewards", "denom-reward-participant", user.FormattedAddress(), s.stk)
	require.NoError(t, err)
	return parseInt(res.Participant.Amount), res.Pending
}

func (s *denomRewardsSuite) requirePending(t *testing.T, user ibc.Wallet, want sdk.Coins) {
	t.Helper()
	_, pending := s.participant(t, user)
	require.Equal(t, want.String(), sdk.NewCoins(pending...).String(), "pending of %s", user.KeyName())
}

type pendingUnlockInfo struct {
	Index   string `json:"index"`
	Address string `json:"address"`
	Amount  string `json:"amount"`
	Denom   string `json:"denom"`
}

func (s *denomRewardsSuite) carolUnlockIndex() string {
	return fmt.Sprintf("%d/dr/%s/%s", s.carolUnlock, s.stk, s.carol.FormattedAddress())
}

func (s *denomRewardsSuite) pendingUnlock(t *testing.T, index string) (pendingUnlockInfo, bool) {
	t.Helper()
	res, err := helpers.Query[struct {
		List []pendingUnlockInfo `json:"list"`
	}](s.ctx, s.bze, "rewards", "all-pending-unlock-participants")
	require.NoError(t, err)
	for _, e := range res.List {
		if e.Index == index {
			return e, true
		}
	}
	return pendingUnlockInfo{}, false
}

// rewardsParams holds the rewards params the suite asserts against.
type rewardsParams struct {
	CreateDenomRewardFee      sdk.Coin
	CreateDenomRewardPrizeFee sdk.Coin
	AddDenomRewardScheduleFee sdk.Coin
	ExtraGasForDenomExit      helpers.Int64
	DenomRewardLock           helpers.Int64
	DenomRewardMinStake       helpers.Int64
}

// queryParams reads `bzed q rewards params`. The params proto fields are
// camelCase while their JSON tags are snake_case, and which one the CLI
// prints depends on the encoder, so both spellings are accepted.
func (s *denomRewardsSuite) queryParams(t *testing.T) rewardsParams {
	t.Helper()
	res, err := helpers.Query[struct {
		Params map[string]json.RawMessage `json:"params"`
	}](s.ctx, s.bze, "rewards", "params")
	require.NoError(t, err)

	field := func(dst any, camel, snake string) {
		raw, ok := res.Params[camel]
		if !ok {
			raw, ok = res.Params[snake]
		}
		if !ok {
			return // zero values are omitted
		}
		require.NoError(t, json.Unmarshal(raw, dst), "param %s", camel)
	}
	var p rewardsParams
	field(&p.CreateDenomRewardFee, "createDenomRewardFee", "create_denom_reward_fee")
	field(&p.CreateDenomRewardPrizeFee, "createDenomRewardPrizeFee", "create_denom_reward_prize_fee")
	field(&p.AddDenomRewardScheduleFee, "addDenomRewardScheduleFee", "add_denom_reward_schedule_fee")
	field(&p.ExtraGasForDenomExit, "extraGasForDenomExit", "extra_gas_for_denom_exit")
	field(&p.DenomRewardLock, "denomRewardLock", "denom_reward_lock")
	field(&p.DenomRewardMinStake, "denomRewardMinStake", "denom_reward_min_stake")

	for name, fee := range map[string]sdk.Coin{
		"create_denom_reward_fee":       p.CreateDenomRewardFee,
		"create_denom_reward_prize_fee": p.CreateDenomRewardPrizeFee,
		"add_denom_reward_schedule_fee": p.AddDenomRewardScheduleFee,
	} {
		require.False(t, fee.Amount.IsNil(), "param %s missing from %v", name, res.Params)
	}
	require.Positive(t, int64(p.ExtraGasForDenomExit), "param extra_gas_for_denom_exit missing from %v", res.Params)
	return p
}

func coins(amount int64, denom string) sdk.Coins {
	return sdk.NewCoins(sdk.NewInt64Coin(denom, amount))
}

func coinStr(amount int64, denom string) string {
	return sdk.NewInt64Coin(denom, amount).String()
}

// parseInt reads a math.Int the CLI printed; an omitted zero reads as "".
func parseInt(v string) sdkmath.Int {
	if v == "" {
		return sdkmath.ZeroInt()
	}
	i, ok := sdkmath.NewIntFromString(v)
	if !ok {
		panic(fmt.Sprintf("not an integer: %q", v))
	}
	return i
}
