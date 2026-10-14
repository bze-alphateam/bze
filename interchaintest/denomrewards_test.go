package interchaintest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
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
	// 600,000 BZE: six denoms, two DRs, four prizes, four schedules (25,000 BZE each)
	drCreatorFunds = 600_000_000_000
	drUserFunds    = 1_000_000_000 // 1,000 BZE for gas

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
	drCpCollector   = "txfeecollector_cp"
	drCodespace     = "rewards"
	// errors built with fmt.Errorf ("user balance is too low") carry no ABCI code
	drUndefinedSpace = "undefined"

	// The second DR (stk2) is about rounding, so its amounts are tiny and its
	// airdrop splits deliberately inexact: 10 over 7 staked, 2 and 3 over 21.
	dr2DaveJoin   = 3
	dr2DaveTopUp  = 7
	dr2ErinStake  = 4
	dr2FrankStake = 7
	dr2Staked     = dr2DaveJoin + dr2DaveTopUp + dr2ErinStake + dr2FrankStake // 21
	dr2Air1       = 10
	dr2Air2       = 2
	dr2Air3       = 3
	// schedules of step 14: 2,100 and 420 a day in pa, 210 in pb, all exact over 21
	dr2PaDaily     = 2_100
	dr2PaDays      = 2
	dr2PaDaily2    = 420
	dr2PbDaily     = 210
	dr2StkSupply   = dr2Staked + dr2ErinStake // erin re-joins while her first stake is locked
	dr2PaSupply    = 20_000_000               // covers the 36,500-day extension refused in step 14
	dr2PbSupply    = 1_000
	dr2MaxDuration = 36_500 // types.HundredYearsInDays

	// drTxGas is the fixed gas of the refused messages: an exit charges
	// extra_gas_for_denom_exit (1,000,000) before anything else.
	drTxGas = 2_000_000
	// drBatchGasPerMsg is the gas budgeted per message of the prize-cap batches.
	drBatchGasPerMsg = 500_000
	// drCapBudgetPerDenom funds the prize-cap user per denom it creates: the
	// create-denom fee plus the prize fee (25,000 BZE each) with a margin.
	drCapBudgetPerDenom = 100_000_000_000
	// drCapAffordable bounds the cap the suite is willing to fill.
	drCapAffordable = 100

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
//
// Steps 12-18 run after the first DR is finished. Step 13 drives a second DR
// (stk2: dave, erin, frank) with airdrops only, so no day tick touches its
// accumulator; from step 14 on it has schedules (W = the day its schedules are
// created on):
//
//	W    three schedules created (14)
//	W+1  first payout; erin exits and re-joins (15)
//	W+2  last payout; prize cap (16), refusals (17), erin's unlock (18)
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
	t.Run("12 re-join after the unlock starts a fresh position", s.rejoinAfterUnlock)
	t.Run("13 fresh join, top-up dust and dust-only claim", s.dustAndFreshJoin)
	t.Run("14 schedules on a new and on existing prize denoms", s.schedulesCoexist)
	t.Run("15 exit pays every prize, re-join while the unlock is pending", s.exitAndRejoin)
	t.Run("16 prize denom cap", s.prizeDenomCap)
	// the unlock runs before the refusals: an unlock paid while a refusal is
	// checked would read as a state change
	t.Run("17 the pending unlock pays beside the new position", s.unlockBesideNewPosition)
	t.Run("18 refusals leave the state unchanged", s.refusals)
}

type denomRewardsSuite struct {
	ctx context.Context
	bze *cosmos.CosmosChain

	creator, alice, bob, carol ibc.Wallet
	dave, erin, frank          ibc.Wallet
	stk, prz, prz2             string
	// the second DR's staking denom and prize denoms, and the prize-cap DR's
	stk2, pa, pb, capStk string
	module, cpCollector  string
	params               rewardsParams

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

	// dr2Window is the day epoch the second DR's schedules are created in (W)
	dr2Window int64
	// dr2Schedules are the ids of the second DR's schedules: pa, pa, pb
	dr2Schedules [3]string
	// erinUnlock is the hour epoch erin's first stk2 stake unlocks at
	erinUnlock int64
}

func setupDenomRewards(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *denomRewardsSuite {
	s := &denomRewardsSuite{ctx: ctx, bze: bze, baseline: map[string]sdkmath.Int{}}

	s.creator = interchaintest.GetAndFundTestUsers(t, ctx, "drcreator", sdkmath.NewInt(drCreatorFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "dr", sdkmath.NewInt(drUserFunds), bze, bze, bze, bze, bze, bze)
	s.alice, s.bob, s.carol = users[0], users[1], users[2]
	s.dave, s.erin, s.frank = users[3], users[4], users[5]
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
	s.cpCollector, err = helpers.ModuleAddress(ctx, bze, drCpCollector)
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
	require.Equal(t, fee.Amount.String(), s.cpFee(t, res).String(), "the create fee must reach the txfeecollector_cp module")

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
	require.Equal(t, fees.String(), s.cpFee(t, res).String(), "the prize and schedule fees must reach the txfeecollector_cp module")
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
	require.True(t, s.cpFee(t, res).IsZero(), "extending a schedule sends nothing to txfeecollector_cp")
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

	s.carolUnlock = s.unlockEpoch(t, res.Height)

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
	require.Equal(t, s.params.CreateDenomRewardPrizeFee.Amount.String(), s.cpFee(t, res).String(), "the prize fee must reach the txfeecollector_cp module")
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

// rejoinAfterUnlock: carol, whose stake came back in 10b, joins again. The
// accumulators of both prize denoms are above zero, yet her fresh position is
// stamped at them and starts with nothing pending.
func (s *denomRewardsSuite) rejoinAfterUnlock(t *testing.T) {
	carol := s.carol.FormattedAddress()
	for _, prize := range []string{s.prz, s.prz2} {
		require.Positive(t, s.accumulator(t, s.stk, prize).Sign(), "the %s accumulator must be above zero", prize)
	}
	require.Empty(t, s.positions(t, s.carol), "carol has no position before re-joining")

	res := s.mustExec(t, s.carol, "rewards", "join-denom-reward", s.stk, strconv.Itoa(drCarolStake))
	s.requireTxEvent(t, res, "DenomRewardJoinEvent", map[string]string{"denom": s.stk, "address": carol, "amount": strconv.Itoa(drCarolStake)})
	amount, pending := s.participant(t, s.carol)
	require.Equal(t, strconv.Itoa(drCarolStake), amount.String())
	require.Empty(t, pending, "a re-join must not inherit the accumulators")
	require.Equal(t, "0", s.balance(t, carol, s.stk).String())
	require.Equal(t, strconv.Itoa(drAliceStake+drBobJoin+drBobTopUp+drCarolStake), s.denomReward(t).staked().String())
}

// dustAndFreshJoin creates the second DR and drives it with airdrops only (no
// day tick changes its accumulator). Accumulator values are truncated to 18
// decimals; every pending below is floor(amount × (S − index)).
func (s *denomRewardsSuite) dustAndFreshJoin(t *testing.T) {
	creator := s.creator.FormattedAddress()
	factory := "factory/" + creator + "/"
	s.stk2, s.pa, s.pb = factory+"stk2", factory+"pa", factory+"pb"
	for _, d := range []struct {
		sub    string
		supply int64
	}{{"stk2", dr2StkSupply}, {"pa", dr2PaSupply}, {"pb", dr2PbSupply}} {
		s.mustExec(t, s.creator, "tokenfactory", "create-denom", d.sub)
		s.mustExec(t, s.creator, "tokenfactory", "mint", coinStr(d.supply, factory+d.sub))
	}
	for _, send := range []struct {
		to     ibc.Wallet
		amount int64
	}{
		{s.dave, dr2DaveJoin + dr2DaveTopUp},
		{s.erin, 2 * dr2ErinStake},
		{s.frank, dr2FrankStake},
	} {
		s.mustExec(t, s.creator, "bank", "send", creator, send.to.FormattedAddress(), coinStr(send.amount, s.stk2))
	}
	for _, denom := range []string{s.stk2, s.pa, s.pb} {
		s.baseline[denom] = s.balance(t, s.module, denom)
	}

	ubzeBefore := s.balance(t, creator, chain.Denom)
	res := s.mustExec(t, s.creator, "rewards", "create-denom-reward", s.stk2)
	s.requireTxEvent(t, res, "DenomRewardCreateEvent", map[string]string{"denom": s.stk2})
	fee := s.params.CreateDenomRewardFee.Amount
	require.Equal(t, fee.String(), s.cpFee(t, res).String(), "the create fee must reach the txfeecollector_cp module")
	require.Equal(t, ubzeBefore.Sub(fee).Sub(s.txFee(t, res)).String(), s.balance(t, creator, chain.Denom).String())

	s.join2(t, s.dave, dr2DaveJoin)
	s.join2(t, s.erin, dr2ErinStake)

	// 10 over 7 staked: S = 1.428571428571428571
	s.airdrop2(t, s.pb, dr2Air1, true)
	sPb := drQuo(dr2Air1, dr2DaveJoin+dr2ErinStake)
	s.requireAccumulator(t, s.stk2, s.pb, sPb)
	s.requirePendingIn(t, s.dave, s.stk2, coins(4, s.pb)) // 3 × 1.428… = 4.28
	s.requirePendingIn(t, s.erin, s.stk2, coins(5, s.pb)) // 4 × 1.428… = 5.71

	// frank joins with S > 0: his index is stamped at S, he inherits nothing
	s.join2(t, s.frank, dr2FrankStake)
	s.requirePendingIn(t, s.frank, s.stk2, sdk.NewCoins())
	s.requirePendingIn(t, s.dave, s.stk2, coins(4, s.pb))

	// dave's top-up pays floor(4.28…) = 4 and restamps his index at S: the 2/7
	// left over is forfeited
	dave := s.dave.FormattedAddress()
	pbBefore := s.balance(t, dave, s.pb)
	res = s.join2(t, s.dave, dr2DaveTopUp)
	require.Equal(t, "4", s.transferred(t, res, s.module, dave, s.pb).String(), "the top-up must pay the whole units pending")
	require.Equal(t, pbBefore.AddRaw(4).String(), s.balance(t, dave, s.pb).String())
	amount, pending := s.participantIn(t, s.dave, s.stk2)
	require.Equal(t, strconv.Itoa(dr2DaveJoin+dr2DaveTopUp), amount.String())
	require.Empty(t, pending)
	require.Equal(t, strconv.Itoa(dr2Staked), s.denomRewardOf(t, s.stk2).staked().String())

	// nothing pending: the claim is refused
	s.requireRefusedMsg(t, s.dave, drMsg("MsgClaimDenomRewards", s.dave, s.stk2, nil), drCodespace, 5012, "no rewards available to claim", false)

	// 2 over 21: dave 10 × 0.0952… = 0.95 is dust only (with the forfeited 2/7
	// kept he would read 1); erin 4 × 1.5238… = 6.09; frank 7 × 0.0952… = 0.67
	s.airdrop2(t, s.pb, dr2Air2, false)
	sPb.Add(sPb, drQuo(dr2Air2, dr2Staked))
	s.requireAccumulator(t, s.stk2, s.pb, sPb)
	s.requirePendingIn(t, s.dave, s.stk2, sdk.NewCoins())
	s.requirePendingIn(t, s.erin, s.stk2, coins(6, s.pb))
	s.requirePendingIn(t, s.frank, s.stk2, sdk.NewCoins())

	// a dust-only claim pays nothing, so it is refused, and dave's index stays
	// where his top-up put it
	s.requireRefusedMsg(t, s.dave, drMsg("MsgClaimDenomRewards", s.dave, s.stk2, nil), drCodespace, 5012, "no rewards available to claim", false)

	// 3 over 21: dave 10 × (S − his index) = 10 × 0.238… = 2.38. Had the dust
	// claim moved his index he would read 10 × 0.1428… = 1.43. frank, 7 ×
	// 0.238… = 1.67, gets only his share of the airdrops since he joined.
	s.airdrop2(t, s.pb, dr2Air3, false)
	sPb.Add(sPb, drQuo(dr2Air3, dr2Staked))
	s.requireAccumulator(t, s.stk2, s.pb, sPb)
	s.requirePendingIn(t, s.dave, s.stk2, coins(2, s.pb))
	s.requirePendingIn(t, s.erin, s.stk2, coins(6, s.pb)) // 4 × 1.666… = 6.67
	s.requirePendingIn(t, s.frank, s.stk2, coins(1, s.pb))

	res = s.mustExec(t, s.dave, "rewards", "claim-denom-rewards", s.stk2)
	claim := s.requireTxEvent(t, res, "DenomRewardClaimEvent", map[string]string{"denom": s.stk2, "address": dave})
	paid, err := sdk.ParseCoinsNormalized(claim.Attributes["amounts"])
	require.NoError(t, err)
	require.Equal(t, coins(2, s.pb).String(), paid.String())
	require.Equal(t, pbBefore.AddRaw(4+2).String(), s.balance(t, dave, s.pb).String())
	s.requirePendingIn(t, s.dave, s.stk2, sdk.NewCoins())

	// 15 airdropped, 6 paid to dave: erin's 6 and frank's 1 are pending and 2
	// units of truncation dust stay in escrow
	require.Equal(t, s.baseline[s.pb].AddRaw(dr2Air1+dr2Air2+dr2Air3-6).String(), s.balance(t, s.module, s.pb).String())
}

// schedulesCoexist attaches three schedules to the second DR in one day
// epoch: pa (a new prize denom), pa again and pb (existing prize denoms: the
// schedule fee only and no new prize slot). The next tick pays each one its
// own daily amount.
func (s *denomRewardsSuite) schedulesCoexist(t *testing.T) {
	// a fresh day epoch leaves the whole epoch for the creations
	s.dr2Window = int64(s.waitDayTick(t).CurrentEpoch)
	s.dr2Schedules[0] = s.schedule2(t, s.pa, dr2PaDaily2, 1, true)

	// extending the 1-day schedule by 36,500 days would make it 36,501: the
	// extra budget passes the balance check and the bound refuses it. Done
	// first in the epoch, so no tick lands between the two state snapshots.
	s.requireRefusedMsg(t, s.creator, drMsg("MsgUpdateDenomRewardSchedule", s.creator, s.stk2, map[string]any{
		"schedule_id": s.dr2Schedules[0],
		"duration":    strconv.Itoa(dr2MaxDuration),
	}), drCodespace, 5005, fmt.Sprintf("exceeds the maximum allowed of %d days", dr2MaxDuration), false)

	s.dr2Schedules[1] = s.schedule2(t, s.pa, dr2PaDaily, dr2PaDays, false)
	s.dr2Schedules[2] = s.schedule2(t, s.pb, dr2PbDaily, 1, false)
	s.requireSameDay(t, s.dr2Window, "the three schedules")
	require.Len(t, s.prizesOf(t, s.stk2), 2, "pb and pa are the only prize denoms")
	require.Len(t, s.schedulesOf(t, s.stk2), 3)

	sPa, sPb := s.accumulator(t, s.stk2, s.pa), s.accumulator(t, s.stk2, s.pb)
	require.Zero(t, sPa.Sign(), "pa is not distributed before the tick")
	day := s.waitDayTick(t)
	require.Equal(t, s.dr2Window+1, int64(day.CurrentEpoch))
	require.ElementsMatch(t, []string{strconv.Itoa(dr2PaDaily2), strconv.Itoa(dr2PaDaily)}, s.distributedAmounts(t, day, s.stk2, s.pa),
		"each pa schedule pays its own daily amount")
	require.ElementsMatch(t, []string{strconv.Itoa(dr2PbDaily)}, s.distributedAmounts(t, day, s.stk2, s.pb))
	for i, finishes := range []int{1, 0, 1} {
		require.Len(t, s.finishEvents(t, day, s.dr2Schedules[i]), finishes, "finish events of schedule %s", s.dr2Schedules[i])
	}
	list := s.schedulesOf(t, s.stk2)
	require.Len(t, list, 1, "the 1-day schedules are deleted by their only payout")
	require.Equal(t, s.dr2Schedules[1], list[0].ScheduleID)
	require.Equal(t, int64(dr2PaDays), int64(list[0].Duration))
	require.Equal(t, int64(1), int64(list[0].Payouts))

	// 2,520 pa and 210 pb over 21 staked: +120 and +10, exactly
	s.requireAccumulator(t, s.stk2, s.pa, drUnits(120))
	s.requireAccumulator(t, s.stk2, s.pb, new(big.Int).Add(sPb, drUnits(10)))
	// pb: dave 10 × 10; erin 4 × 11.666… = 46.67; frank 7 × 10.238… = 71.67
	s.requirePendingIn(t, s.dave, s.stk2, coins(1_200, s.pa).Add(coins(100, s.pb)...))
	s.requirePendingIn(t, s.erin, s.stk2, coins(480, s.pa).Add(coins(46, s.pb)...))
	s.requirePendingIn(t, s.frank, s.stk2, coins(840, s.pa).Add(coins(71, s.pb)...))
}

// exitAndRejoin: erin exits with 480 pa and 46.67 pb pending, then joins again
// while her first stake is still locked.
func (s *denomRewardsSuite) exitAndRejoin(t *testing.T) {
	erin := s.erin.FormattedAddress()
	paBefore, pbBefore, stkBefore := s.balance(t, erin, s.pa), s.balance(t, erin, s.pb), s.balance(t, erin, s.stk2)

	res := s.mustExec(t, s.erin, "rewards", "exit-denom-reward", s.stk2)
	s.requireTxEvent(t, res, "DenomRewardExitEvent", map[string]string{"denom": s.stk2, "address": erin, "amount": strconv.Itoa(dr2ErinStake)})
	s.requireSameDay(t, s.dr2Window+1, "the payout and erin's exit")
	// both prizes are paid by the exit tx itself; the 0.67 pb dust is forfeited
	require.Equal(t, "480", s.transferred(t, res, s.module, erin, s.pa).String(), "the exit must pay the pending pa")
	require.Equal(t, "46", s.transferred(t, res, s.module, erin, s.pb).String(), "the exit must pay the pending pb")
	require.Equal(t, paBefore.AddRaw(480).String(), s.balance(t, erin, s.pa).String())
	require.Equal(t, pbBefore.AddRaw(46).String(), s.balance(t, erin, s.pb).String())
	require.Equal(t, stkBefore.String(), s.balance(t, erin, s.stk2).String(), "the stake stays locked until the unlock epoch")
	require.Empty(t, s.positions(t, s.erin), "the exit must remove the position")
	require.Equal(t, strconv.Itoa(dr2Staked-dr2ErinStake), s.denomRewardOf(t, s.stk2).staked().String())
	s.erinUnlock = s.unlockEpoch(t, res.Height)
	s.requireErinUnlock(t, true)

	// the code lets her re-join while the unlock is pending: a fresh position
	// stamped at the current accumulators, the pending unlock untouched
	s.join2(t, s.erin, dr2ErinStake)
	s.requireSameDay(t, s.dr2Window+1, "erin's exit and re-join")
	amount, pending := s.participantIn(t, s.erin, s.stk2)
	require.Equal(t, strconv.Itoa(dr2ErinStake), amount.String())
	require.Empty(t, pending, "a re-join must not inherit the accumulators")
	s.requireErinUnlock(t, true)
	require.Equal(t, "0", s.balance(t, erin, s.stk2).String())
	require.Equal(t, strconv.Itoa(dr2Staked), s.denomRewardOf(t, s.stk2).staked().String())

	// the last pa payout: 2,100 over 21, +100
	day := s.waitDayTick(t)
	require.Equal(t, s.dr2Window+2, int64(day.CurrentEpoch))
	require.Equal(t, []string{strconv.Itoa(dr2PaDaily)}, s.distributedAmounts(t, day, s.stk2, s.pa))
	require.Empty(t, s.distributedAmounts(t, day, s.stk2, s.pb))
	require.Len(t, s.finishEvents(t, day, s.dr2Schedules[1]), 1)
	require.Empty(t, s.schedulesOf(t, s.stk2))
	s.requireAccumulator(t, s.stk2, s.pa, drUnits(220))
	s.requirePendingIn(t, s.dave, s.stk2, coins(2_200, s.pa).Add(coins(100, s.pb)...))
	s.requirePendingIn(t, s.erin, s.stk2, coins(400, s.pa))
	s.requirePendingIn(t, s.frank, s.stk2, coins(1_540, s.pa).Add(coins(71, s.pb)...))

	// pa: 4,620 in, 480 paid to erin, the 4,140 left all pending (no dust)
	require.Equal(t, s.baseline[s.pa].AddRaw(dr2PaDaily2+dr2PaDaily*dr2PaDays-480).String(), s.balance(t, s.module, s.pa).String())
	// pb: 225 in, 52 paid (dave 4 + 2, erin 46): 171 pending, 2 units of dust
	require.Equal(t, s.baseline[s.pb].AddRaw(dr2Air1+dr2Air2+dr2Air3+dr2PbDaily-52).String(), s.balance(t, s.module, s.pb).String())
	require.Equal(t, s.baseline[s.stk2].AddRaw(dr2Staked+dr2ErinStake).String(), s.balance(t, s.module, s.stk2).String(),
		"the module holds the live stakes and erin's locked one")
}

// prizeDenomCap fills a third DR up to max_prize_denoms_per_dr prize denoms
// with batched airdrops: one more prize denom is refused, an airdrop in a
// prize denom it already has is still accepted and free.
func (s *denomRewardsSuite) prizeDenomCap(t *testing.T) {
	n := int(s.params.MaxPrizeDenomsPerDr)
	require.Positive(t, n, "max_prize_denoms_per_dr")
	if n > drCapAffordable {
		// every slot costs a create-denom fee and a prize fee; params_test.go
		// covers the cap lowered to 1
		t.Skipf("max_prize_denoms_per_dr %d is above the %d prize denoms the suite fills", n, drCapAffordable)
	}
	user := interchaintest.GetAndFundTestUsers(t, s.ctx, "drcap", sdkmath.NewInt(int64(n+2)*drCapBudgetPerDenom), s.bze)[0]
	require.NoError(t, testutil.WaitForBlocks(s.ctx, 2, s.bze))
	addr := user.FormattedAddress()

	// denoms[0] is staked, denoms[1..n] fill the cap, denoms[n+1] is one too many
	denoms := make([]string, n+2)
	var msgs []any
	for i := range denoms {
		sub := fmt.Sprintf("cap%02d", i)
		denoms[i] = "factory/" + addr + "/" + sub
		msgs = append(msgs,
			map[string]any{"@type": "/bze.tokenfactory.MsgCreateDenom", "creator": addr, "subdenom": sub},
			map[string]any{"@type": "/bze.tokenfactory.MsgMint", "creator": addr, "coins": coinStr(2, denoms[i])},
		)
	}
	s.mustExecMsgs(t, user, msgs...)
	s.capStk = denoms[0]

	msgs = []any{
		drMsg("MsgCreateDenomReward", user, s.capStk, nil),
		drMsg("MsgJoinDenomReward", user, s.capStk, map[string]any{"amount": "1"}),
	}
	for _, prize := range denoms[1 : n+1] {
		msgs = append(msgs, drAirdropMsg(user, s.capStk, prize, "1"))
	}
	res := s.mustExecMsgs(t, user, msgs...)
	require.Len(t, helpers.TxEvents(res, drEventPrefix+"DenomRewardPrizeCreateEvent"), n)
	fees := s.params.CreateDenomRewardFee.Amount.Add(s.params.CreateDenomRewardPrizeFee.Amount.MulRaw(int64(n)))
	require.Equal(t, fees.String(), s.cpFee(t, res).String(), "the create fee and %d prize fees must reach txfeecollector_cp", n)
	require.Len(t, s.prizesOf(t, s.capStk), n)

	s.requireRefusedMsg(t, user, drAirdropMsg(user, s.capStk, denoms[n+1], "1"), drCodespace, 5017, "prize denom cap reached", false)

	ubzeBefore := s.balance(t, addr, chain.Denom)
	res = s.mustExecMsgs(t, user, drAirdropMsg(user, s.capStk, denoms[1], "1"))
	s.requireTxEvent(t, res, "DenomRewardDistributionEvent", map[string]string{"denom": s.capStk, "prize_denom": denoms[1], "amount": "1"})
	require.Empty(t, helpers.TxEvents(res, drEventPrefix+"DenomRewardPrizeCreateEvent"))
	require.True(t, s.cpFee(t, res).IsZero(), "an existing prize denom pays no prize fee")
	require.Equal(t, ubzeBefore.Sub(s.txFee(t, res)).String(), s.balance(t, addr, chain.Denom).String())
	require.Len(t, s.prizesOf(t, s.capStk), n, "an existing prize denom takes no new slot")
}

// refusals: each message is refused with its exact code; a refusal in the
// msg server costs the tx fee only, one in ValidateBasic (CheckTx) nothing.
// No DR has a schedule any more, so no day tick changes the state meanwhile.
func (s *denomRewardsSuite) refusals(t *testing.T) {
	require.Empty(t, s.schedulesOf(t, s.stk2))
	require.Empty(t, s.schedules(t))
	creator := s.creator.FormattedAddress()
	ghost := "factory/" + creator + "/ghost" // a valid denom without supply
	over := s.balance(t, creator, s.pb).AddRaw(1).String()
	create := func(denom string) map[string]any { return drMsg("MsgCreateDenomReward", s.creator, denom, nil) }
	schedule := func(denom, prize, daily, duration string) map[string]any {
		return drMsg("MsgCreateDenomRewardSchedule", s.creator, denom, map[string]any{"prize_denom": prize, "daily_amount": daily, "duration": duration})
	}
	duration := fmt.Sprintf("duration should be between 1 and %d days", dr2MaxDuration)
	for _, c := range []struct {
		name      string
		user      ibc.Wallet
		msg       map[string]any
		codespace string
		code      uint32
		reason    string
		checkTx   bool
	}{
		{"create: denom without supply", s.creator, create(ghost), drCodespace, 5003, "invalid staking denom", false},
		{"create: invalid denom", s.creator, create("1bad"), drCodespace, 5003, "invalid staking denom", true},
		{"create: balance below the fee", s.frank, drMsg("MsgCreateDenomReward", s.frank, s.prz, nil), drUndefinedSpace, 1, "user balance is too low", false},

		{"schedule: no denom reward", s.creator, schedule(s.prz, s.pb, "1", "1"), drCodespace, 5015, "denom reward not found", false},
		{"schedule: prize denom without supply", s.creator, schedule(s.stk2, ghost, "1", "1"), drCodespace, 5002, "invalid prize denom", false},
		{"schedule: duration 0", s.creator, schedule(s.stk2, s.pb, "1", "0"), drCodespace, 5005, duration, true},
		{"schedule: duration above the max", s.creator, schedule(s.stk2, s.pb, "1", strconv.Itoa(dr2MaxDuration+1)), drCodespace, 5005, duration, true},
		{"schedule: daily amount 0", s.creator, schedule(s.stk2, s.pb, "0", "1"), drCodespace, 5000, "daily_amount should be greater than 0", true},
		{"schedule: negative daily amount", s.creator, schedule(s.stk2, s.pb, "-1", "1"), drCodespace, 5000, "daily_amount should be greater than 0", true},
		{"schedule: budget above the balance", s.creator, schedule(s.stk2, s.pb, over, "1"), drUndefinedSpace, 1, "user balance is too low", false},

		{"update: unknown schedule", s.creator, drMsg("MsgUpdateDenomRewardSchedule", s.creator, s.stk2, map[string]any{"schedule_id": "999999", "duration": "1"}),
			"sdk", 22, "denom reward schedule not found", false},

		{"airdrop: no denom reward", s.creator, drAirdropMsg(s.creator, s.prz, s.pb, "1"), drCodespace, 5015, "denom reward not found", false},
		{"airdrop: amount 0", s.creator, drAirdropMsg(s.creator, s.stk2, s.pb, "0"), drCodespace, 5000, "amount should be greater than 0", true},
		{"airdrop: amount above the balance", s.creator, drAirdropMsg(s.creator, s.stk2, s.pb, over), drUndefinedSpace, 1, "user balance is too low", false},

		// the DR's snapshotted min_stake is 0, so 0 is the only amount below it;
		// params_test.go raises the param by gov for a real minimum
		{"join: below the min stake", s.frank, drMsg("MsgJoinDenomReward", s.frank, s.stk2, map[string]any{"amount": "0"}), drCodespace, 5000, "amount should be greater than 0", true},

		{"claim: not a participant", s.creator, drMsg("MsgClaimDenomRewards", s.creator, s.stk2, nil), drCodespace, 5015, "you are not a participant", false},
		{"exit: not a participant", s.creator, drMsg("MsgExitDenomReward", s.creator, s.stk2, nil), drCodespace, 5015, "you are not a participant", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			s.requireRefusedMsg(t, c.user, c.msg, c.codespace, c.code, c.reason, c.checkTx)
		})
	}
}

// unlockBesideNewPosition: erin's first stake comes back at its hour epoch
// while her new position keeps its amount.
func (s *denomRewardsSuite) unlockBesideNewPosition(t *testing.T) {
	ctx, cancel := context.WithTimeout(s.ctx, unlockTimeout)
	defer cancel()
	require.NoError(t, helpers.WaitForEpoch(ctx, s.bze, "hour", s.erinUnlock+1))

	require.Equal(t, strconv.Itoa(dr2ErinStake), s.balance(t, s.erin.FormattedAddress(), s.stk2).String(), "the first stake must come back")
	s.requireErinUnlock(t, false)
	amount, _ := s.participantIn(t, s.erin, s.stk2)
	require.Equal(t, strconv.Itoa(dr2ErinStake), amount.String(), "the unlock must not touch the new position")
	require.Equal(t, strconv.Itoa(dr2Staked), s.denomRewardOf(t, s.stk2).staked().String())
	require.Equal(t, s.baseline[s.stk2].AddRaw(dr2Staked).String(), s.balance(t, s.module, s.stk2).String())
}

// join2 joins the second DR and asserts the join event.
func (s *denomRewardsSuite) join2(t *testing.T, user ibc.Wallet, amount int64) helpers.TxResult {
	t.Helper()
	res := s.mustExec(t, user, "rewards", "join-denom-reward", s.stk2, strconv.FormatInt(amount, 10))
	s.requireTxEvent(t, res, "DenomRewardJoinEvent", map[string]string{
		"denom": s.stk2, "address": user.FormattedAddress(), "amount": strconv.FormatInt(amount, 10),
	})
	return res
}

// airdrop2 airdrops from the creator into the second DR. A new prize denom
// pays exactly the prize fee to txfeecollector_cp, an existing one nothing.
func (s *denomRewardsSuite) airdrop2(t *testing.T, prize string, amount int64, newPrize bool) {
	t.Helper()
	creator := s.creator.FormattedAddress()
	ubzeBefore, prizeBefore, escrowBefore := s.balance(t, creator, chain.Denom), s.balance(t, creator, prize), s.balance(t, s.module, prize)
	prizes := len(s.prizesOf(t, s.stk2))

	res := s.mustExec(t, s.creator, "rewards", "distribute-denom-rewards", s.stk2, prize, strconv.FormatInt(amount, 10))
	s.requireTxEvent(t, res, "DenomRewardDistributionEvent", map[string]string{
		"denom": s.stk2, "prize_denom": prize, "amount": strconv.FormatInt(amount, 10),
	})
	fee, created := sdkmath.ZeroInt(), 0
	if newPrize {
		fee, created = s.params.CreateDenomRewardPrizeFee.Amount, 1
	}
	require.Len(t, s.prizeCreated(res, s.stk2, prize), created)
	require.Equal(t, fee.String(), s.cpFee(t, res).String(), "prize fee at txfeecollector_cp")
	require.Equal(t, ubzeBefore.Sub(fee).Sub(s.txFee(t, res)).String(), s.balance(t, creator, chain.Denom).String())
	require.Equal(t, prizeBefore.SubRaw(amount).String(), s.balance(t, creator, prize).String())
	require.Equal(t, escrowBefore.AddRaw(amount).String(), s.balance(t, s.module, prize).String())
	require.Len(t, s.prizesOf(t, s.stk2), prizes+created, "only a new prize denom takes a prize slot")
}

// schedule2 attaches a schedule to the second DR and returns its id. The
// creator pays the schedule fee, plus the prize fee for a new prize denom.
func (s *denomRewardsSuite) schedule2(t *testing.T, prize string, daily, days int64, newPrize bool) string {
	t.Helper()
	creator := s.creator.FormattedAddress()
	ubzeBefore, prizeBefore, escrowBefore := s.balance(t, creator, chain.Denom), s.balance(t, creator, prize), s.balance(t, s.module, prize)
	prizes := len(s.prizesOf(t, s.stk2))

	res := s.mustExec(t, s.creator, "rewards", "create-denom-reward-schedule",
		s.stk2, prize, strconv.FormatInt(daily, 10), strconv.FormatInt(days, 10))
	created := s.requireTxEvent(t, res, "DenomRewardScheduleCreateEvent", map[string]string{
		"denom":        s.stk2,
		"prize_denom":  prize,
		"daily_amount": strconv.FormatInt(daily, 10),
		"duration":     strconv.FormatInt(days, 10),
	})
	require.NotEmpty(t, created.Attributes["schedule_id"])
	fees, newPrizes := s.params.AddDenomRewardScheduleFee.Amount, 0
	if newPrize {
		fees, newPrizes = fees.Add(s.params.CreateDenomRewardPrizeFee.Amount), 1
	}
	require.Len(t, s.prizeCreated(res, s.stk2, prize), newPrizes)
	require.Equal(t, fees.String(), s.cpFee(t, res).String(), "fees at txfeecollector_cp")
	require.Equal(t, ubzeBefore.Sub(fees).Sub(s.txFee(t, res)).String(), s.balance(t, creator, chain.Denom).String())
	require.Equal(t, prizeBefore.SubRaw(daily*days).String(), s.balance(t, creator, prize).String())
	require.Equal(t, escrowBefore.AddRaw(daily*days).String(), s.balance(t, s.module, prize).String())
	require.Len(t, s.prizesOf(t, s.stk2), prizes+newPrizes, "only a new prize denom takes a prize slot")
	return created.Attributes["schedule_id"]
}

func (s *denomRewardsSuite) prizeCreated(res helpers.TxResult, denom, prize string) []helpers.Event {
	return helpers.EventsWith(helpers.TxEvents(res, drEventPrefix+"DenomRewardPrizeCreateEvent"), map[string]string{"denom": denom, "prize_denom": prize})
}

// cpFee is the ubze the tx moved from the rewards module to txfeecollector_cp.
func (s *denomRewardsSuite) cpFee(t *testing.T, res helpers.TxResult) sdkmath.Int {
	t.Helper()
	return s.transferred(t, res, s.module, s.cpCollector, chain.Denom)
}

func (s *denomRewardsSuite) transferred(t *testing.T, res helpers.TxResult, from, to, denom string) sdkmath.Int {
	t.Helper()
	moved, err := helpers.Transferred(res.Events, from, to, denom)
	require.NoError(t, err)
	return moved
}

// mustExecMsgs sends msgs in one tx, with drBatchGasPerMsg gas per message.
func (s *denomRewardsSuite) mustExecMsgs(t *testing.T, user ibc.Wallet, msgs ...any) helpers.TxResult {
	t.Helper()
	gas := uint64(len(msgs)) * drBatchGasPerMsg
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(gas)}, msgs...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx of %d msgs failed: %s", len(msgs), res.RawLog)
	return res
}

// drMsg builds a bze.rewards message of the user on a staking denom.
func drMsg(typ string, user ibc.Wallet, denom string, fields map[string]any) map[string]any {
	m := map[string]any{"@type": "/bze.rewards." + typ, "creator": user.FormattedAddress(), "denom": denom}
	for k, v := range fields {
		m[k] = v
	}
	return m
}

func drAirdropMsg(user ibc.Wallet, denom, prize, amount string) map[string]any {
	return drMsg("MsgDistributeDenomRewards", user, denom, map[string]any{"prize_denom": prize, "amount": amount})
}

// drState is what a refused tx must leave unchanged: the rewards module's
// and the sender's balances, every DR, the second DR's schedules, the prize
// accumulators of the second and the prize-cap DR, the pending unlocks and the
// sender's positions with their pending prizes (which pins its indexes too).
type drState struct {
	Module    string
	Sender    sdk.Coins
	Rewards   []denomRewardInfo
	Schedules []scheduleInfo
	Prizes    [][]prizeInfo
	Unlocks   []pendingUnlockInfo
	Positions []string
}

func (s *denomRewardsSuite) drSnapshot(t *testing.T, sender ibc.Wallet) drState {
	t.Helper()
	st := drState{
		Module:    s.allBalances(t, s.module).String(),
		Sender:    s.allBalances(t, sender.FormattedAddress()),
		Rewards:   s.allDenomRewards(t),
		Schedules: s.schedulesOf(t, s.stk2),
		Unlocks:   s.pendingUnlocks(t),
	}
	for _, denom := range []string{s.stk2, s.capStk} {
		if denom != "" {
			st.Prizes = append(st.Prizes, s.prizesOf(t, denom))
		}
	}
	for _, p := range s.positions(t, sender) {
		_, pending := s.participantIn(t, sender, p.StakingDenom)
		st.Positions = append(st.Positions, fmt.Sprintf("%s %s %s", p.StakingDenom, p.Amount, sdk.NewCoins(pending...)))
	}
	return st
}

// requireRefusedMsg sends msg with a fixed gas limit and asserts the exact
// refusal (codespace, code, reason, CheckTx or not) and that it changed
// nothing but the sender's tx fee (none when CheckTx refused it).
func (s *denomRewardsSuite) requireRefusedMsg(t *testing.T, user ibc.Wallet, msg map[string]any, codespace string, code uint32, reason string, checkTx bool) {
	t.Helper()
	before := s.drSnapshot(t, user)
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(drTxGas)}, msg)
	got, err := helpers.RejectionReason(res, err)
	require.NoError(t, err, "msg %v", msg)
	require.Contains(t, got, reason)
	require.Equal(t, codespace, res.Codespace, got)
	require.EqualValues(t, code, res.Code, got)
	require.Equal(t, checkTx, res.Height == 0, "refused by CheckTx: want %v, got height %d", checkTx, res.Height)

	after := s.drSnapshot(t, user)
	wantSender := before.Sender
	if !checkTx {
		wantSender = wantSender.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res)))
	}
	require.Equal(t, wantSender.String(), after.Sender.String(), "the sender pays the tx fee only")
	before.Sender, after.Sender = nil, nil
	require.Equal(t, before, after, "a refused tx must not change the rewards state")
}

func (s *denomRewardsSuite) allBalances(t *testing.T, addr string) sdk.Coins {
	t.Helper()
	res, err := helpers.Query[struct {
		Balances sdk.Coins `json:"balances"`
	}](s.ctx, s.bze, "bank", "balances", addr)
	require.NoError(t, err)
	return sdk.NewCoins(res.Balances...)
}

func (s *denomRewardsSuite) allDenomRewards(t *testing.T) []denomRewardInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		List []denomRewardInfo `json:"list"`
	}](s.ctx, s.bze, "rewards", "all-denom-rewards")
	require.NoError(t, err)
	return res.List
}

type positionInfo struct {
	StakingDenom string `json:"staking_denom"`
	Amount       string `json:"amount"`
}

// positions lists the user's DR positions (`denom-reward-participations`).
func (s *denomRewardsSuite) positions(t *testing.T, user ibc.Wallet) []positionInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		List []positionInfo `json:"list"`
	}](s.ctx, s.bze, "rewards", "denom-reward-participations", user.FormattedAddress())
	require.NoError(t, err)
	return res.List
}

type prizeInfo struct {
	PrizeDenom       string `json:"prize_denom"`
	DistributedStake string `json:"distributed_stake"`
}

func (s *denomRewardsSuite) prizesOf(t *testing.T, denom string) []prizeInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		List []prizeInfo `json:"list"`
	}](s.ctx, s.bze, "rewards", "denom-reward-prizes", denom)
	require.NoError(t, err)
	return res.List
}

// accumulator returns a prize's S scaled by 10^18.
func (s *denomRewardsSuite) accumulator(t *testing.T, denom, prize string) *big.Int {
	t.Helper()
	for _, p := range s.prizesOf(t, denom) {
		if p.PrizeDenom == prize {
			return drScaled(t, p.DistributedStake)
		}
	}
	require.FailNow(t, "no prize accumulator", "%s on %s", prize, denom)
	return nil
}

func (s *denomRewardsSuite) requireAccumulator(t *testing.T, denom, prize string, want *big.Int) {
	t.Helper()
	require.Equal(t, want.String(), s.accumulator(t, denom, prize).String(), "accumulator of %s (×10^18)", prize)
}

// distributedAmounts returns the amounts of the DenomRewardDistributionEvents
// of a day tick's block for one prize denom.
func (s *denomRewardsSuite) distributedAmounts(t *testing.T, day helpers.EpochInfo, denom, prize string) []string {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, int64(day.CurrentEpochStartHeight), drEventPrefix+"DenomRewardDistributionEvent")
	require.NoError(t, err)
	var amounts []string
	for _, e := range helpers.EventsWith(events, map[string]string{"denom": denom, "prize_denom": prize}) {
		amounts = append(amounts, e.Attributes["amount"])
	}
	return amounts
}

func (s *denomRewardsSuite) finishEvents(t *testing.T, day helpers.EpochInfo, scheduleID string) []helpers.Event {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, int64(day.CurrentEpochStartHeight), drEventPrefix+"DenomRewardScheduleFinishEvent")
	require.NoError(t, err)
	return helpers.EventsWith(events, map[string]string{"schedule_id": scheduleID})
}

// unlockEpoch is the hour epoch a DR exit at height unlocks at.
func (s *denomRewardsSuite) unlockEpoch(t *testing.T, height int64) int64 {
	t.Helper()
	hour, err := helpers.Query[struct {
		CurrentEpoch helpers.Int64 `json:"current_epoch"`
	}](s.ctx, s.bze, "epochs", "identifier", "hour", "--height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	return int64(hour.CurrentEpoch) + int64(s.params.DenomRewardLock)*24
}

func (s *denomRewardsSuite) requireErinUnlock(t *testing.T, pending bool) {
	t.Helper()
	erin := s.erin.FormattedAddress()
	index := fmt.Sprintf("%d/dr/%s/%s", s.erinUnlock, s.stk2, erin)
	entry, found := s.pendingUnlock(t, index)
	require.Equal(t, pending, found, "pending unlock %s", index)
	if pending {
		require.Equal(t, pendingUnlockInfo{Index: index, Address: erin, Amount: strconv.Itoa(dr2ErinStake), Denom: s.stk2}, entry)
	}
}

// drUnits returns v scaled by 10^18, the accumulator's fixed point.
func drUnits(v int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(v), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
}

// drQuo is amount / staked truncated to 18 decimals, scaled by 10^18: what
// one distribution adds to an accumulator (LegacyDec.QuoTruncate).
func drQuo(amount, staked int64) *big.Int {
	return new(big.Int).Quo(drUnits(amount), big.NewInt(staked))
}

// drScaled reads an accumulator the CLI printed, scaled by 10^18. A gogoproto
// LegacyDec field holds its raw scaled integer, which the CLI may print as is
// ("1428571428571428571") or as a decimal ("1.428571428571428571").
func drScaled(t *testing.T, v string) *big.Int {
	t.Helper()
	if strings.Contains(v, ".") {
		d, err := sdkmath.LegacyNewDecFromStr(v)
		require.NoError(t, err, "decimal %q", v)
		return d.BigInt()
	}
	if v == "" {
		return new(big.Int)
	}
	i, ok := new(big.Int).SetString(v, 10)
	require.True(t, ok, "accumulator %q", v)
	return i
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
	return s.denomRewardOf(t, s.stk)
}

func (s *denomRewardsSuite) denomRewardOf(t *testing.T, denom string) denomRewardInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		DenomReward denomRewardInfo `json:"denom_reward"`
	}](s.ctx, s.bze, "rewards", "denom-reward", denom)
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
	return s.schedulesOf(t, s.stk)
}

func (s *denomRewardsSuite) schedulesOf(t *testing.T, denom string) []scheduleInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		List []scheduleInfo `json:"list"`
	}](s.ctx, s.bze, "rewards", "denom-reward-schedules", denom)
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
	return s.participantIn(t, user, s.stk)
}

func (s *denomRewardsSuite) participantIn(t *testing.T, user ibc.Wallet, denom string) (sdkmath.Int, sdk.Coins) {
	t.Helper()
	res, err := helpers.Query[struct {
		Participant struct {
			Amount string `json:"amount"`
		} `json:"participant"`
		Pending sdk.Coins `json:"pending"`
	}](s.ctx, s.bze, "rewards", "denom-reward-participant", user.FormattedAddress(), denom)
	require.NoError(t, err)
	return parseInt(res.Participant.Amount), res.Pending
}

func (s *denomRewardsSuite) requirePending(t *testing.T, user ibc.Wallet, want sdk.Coins) {
	t.Helper()
	s.requirePendingIn(t, user, s.stk, want)
}

func (s *denomRewardsSuite) requirePendingIn(t *testing.T, user ibc.Wallet, denom string, want sdk.Coins) {
	t.Helper()
	_, pending := s.participantIn(t, user, denom)
	require.Equal(t, want.String(), sdk.NewCoins(pending...).String(), "pending of %s in %s", user.KeyName(), denom)
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
	for _, e := range s.pendingUnlocks(t) {
		if e.Index == index {
			return e, true
		}
	}
	return pendingUnlockInfo{}, false
}

func (s *denomRewardsSuite) pendingUnlocks(t *testing.T) []pendingUnlockInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		List []pendingUnlockInfo `json:"list"`
	}](s.ctx, s.bze, "rewards", "all-pending-unlock-participants")
	require.NoError(t, err)
	return res.List
}

// rewardsParams holds the rewards params the suite asserts against.
type rewardsParams struct {
	CreateDenomRewardFee      sdk.Coin
	CreateDenomRewardPrizeFee sdk.Coin
	AddDenomRewardScheduleFee sdk.Coin
	ExtraGasForDenomExit      helpers.Int64
	DenomRewardLock           helpers.Int64
	DenomRewardMinStake       helpers.Int64
	MaxPrizeDenomsPerDr       helpers.Int64
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
	field(&p.MaxPrizeDenomsPerDr, "maxPrizeDenomsPerDr", "max_prize_denoms_per_dr")

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
