package interchaintest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"testing"

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

// The first payout splits exactly (1,000,000 over 4,000,000 staked is 0.25 per
// unit). Carol's 3 extra stk make the second one 1,000,000 / 4,000,003, which
// does not divide: the accumulator rounds down and the remainder stays in the
// rewards module as dust.
const (
	srCreatorFunds = 150_000_000_000 // 150,000 BZE: two denoms, two staking rewards
	srUserFunds    = 1_000_000_000   // 1,000 BZE for gas

	srPrize    = 1_000_000
	srDuration = 2
	srLock     = 1

	srAliceStake = 3_000_000
	srBobStake   = 1_000_000
	srCarolStake = 3

	// the second staking reward, only to be refused deletion while unfinished
	srFreshPrize = 1_000

	// LegacyDec precision: the accumulator S has 18 decimals
	decPrecision = 18
)

// TestStakingRewards runs a staking reward's lifecycle on one network with
// fast epochs: the day-epoch payout queue in EndBlock, claims, exits into the
// hour-epoch pending unlock and the removal of the finished reward. Subtests
// run in order and share state; the step numbers match the suite's design.
//
// Timeline (D = the day epoch right after the reward is created):
//
//	D    alice and bob join (2)
//	D+1  first payout; carol joins (3)
//	D+2  second and last payout; alice claims (4), delete refused (6),
//	     alice and bob exit (7a), a fresh reward refuses deletion (9)
//	next tick  nothing is paid (5); carol's exit removes the reward (8)
//	+24 hour epochs  alice's and bob's stakes return (7b)
func TestStakingRewards(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	t.Parallel()
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupStakingRewards(t, ctx, net.Chain)

	t.Run("1 create a staking reward", s.create)
	t.Run("2 join and first payout", s.joinAndFirstPayout)
	t.Run("3 second payout rounds down", s.roundingDown)
	t.Run("4 claim", s.claim)
	t.Run("6 delete refused while staked", s.deleteWhileStaked)
	t.Run("7a exit queues the stake for unlock", s.exit)
	t.Run("9 delete refused while unfinished", s.deleteUnfinished)
	t.Run("5 nothing is paid after the last payout", s.afterLastPayout)
	t.Run("8 the last exit removes the finished reward", s.lastExitRemoves)
	t.Run("7b unlock returns the stake", s.unlock)
	t.Run("10 escrow at the end", s.escrowAtEnd)
}

type stakingRewardsSuite struct {
	ctx context.Context
	bze *cosmos.CosmosChain

	creator, alice, bob, carol, dave ibc.Wallet
	stk, prz                         string
	module, cpCollector              string
	params                           srParams

	// baseline rewards module balances, before any staking reward exists
	baseline map[string]sdkmath.Int
	// funded is all prz escrowed by staking rewards, paid all prz paid out of
	// the escrow; the module holds exactly baseline + funded - paid
	funded, paid sdkmath.Int

	rewardID string
	window   int64 // the day epoch D
	// delta2 is what the second payout added to S, scaled by 10^18
	delta2 *big.Int
	// unlockAt is the hour epoch each exited staker's stake unlocks at
	unlockAt map[string]int64
}

func setupStakingRewards(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *stakingRewardsSuite {
	s := &stakingRewardsSuite{
		ctx: ctx, bze: bze,
		baseline: map[string]sdkmath.Int{},
		funded:   sdkmath.ZeroInt(), paid: sdkmath.ZeroInt(),
		unlockAt: map[string]int64{},
	}

	s.creator = interchaintest.GetAndFundTestUsers(t, ctx, "srcreator", sdkmath.NewInt(srCreatorFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "sr", sdkmath.NewInt(srUserFunds), bze, bze, bze, bze)
	s.alice, s.bob, s.carol, s.dave = users[0], users[1], users[2], users[3]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	factory := "factory/" + s.creator.FormattedAddress() + "/"
	s.stk, s.prz = factory+"stk", factory+"prz"
	for denom, amount := range map[string]int64{
		s.stk: srAliceStake + srBobStake + srCarolStake,
		s.prz: srPrize*srDuration + srFreshPrize,
	} {
		s.mustExec(t, s.creator, "tokenfactory", "create-denom", denom[len(factory):])
		s.mustExec(t, s.creator, "tokenfactory", "mint", coinStr(amount, denom))
	}
	for _, send := range []struct {
		to     ibc.Wallet
		amount int64
	}{
		{s.alice, srAliceStake},
		{s.bob, srBobStake},
		{s.carol, srCarolStake},
	} {
		s.mustExec(t, s.creator, "bank", "send", s.creator.FormattedAddress(), send.to.FormattedAddress(), coinStr(send.amount, s.stk))
	}

	var err error
	s.module, err = helpers.ModuleAddress(ctx, bze, drRewardsModule)
	require.NoError(t, err)
	s.cpCollector, err = helpers.ModuleAddress(ctx, bze, "txfeecollector_cp")
	require.NoError(t, err)
	for _, denom := range []string{s.stk, s.prz} {
		s.baseline[denom] = s.balance(t, s.module, denom)
	}
	s.params = s.queryParams(t)
	return s
}

func (s *stakingRewardsSuite) create(t *testing.T) {
	fee := s.params.CreateStakingRewardFee
	require.Equal(t, chain.Denom, fee.Denom, "create_staking_reward_fee is not in %s", chain.Denom)
	require.True(t, fee.IsPositive(), "create_staking_reward_fee is zero")
	creator := s.creator.FormattedAddress()
	ubzeBefore, przBefore := s.balance(t, creator, chain.Denom), s.balance(t, creator, s.prz)

	res := s.mustExec(t, s.creator, "rewards", "create-staking-reward",
		strconv.Itoa(srPrize), s.prz, s.stk, strconv.Itoa(srDuration), "1", strconv.Itoa(srLock))
	created := s.requireTxEvent(t, res, "StakingRewardCreateEvent", map[string]string{
		"prize_amount":  strconv.Itoa(srPrize),
		"prize_denom":   s.prz,
		"staking_denom": s.stk,
		"duration":      strconv.Itoa(srDuration),
		"min_stake":     "1",
		"lock":          strconv.Itoa(srLock),
	})
	s.rewardID = created.Attributes["reward_id"]
	require.NotEmpty(t, s.rewardID)

	budget := sdkmath.NewInt(srPrize * srDuration)
	s.funded = s.funded.Add(budget)
	require.Equal(t, ubzeBefore.Sub(fee.Amount).Sub(s.txFee(t, res)).String(), s.balance(t, creator, chain.Denom).String(),
		"creator must pay exactly the create fee and the tx fee")
	feeMoved, err := helpers.Transferred(helpers.TxEvents(res, "transfer"), s.module, s.cpCollector, chain.Denom)
	require.NoError(t, err)
	require.Equal(t, fee.Amount.String(), feeMoved.String(), "the create fee must reach the txfeecollector_cp module")
	require.Equal(t, przBefore.Sub(budget).String(), s.balance(t, creator, s.prz).String())
	s.requireEscrow(t)

	sr := s.stakingReward(t)
	require.Equal(t, strconv.Itoa(srPrize), sr.PrizeAmount)
	require.Equal(t, int64(srDuration), int64(sr.Duration))
	require.Zero(t, int64(sr.Payouts))
	require.Equal(t, int64(srLock), int64(sr.Lock))
	require.Equal(t, "0", sr.staked().String())
	require.Zero(t, decScaled(t, sr.DistributedStake).Sign())

	// D starts here: the joins of step 2 must fall in one day epoch
	s.window = int64(s.waitDayTick(t).CurrentEpoch)
	require.Zero(t, int64(s.stakingReward(t).Payouts), "a reward without stakers must not pay")
}

func (s *stakingRewardsSuite) joinAndFirstPayout(t *testing.T) {
	for _, join := range []struct {
		user   ibc.Wallet
		amount int64
	}{
		{s.alice, srAliceStake},
		{s.bob, srBobStake},
	} {
		res := s.mustExec(t, join.user, "rewards", "join-staking", s.rewardID, strconv.FormatInt(join.amount, 10))
		s.requireTxEvent(t, res, "StakingRewardJoinEvent", map[string]string{
			"reward_id": s.rewardID, "address": join.user.FormattedAddress(), "amount": strconv.FormatInt(join.amount, 10),
		})
	}
	s.requireSameDay(t, s.window, "alice's and bob's joins")
	require.Equal(t, strconv.Itoa(srAliceStake+srBobStake), s.stakingReward(t).staked().String())
	require.Equal(t, s.baseline[s.stk].AddRaw(srAliceStake+srBobStake).String(), s.balance(t, s.module, s.stk).String())

	day := s.waitDayTick(t)
	require.Equal(t, s.window+1, int64(day.CurrentEpoch))
	sr := s.stakingReward(t)
	require.Equal(t, int64(1), int64(sr.Payouts))
	// S = 1,000,000 / 4,000,000 = 0.25 exactly
	require.Equal(t, "0.250000000000000000", sr.DistributedStake)
	s.requirePending(t, s.alice, 750_000)
	s.requirePending(t, s.bob, 250_000)
	s.requireEscrow(t)
	assert.Len(t, s.distributionEvents(t, day), 1, "the payout must emit StakingRewardDistributionEvent in EndBlock")
}

func (s *stakingRewardsSuite) roundingDown(t *testing.T) {
	res := s.mustExec(t, s.carol, "rewards", "join-staking", s.rewardID, strconv.Itoa(srCarolStake))
	s.requireTxEvent(t, res, "StakingRewardJoinEvent", map[string]string{
		"reward_id": s.rewardID, "address": s.carol.FormattedAddress(), "amount": strconv.Itoa(srCarolStake),
	})
	s.requireSameDay(t, s.window+1, "carol's join")
	const staked = srAliceStake + srBobStake + srCarolStake
	require.Equal(t, strconv.Itoa(staked), s.stakingReward(t).staked().String())
	s.requirePending(t, s.carol, 0)

	day := s.waitDayTick(t)
	require.Equal(t, s.window+2, int64(day.CurrentEpoch))
	sr := s.stakingReward(t)
	require.Equal(t, int64(srDuration), int64(sr.Payouts))
	assert.Len(t, s.distributionEvents(t, day), 1, "the payout must emit StakingRewardDistributionEvent in EndBlock")

	// S += QuoTruncate(1,000,000 / 4,000,003): rounded down at 18 decimals
	s.delta2 = new(big.Int).Quo(scaled(srPrize), big.NewInt(staked))
	wantS := new(big.Int).Add(new(big.Int).Quo(scaled(srPrize), big.NewInt(srAliceStake+srBobStake)), s.delta2)
	require.Equal(t, wantS.String(), decScaled(t, sr.DistributedStake).String(), "S must round down to %s", sr.DistributedStake)

	// each claim is floor(stake × ΔS)
	second := map[string]int64{
		s.alice.KeyName(): mulFloor(srAliceStake, s.delta2),
		s.bob.KeyName():   mulFloor(srBobStake, s.delta2),
		s.carol.KeyName(): mulFloor(srCarolStake, s.delta2),
	}
	s.requirePending(t, s.alice, 750_000+second[s.alice.KeyName()])
	s.requirePending(t, s.bob, 250_000+second[s.bob.KeyName()])
	s.requirePending(t, s.carol, second[s.carol.KeyName()])

	var sum int64
	for _, v := range second {
		sum += v
	}
	require.LessOrEqual(t, sum, int64(srPrize), "the second payout must never promise more than it funds")
	require.Positive(t, srPrize-sum, "1,000,000 / 4,000,003 does not divide, so dust must remain")
	t.Logf("second payout: %d of %d claimable, %d dust", sum, srPrize, srPrize-sum)
	s.requireEscrow(t)
}

func (s *stakingRewardsSuite) claim(t *testing.T) {
	alice := s.alice.FormattedAddress()
	pending := s.pending(t, s.alice)
	require.True(t, pending.IsPositive())
	before := s.balance(t, alice, s.prz)

	res := s.mustExec(t, s.alice, "rewards", "claim-staking-rewards", s.rewardID)
	s.requireTxEvent(t, res, "StakingRewardClaimEvent", map[string]string{
		"reward_id": s.rewardID, "address": alice, "amount": pending.String(),
	})
	s.paid = s.paid.Add(pending)
	require.Equal(t, before.Add(pending).String(), s.balance(t, alice, s.prz).String())
	s.requirePending(t, s.alice, 0)
	s.requireEscrow(t)

	s.requireRefused(t, "no rewards available to claim", s.alice, "rewards", "claim-staking-rewards", s.rewardID)
}

func (s *stakingRewardsSuite) deleteWhileStaked(t *testing.T) {
	require.Equal(t, int64(srDuration), int64(s.stakingReward(t).Payouts))
	s.requireRefused(t, "still has staked funds", s.dave, "rewards", "delete-staking-reward", s.rewardID)
}

func (s *stakingRewardsSuite) exit(t *testing.T) {
	for _, user := range []ibc.Wallet{s.alice, s.bob} {
		addr := user.FormattedAddress()
		amount, _ := s.participant(t, user)
		pending := s.pending(t, user)
		przBefore, stkBefore := s.balance(t, addr, s.prz), s.balance(t, addr, s.stk)
		stakedBefore := s.stakingReward(t).staked()

		res := s.mustExec(t, user, "rewards", "exit-staking", s.rewardID)
		s.requireTxEvent(t, res, "StakingRewardExitEvent", map[string]string{"reward_id": s.rewardID, "address": addr})
		require.Greater(t, res.GasUsed, int64(s.params.ExtraGasForExitStake), "exit must charge extra_gas_for_exit_stake")

		// the exit pays what is pending and locks the stake
		s.paid = s.paid.Add(pending)
		require.Equal(t, przBefore.Add(pending).String(), s.balance(t, addr, s.prz).String(), "exit must pay %s's pending", user.KeyName())
		require.Equal(t, stkBefore.String(), s.balance(t, addr, s.stk).String(), "the stake stays locked until the unlock epoch")
		require.Equal(t, stakedBefore.Sub(amount).String(), s.stakingReward(t).staked().String())
		_, found := s.participant(t, user)
		require.False(t, found, "%s must no longer participate", user.KeyName())
		s.requireQueuedUnlock(t, user, res.Height, amount)
	}
	s.requireEscrow(t)
}

func (s *stakingRewardsSuite) deleteUnfinished(t *testing.T) {
	res := s.mustExec(t, s.creator, "rewards", "create-staking-reward",
		strconv.Itoa(srFreshPrize), s.prz, s.stk, "1", "1", "0")
	fresh := s.requireTxEvent(t, res, "StakingRewardCreateEvent", map[string]string{"prize_amount": strconv.Itoa(srFreshPrize)})
	freshID := fresh.Attributes["reward_id"]
	require.NotEqual(t, s.rewardID, freshID)
	s.funded = s.funded.AddRaw(srFreshPrize)

	before := s.stakingRewardByID(t, freshID)
	s.requireRefused(t, "staking reward is not finished", s.dave, "rewards", "delete-staking-reward", freshID)
	require.Equal(t, before, s.stakingRewardByID(t, freshID), "a refused delete must not change the reward")
	s.requireEscrow(t)
}

func (s *stakingRewardsSuite) afterLastPayout(t *testing.T) {
	before := s.stakingReward(t)
	carolPending := s.pending(t, s.carol)

	// the steps since the last payout may already have crossed D+3; any later
	// tick proves the point as well
	day := s.waitDayTick(t)
	require.Greater(t, int64(day.CurrentEpoch), s.window+2)
	require.Equal(t, before, s.stakingReward(t), "a finished reward must not change on a day tick")
	require.Equal(t, int64(srDuration), int64(before.Payouts))
	require.Empty(t, s.distributionEvents(t, day), "a finished reward must not pay")
	require.Equal(t, carolPending.String(), s.pending(t, s.carol).String())
}

func (s *stakingRewardsSuite) lastExitRemoves(t *testing.T) {
	carol := s.carol.FormattedAddress()
	pending := s.pending(t, s.carol)
	przBefore := s.balance(t, carol, s.prz)

	res := s.mustExec(t, s.carol, "rewards", "exit-staking", s.rewardID)
	s.requireTxEvent(t, res, "StakingRewardExitEvent", map[string]string{"reward_id": s.rewardID, "address": carol})
	// the keeper removes a finished reward on its last exit, so
	// MsgDeleteStakingReward has nothing left to delete
	s.requireTxEvent(t, res, "StakingRewardFinishEvent", map[string]string{"reward_id": s.rewardID})
	s.paid = s.paid.Add(pending)
	require.Equal(t, przBefore.Add(pending).String(), s.balance(t, carol, s.prz).String())
	s.requireQueuedUnlock(t, s.carol, res.Height, sdkmath.NewInt(srCarolStake))

	_, err := helpers.Query[json.RawMessage](s.ctx, s.bze, "rewards", "get-staking-reward", s.rewardID)
	require.Error(t, err, "the finished reward must be gone")
	for _, sr := range s.allStakingRewards(t) {
		require.NotEqual(t, s.rewardID, sr.RewardID, "the finished reward must be gone from the list")
	}
	s.requireRefused(t, "invalid reward_id", s.dave, "rewards", "delete-staking-reward", s.rewardID)

	// what the rounding left behind stays in the module: the reward's whole
	// budget minus everything paid
	dust := sdkmath.NewInt(srPrize * srDuration).Sub(s.paid)
	require.Equal(t, dust.String(), sdkmath.NewInt(srPrize-
		mulFloor(srAliceStake, s.delta2)-mulFloor(srBobStake, s.delta2)-mulFloor(srCarolStake, s.delta2)).String(),
		"the dust must be exactly the second payout's remainder")
	s.requireEscrow(t)
}

func (s *stakingRewardsSuite) unlock(t *testing.T) {
	last := int64(0)
	for _, user := range []ibc.Wallet{s.alice, s.bob} {
		last = max(last, s.unlockAt[user.KeyName()])
	}
	ctx, cancel := context.WithTimeout(s.ctx, unlockTimeout)
	defer cancel()
	// AfterEpochEnd(h) runs when the hour epoch moves to h+1; the unlock queue
	// it fills is drained in that block's EndBlock.
	require.NoError(t, helpers.WaitForEpoch(ctx, s.bze, "hour", last+1))

	for _, back := range []struct {
		user   ibc.Wallet
		amount int64
	}{
		{s.alice, srAliceStake},
		{s.bob, srBobStake},
	} {
		require.Equal(t, strconv.FormatInt(back.amount, 10), s.balance(t, back.user.FormattedAddress(), s.stk).String(),
			"%s's stake must be returned", back.user.KeyName())
		_, found := s.pendingUnlock(t, s.unlockIndex(back.user))
		require.False(t, found, "the pending unlock must be removed once paid")
	}
}

func (s *stakingRewardsSuite) escrowAtEnd(t *testing.T) {
	// carol's stake unlocks 24 hour epochs after her exit, past the end of
	// the suite: it is the only stk left in the module
	_, found := s.pendingUnlock(t, s.unlockIndex(s.carol))
	require.True(t, found, "carol's unlock must still be pending")
	require.Equal(t, s.baseline[s.stk].AddRaw(srCarolStake).String(), s.balance(t, s.module, s.stk).String())
	s.requireEscrow(t)
}

// requireEscrow asserts the rewards module holds exactly what was funded minus
// what was paid, and that this covers every open promise: the unpaid budget
// of each reward plus everything currently claimable.
func (s *stakingRewardsSuite) requireEscrow(t *testing.T) {
	t.Helper()
	escrow := s.balance(t, s.module, s.prz).Sub(s.baseline[s.prz])
	require.Equal(t, s.funded.Sub(s.paid).String(), escrow.String(), "rewards module prz must be funded - paid")

	owed := sdkmath.ZeroInt()
	for _, sr := range s.allStakingRewards(t) {
		unpaid := int64(sr.Duration) - int64(sr.Payouts)
		owed = owed.Add(parseInt(sr.PrizeAmount).MulRaw(unpaid))
	}
	for _, user := range []ibc.Wallet{s.alice, s.bob, s.carol} {
		if _, found := s.participant(t, user); found {
			owed = owed.Add(s.pending(t, user))
		}
	}
	require.True(t, escrow.GTE(owed), "the escrow %s must cover the unpaid budget and pending claims %s", escrow, owed)
}

func (s *stakingRewardsSuite) requireQueuedUnlock(t *testing.T, user ibc.Wallet, height int64, amount sdkmath.Int) {
	t.Helper()
	hour, err := helpers.Query[struct {
		CurrentEpoch helpers.Int64 `json:"current_epoch"`
	}](s.ctx, s.bze, "epochs", "identifier", "hour", "--height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	s.unlockAt[user.KeyName()] = int64(hour.CurrentEpoch) + srLock*24

	entry, found := s.pendingUnlock(t, s.unlockIndex(user))
	require.True(t, found, "no pending unlock %s", s.unlockIndex(user))
	require.Equal(t, user.FormattedAddress(), entry.Address)
	require.Equal(t, s.stk, entry.Denom)
	require.Equal(t, amount.String(), entry.Amount)
}

func (s *stakingRewardsSuite) unlockIndex(user ibc.Wallet) string {
	return fmt.Sprintf("%d/%s/%s", s.unlockAt[user.KeyName()], s.rewardID, user.FormattedAddress())
}

func (s *stakingRewardsSuite) pendingUnlock(t *testing.T, index string) (pendingUnlockInfo, bool) {
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

type srInfo struct {
	RewardID         string        `json:"reward_id"`
	PrizeAmount      string        `json:"prize_amount"`
	PrizeDenom       string        `json:"prize_denom"`
	StakingDenom     string        `json:"staking_denom"`
	Duration         helpers.Int64 `json:"duration"`
	Payouts          helpers.Int64 `json:"payouts"`
	MinStake         helpers.Int64 `json:"min_stake"`
	Lock             helpers.Int64 `json:"lock"`
	StakedAmount     string        `json:"staked_amount"`
	DistributedStake string        `json:"distributed_stake"`
}

func (sr srInfo) staked() sdkmath.Int {
	return parseInt(sr.StakedAmount)
}

func (s *stakingRewardsSuite) stakingReward(t *testing.T) srInfo {
	t.Helper()
	return s.stakingRewardByID(t, s.rewardID)
}

func (s *stakingRewardsSuite) stakingRewardByID(t *testing.T, id string) srInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		StakingReward srInfo `json:"staking_reward"`
	}](s.ctx, s.bze, "rewards", "get-staking-reward", id)
	require.NoError(t, err)
	require.Equal(t, id, res.StakingReward.RewardID)
	return res.StakingReward
}

func (s *stakingRewardsSuite) allStakingRewards(t *testing.T) []srInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		List []srInfo `json:"list"`
	}](s.ctx, s.bze, "rewards", "all-staking-rewards")
	require.NoError(t, err)
	return res.List
}

// participant returns the user's stake in the suite's reward and its S0.
func (s *stakingRewardsSuite) participant(t *testing.T, user ibc.Wallet) (sdkmath.Int, bool) {
	t.Helper()
	p, found := s.participation(t, user)
	if !found {
		return sdkmath.ZeroInt(), false
	}
	return parseInt(p.Amount), true
}

type srParticipant struct {
	RewardID string `json:"reward_id"`
	Amount   string `json:"amount"`
	JoinedAt string `json:"joined_at"`
}

func (s *stakingRewardsSuite) participation(t *testing.T, user ibc.Wallet) (srParticipant, bool) {
	t.Helper()
	res, err := helpers.Query[struct {
		List []srParticipant `json:"list"`
	}](s.ctx, s.bze, "rewards", "staking-reward-participant", user.FormattedAddress())
	require.NoError(t, err)
	for _, p := range res.List {
		if p.RewardID == s.rewardID {
			return p, true
		}
	}
	return srParticipant{}, false
}

// pending is what the user can claim now: floor(stake × (S − S0)). The chain
// has no pending query for staking rewards, so it is derived from the stored
// stake, the user's S0 and the reward's S.
func (s *stakingRewardsSuite) pending(t *testing.T, user ibc.Wallet) sdkmath.Int {
	t.Helper()
	p, found := s.participation(t, user)
	require.True(t, found, "%s does not participate", user.KeyName())
	delta := new(big.Int).Sub(decScaled(t, s.stakingReward(t).DistributedStake), decScaled(t, p.JoinedAt))
	require.GreaterOrEqual(t, delta.Sign(), 0, "S0 of %s is above S", user.KeyName())
	return sdkmath.NewInt(mulFloor(parseInt(p.Amount).Int64(), delta))
}

func (s *stakingRewardsSuite) requirePending(t *testing.T, user ibc.Wallet, want int64) {
	t.Helper()
	require.Equal(t, strconv.FormatInt(want, 10), s.pending(t, user).String(), "pending of %s", user.KeyName())
}

func (s *stakingRewardsSuite) distributionEvents(t *testing.T, day helpers.EpochInfo) []helpers.Event {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, int64(day.CurrentEpochStartHeight), drEventPrefix+"StakingRewardDistributionEvent")
	require.NoError(t, err)
	return helpers.EventsWith(events, map[string]string{"reward_id": s.rewardID, "amount": strconv.Itoa(srPrize)})
}

func (s *stakingRewardsSuite) mustExec(t *testing.T, user ibc.Wallet, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTx(s.ctx, s.bze, user, args...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", args, res.RawLog)
	return res
}

// requireRefused asserts the tx is rejected with reason and that it changed
// neither the rewards module's balances nor the staking rewards.
func (s *stakingRewardsSuite) requireRefused(t *testing.T, reason string, user ibc.Wallet, args ...string) {
	t.Helper()
	before := s.moduleState(t)
	res, err := helpers.ExecTx(s.ctx, s.bze, user, args...)
	msg, err := helpers.RejectionReason(res, err)
	require.NoError(t, err, "tx %v", args)
	require.Contains(t, msg, reason)
	require.Equal(t, before, s.moduleState(t), "a refused tx must not change the rewards state")
}

type srModuleState struct {
	Stk, Prz string
	Rewards  []srInfo
}

func (s *stakingRewardsSuite) moduleState(t *testing.T) srModuleState {
	t.Helper()
	return srModuleState{
		Stk:     s.balance(t, s.module, s.stk).String(),
		Prz:     s.balance(t, s.module, s.prz).String(),
		Rewards: s.allStakingRewards(t),
	}
}

func (s *stakingRewardsSuite) requireTxEvent(t *testing.T, res helpers.TxResult, name string, attrs map[string]string) helpers.Event {
	t.Helper()
	events := helpers.EventsWith(helpers.TxEvents(res, drEventPrefix+name), attrs)
	require.Len(t, events, 1, "want one %s with %v in tx %s, got %+v", name, attrs, res.TxHash, helpers.TxEvents(res, drEventPrefix+name))
	return events[0]
}

func (s *stakingRewardsSuite) txFee(t *testing.T, res helpers.TxResult) sdkmath.Int {
	t.Helper()
	fee, err := helpers.TxFee(res, chain.Denom)
	require.NoError(t, err)
	return fee
}

// waitDayTick waits for the next day epoch and returns it. The tick's block
// already holds the payouts: the epoch hook queues them and the rewards
// EndBlock pays them in the same block.
func (s *stakingRewardsSuite) waitDayTick(t *testing.T) helpers.EpochInfo {
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
func (s *stakingRewardsSuite) requireSameDay(t *testing.T, want int64, what string) {
	t.Helper()
	cur, err := helpers.CurrentEpoch(s.ctx, s.bze, "day")
	require.NoError(t, err)
	require.Equal(t, want, cur, "a day tick fell during %s; the scenario needs them in one day epoch", what)
}

func (s *stakingRewardsSuite) balance(t *testing.T, addr, denom string) sdkmath.Int {
	t.Helper()
	b, err := helpers.Balance(s.ctx, s.bze, addr, denom)
	require.NoError(t, err)
	return b
}

type srParams struct {
	CreateStakingRewardFee sdk.Coin
	ExtraGasForExitStake   helpers.Int64
}

// queryParams reads `bzed q rewards params`, accepting the camelCase and the
// snake_case spelling (see denomRewardsSuite.queryParams).
func (s *stakingRewardsSuite) queryParams(t *testing.T) srParams {
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
		if ok {
			require.NoError(t, json.Unmarshal(raw, dst), "param %s", camel)
		}
	}
	var p srParams
	field(&p.CreateStakingRewardFee, "createStakingRewardFee", "create_staking_reward_fee")
	field(&p.ExtraGasForExitStake, "extraGasForExitStake", "extra_gas_for_exit_stake")
	require.False(t, p.CreateStakingRewardFee.Amount.IsNil(), "param create_staking_reward_fee missing from %v", res.Params)
	require.Positive(t, int64(p.ExtraGasForExitStake), "param extra_gas_for_exit_stake missing from %v", res.Params)
	return p
}

// scaled returns v × 10^18, an integer in the accumulator's fixed point.
func scaled(v int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(v), new(big.Int).Exp(big.NewInt(10), big.NewInt(decPrecision), nil))
}

// mulFloor returns floor(amount × d / 10^18) for a fixed-point d.
func mulFloor(amount int64, d *big.Int) int64 {
	p := new(big.Int).Mul(big.NewInt(amount), d)
	return p.Quo(p, scaled(1)).Int64()
}

// decScaled reads an 18-decimal string the chain stored (e.g. "0.25" or
// "0.250000000000000000") as an integer scaled by 10^18.
func decScaled(t *testing.T, v string) *big.Int {
	t.Helper()
	d, err := sdkmath.LegacyNewDecFromStr(v)
	require.NoError(t, err, "decimal %q", v)
	return d.BigInt()
}
