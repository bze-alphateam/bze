package interchaintest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
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
	srCreatorFunds = 250_000_000_000 // 250,000 BZE: three denoms, three staking rewards
	srUserFunds    = 1_000_000_000   // 1,000 BZE for gas
	// erin pays the create fee in a factory token: the denom, a pool whose ubze
	// side stays above MinNativeLiquidityForModuleSwap (1e11) after the fee
	// swap, and the 25,000 BZE the create checks as spendable first
	srErinFunds = 250_000_000_000

	srPrize    = 1_000_000
	srDuration = 2
	srLock     = 1

	srAliceStake = 3_000_000
	srBobStake   = 1_000_000
	srCarolStake = 3

	// the second staking reward, only to be refused deletion while unfinished
	srFreshPrize = 1_000

	// reward B (stb staked for prz): extended by dave, then joined, topped up,
	// funded by dave and exited
	srBPrize    = 1_000
	srBMinStake = 10
	srBLock     = 1
	srBAddDays  = 2
	// 999,999 / 300 does not divide: S grows by 3333.333…333 (18 decimals)
	srDistribute = 999_999
	// reward C, created by erin with the fee paid in her factory token: lock 0,
	// long enough never to finish during the suite
	srCPrize    = 1_000
	srCDuration = 30
	// reward D, created by a payer holding no ubze at all (step 12b)
	srDPrize      = 1_000
	srDDuration   = 1
	srZeroFunds   = 1_000_000 // the ubze the payer starts with and sends away
	srZeroFeeToks = 100_000_000

	srAliceJoinB  = 300
	srAliceTopUpB = 200
	srBobJoinC    = 50
	srBobJoinB    = 70
	srBobReJoinB  = 30 // between the two exits of the merged-unlock tx
	srBobJoinB2   = 40 // after the full exit
	srCarolBelow  = 5  // below B's min stake

	srMaxDuration = 365 * 100 // days, a reward's total duration
	srMaxLock     = 365 * 10  // days
	srUnknownID   = "999999"

	// dave extends B, tries to extend it past the maximum and funds it
	srDavePrz = srBPrize*srBAddDays + srBPrize*srMaxDuration + srDistribute

	// erin's fee token: 1 fee = 1000 ubze in a 0.2% pool whose fees stay with
	// the providers, so each swap adds its whole input to the input reserve
	srFeeMint    = 1_000_000_000
	srFeePoolFee = 150_000_000
	srFeePoolBze = 150_000_000_000
	srPoolFee    = "0.002"
	srPoolDest   = `{"treasury":"0","burner":"0","providers":"1"}`
	srFeeTxFee   = 1_000 // fee units, swappable (its 0.2% is 2 units)
	srFeeTxGas   = 1_000_000
	// a fee paid in another denom may buy at most this much more than the fee
	srFeeTolerancePct = 1

	srTxGas = 500_000

	// LegacyDec precision: the accumulator S has 18 decimals
	decPrecision = 18
)

// TestStakingRewards runs a staking reward's lifecycle on one network with
// fast epochs: the day-epoch payout queue in EndBlock, claims, exits into the
// hour-epoch pending unlock and the removal of the finished reward. Two more
// rewards cover the rest of the messages: B (update, distribute, top-up,
// merged unlocks, re-join) and C (created with the fee paid in a factory
// token, exited with lock 0). Subtests run in order and share state; the step
// numbers match the suite's design.
//
// Timeline (D = the day epoch right after the reward is created):
//
//	D    alice and bob join (2)
//	D+1  first payout; carol joins (3)
//	D+2  second and last payout; alice claims (4), delete refused (6),
//	     alice and bob exit (7a), a fresh reward refuses deletion (9)
//	any  create refusals (11), C (12), B extended (13), funded (14), topped
//	     up (15), exits (16), refusals (17); day ticks may fall anywhere in
//	     these steps: B and C are asserted against the state of the block
//	     before each tx, and a payout at the tx's own block is accounted for
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
	t.Run("11 create refusals", s.createRefusals)
	t.Run("12 create fee paid in a non-native denom", s.createFeeInFactoryToken)
	t.Run("12b create fee paid by a payer holding no ubze", s.createWithoutUbze)
	t.Run("13 anyone may extend a staking reward", s.update)
	t.Run("14 anyone may distribute to the stakers", s.distribute)
	t.Run("15 a top-up pays the pending first", s.topUp)
	t.Run("16 exit with lock 0, merged unlocks and re-join", s.exits)
	t.Run("17 join, claim and exit refusals", s.participantRefusals)
	t.Run("5 nothing is paid after the last payout", s.afterLastPayout)
	t.Run("8 the last exit removes the finished reward", s.lastExitRemoves)
	t.Run("7b unlock returns the stake", s.unlock)
	t.Run("10 escrow at the end", s.escrowAtEnd)
}

type stakingRewardsSuite struct {
	ctx context.Context
	bze *cosmos.CosmosChain

	creator, alice, bob, carol, dave, erin ibc.Wallet
	stk, prz, stb, fee                     string
	module, cpCollector, tradebin, txFees  string
	params                                 srParams

	// rewardB and rewardC are the rewards of steps 11 to 17
	rewardB, rewardC string

	// baseline rewards module balances, before any staking reward exists
	baseline map[string]sdkmath.Int
	// funded is all prz escrowed by staking rewards, paid all prz paid out of
	// the escrow; the module holds exactly baseline + funded - paid
	funded, paid sdkmath.Int
	// paidA is what the first reward (rewardID) paid out, for its dust check
	paidA sdkmath.Int

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
		funded:   sdkmath.ZeroInt(), paid: sdkmath.ZeroInt(), paidA: sdkmath.ZeroInt(),
		unlockAt: map[string]int64{},
	}

	s.creator = interchaintest.GetAndFundTestUsers(t, ctx, "srcreator", sdkmath.NewInt(srCreatorFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "sr", sdkmath.NewInt(srUserFunds), bze, bze, bze, bze)
	s.alice, s.bob, s.carol, s.dave = users[0], users[1], users[2], users[3]
	s.erin = interchaintest.GetAndFundTestUsers(t, ctx, "srfee", sdkmath.NewInt(srErinFunds), bze)[0]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	factory := "factory/" + s.creator.FormattedAddress() + "/"
	s.stk, s.prz, s.stb = factory+"stk", factory+"prz", factory+"stb"
	s.fee = "factory/" + s.erin.FormattedAddress() + "/fee"
	const (
		aliceStb = srAliceJoinB + srAliceTopUpB
		bobStb   = srBobJoinB + srBobReJoinB + srBobJoinB2     // C's stake comes back before B's are needed
		erinPrz  = srCPrize*srCDuration + srDPrize*srDDuration // erin forwards D's budget in step 12b
	)
	for denom, amount := range map[string]int64{
		s.stk: srAliceStake + srBobStake + srCarolStake,
		s.prz: srPrize*srDuration + srFreshPrize + srBPrize + srDavePrz + erinPrz,
		s.stb: aliceStb + bobStb + srCarolBelow,
	} {
		s.mustExec(t, s.creator, "tokenfactory", "create-denom", denom[len(factory):])
		s.mustExec(t, s.creator, "tokenfactory", "mint", coinStr(amount, denom))
	}
	for _, send := range []struct {
		to    ibc.Wallet
		coins sdk.Coins
	}{
		{s.alice, sdk.NewCoins(sdk.NewInt64Coin(s.stk, srAliceStake), sdk.NewInt64Coin(s.stb, aliceStb))},
		{s.bob, sdk.NewCoins(sdk.NewInt64Coin(s.stk, srBobStake), sdk.NewInt64Coin(s.stb, bobStb))},
		{s.carol, sdk.NewCoins(sdk.NewInt64Coin(s.stk, srCarolStake), sdk.NewInt64Coin(s.stb, srCarolBelow))},
		{s.dave, coins(srDavePrz, s.prz)},
		{s.erin, coins(erinPrz, s.prz)},
	} {
		s.mustExec(t, s.creator, "bank", "send", s.creator.FormattedAddress(), send.to.FormattedAddress(), send.coins.String())
	}

	for name, addr := range map[string]*string{
		drRewardsModule:     &s.module,
		"txfeecollector_cp": &s.cpCollector,
		"tradebin":          &s.tradebin,
		"txfeecollector":    &s.txFees,
	} {
		var err error
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	for _, denom := range []string{s.stk, s.prz, s.stb} {
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
	s.paidA = s.paidA.Add(pending)
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
		s.paidA = s.paidA.Add(pending)
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

// 11. Creates the msg server refuses (denoms without supply, a balance below
// the budget) and the ones ValidateBasic refuses in CheckTx, for free.
func (s *stakingRewardsSuite) createRefusals(t *testing.T) {
	nope := "factory/" + s.creator.FormattedAddress() + "/nope"
	create := func(prize, prizeDenom, stakingDenom, duration, lock string) []string {
		return []string{"rewards", "create-staking-reward", prize, prizeDenom, stakingDenom, duration, "1", lock}
	}
	s.requireRefused(t, "invalid staking denom", s.creator, create("1", s.prz, nope, "1", "0")...)
	s.requireRefused(t, "invalid prize denom", s.creator, create("1", nope, s.stk, "1", "0")...)
	// the creator has the create fee in ubze but one prz too few for the budget
	short := s.balance(t, s.creator.FormattedAddress(), s.prz).AddRaw(1).String()
	s.requireRefused(t, "user balance is too low", s.creator, create(short, s.prz, s.stk, "1", "0")...)

	msg := func(prize, duration, lock string) map[string]any {
		return s.rewardsMsg("MsgCreateStakingReward", s.creator,
			"prize_amount", prize, "prize_denom", s.prz, "staking_denom", s.stk,
			"duration", duration, "min_stake", "1", "lock", lock)
	}
	durationRange := fmt.Sprintf("duration should be between 1 and %d days", srMaxDuration)
	for _, c := range []struct {
		name, reason string
		msg          map[string]any
	}{
		{"prize 0", "amount should be greater than 0", msg("0", "1", "0")},
		{"duration 0", durationRange, msg("1", "0", "0")},
		{"duration above 100 years", durationRange, msg("1", strconv.Itoa(srMaxDuration+1), "0")},
		{"lock above 10 years", fmt.Sprintf("locking time should be between 0 and %d days", srMaxLock), msg("1", "1", strconv.Itoa(srMaxLock+1))},
		// min_stake is stored as a uint64
		{"min stake above uint64", "min stake must not exceed 18446744073709551615", s.rewardsMsg("MsgCreateStakingReward", s.creator,
			"prize_amount", "1", "prize_denom", s.prz, "staking_denom", s.stk,
			"duration", "1", "min_stake", "18446744073709551616", "lock", "0")},
	} {
		t.Run(c.name, func(t *testing.T) {
			s.requireRefusedMsg(t, c.reason, s.creator, c.msg)
		})
	}
}

// 12. erin pays the tx fee in her factory token, which has a deep ubze pool:
// the create fee (in ubze) is then taken in that token too. The chain takes
// the input that buys the fee from the pool (rounded up), swaps it and sends
// the whole swap output to the community-pool collector: at least the fee and
// at most srFeeTolerancePct above it; erin spends no ubze. The EndBlock then swaps the tx fee on the
// reserves the fee swap left.
func (s *stakingRewardsSuite) createFeeInFactoryToken(t *testing.T) {
	erin := s.erin.FormattedAddress()
	s.mustExec(t, s.erin, "tokenfactory", "create-denom", "fee")
	s.mustExec(t, s.erin, "tokenfactory", "mint", coinStr(srFeeMint, s.fee))
	s.mustExecWith(t, s.erin, []helpers.TxOption{helpers.WithGas(srTxGas)},
		"tradebin", "create-liquidity-pool", s.fee, chain.Denom, srPoolFee, srPoolDest, "false",
		strconv.Itoa(srFeePoolFee), strconv.Itoa(srFeePoolBze))
	poolID := s.fee + "_" + chain.Denom // "factory/…" sorts before "ubze"

	createFee := s.params.CreateStakingRewardFee.Amount
	minLiquidity := s.minNativeLiquidity(t)
	require.True(t, sdkmath.NewInt(srFeePoolBze).Sub(createFee).GT(minLiquidity),
		"the pool must stay deep after the fee swap (create fee %s, min liquidity %s)", createFee, minLiquidity)
	require.True(t, s.balance(t, s.txFees, s.fee).IsZero(), "txfeecollector holds no fee token before the tx")
	ubzeBefore, feeBefore, przBefore := s.balance(t, erin, chain.Denom), s.balance(t, erin, s.fee), s.balance(t, erin, s.prz)

	txFee := sdk.NewInt64Coin(s.fee, srFeeTxFee)
	res := s.mustExecWith(t, s.erin, []helpers.TxOption{helpers.WithFees(txFee.String()), helpers.WithGas(srFeeTxGas)},
		"rewards", "create-staking-reward", strconv.Itoa(srCPrize), s.prz, s.stb, strconv.Itoa(srCDuration), "1", "0")
	created := s.requireTxEvent(t, res, "StakingRewardCreateEvent", map[string]string{
		"prize_amount": strconv.Itoa(srCPrize), "prize_denom": s.prz, "staking_denom": s.stb, "duration": strconv.Itoa(srCDuration),
	})
	s.rewardC = created.Attributes["reward_id"]
	require.NotEmpty(t, s.rewardC)
	require.Zero(t, int64(s.stakingRewardByID(t, s.rewardC).Lock))
	budget := sdkmath.NewInt(srCPrize * srCDuration)
	s.funded = s.funded.Add(budget)

	// The price of a fee paid in another denom is the pool's, never a fixed
	// one: the chain must deliver at least the fee in ubze, and only a little
	// more (it rounds the input up), never an arbitrary amount.
	in := s.transferred(t, res, erin, s.tradebin, s.fee)
	out := s.transferred(t, res, s.tradebin, s.module, chain.Denom)
	require.True(t, in.IsPositive(), "the fee's input must go to tradebin for the swap")
	require.Equal(t, out.String(), s.transferred(t, res, s.module, s.cpCollector, chain.Denom).String(), "the whole swap output must reach the txfeecollector_cp module")
	require.True(t, out.GTE(createFee), "the community pool must get at least the create fee: %s for %s", out, createFee)
	maxOut := createFee.Add(createFee.MulRaw(srFeeTolerancePct).QuoRaw(100))
	require.True(t, out.LTE(maxOut), "the swapped create fee %s must stay within %d%% of the fee %s", out, srFeeTolerancePct, createFee)
	t.Logf("create fee %s ubze bought with %s %s; the swap gave %s ubze (%s above the fee)", createFee, in, s.fee, out, out.Sub(createFee))

	paid, err := helpers.TxFee(res, s.fee)
	require.NoError(t, err)
	require.Equal(t, txFee.Amount, paid, "tx fee in the factory token")
	require.Equal(t, ubzeBefore.String(), s.balance(t, erin, chain.Denom).String(), "erin must spend no ubze")
	require.Equal(t, feeBefore.Sub(in).Sub(txFee.Amount).String(), s.balance(t, erin, s.fee).String(), "erin pays the fee's input and the tx fee in her token")
	require.Equal(t, przBefore.Sub(budget).String(), s.balance(t, erin, s.prz).String())

	// the fee swap ran on the pool as the previous block left it
	pool := s.poolAt(t, poolID, res.Height-1)
	// the query prints the stored LegacyDec as its 18-decimal scaled integer
	poolFee := sdkmath.LegacyNewDecFromIntWithPrec(parseInt(pool.Fee), sdkmath.LegacyPrecision)
	// the providers keep the pool fee, so the input reserve grows by the whole input
	base, quote := pool.reserveBase().Add(in), pool.reserveQuote().Sub(out)
	txOut := srSwapOut(txFee.Amount, base, quote, poolFee)
	events, err := helpers.BlockEvents(s.ctx, s.bze, res.Height, tradebinEventPrefix+"SwapEvent")
	require.NoError(t, err)
	swaps := helpers.EventsWith(events, map[string]string{"creator": s.txFees, "pool_id": poolID})
	require.Len(t, swaps, 1, "the EndBlock must swap the tx fee once: %+v", events)
	require.Equal(t, txFee, eventCoin(t, swaps[0], "in"), "EndBlock swap input")
	require.Equal(t, sdk.NewCoin(chain.Denom, txOut), eventCoin(t, swaps[0], "out"), "EndBlock swap output")
	after := s.poolAt(t, poolID, res.Height)
	require.Equal(t, base.Add(txFee.Amount).String(), after.reserveBase().String(), "fee token reserve")
	require.Equal(t, quote.Sub(txOut).String(), after.reserveQuote().String(), "ubze reserve")
	s.requireEscrow(t)
}

// 12b. Paying in a fee token must not need ubze at all: gina holds only the
// fee token and D's budget, pays the tx fee and the create fee in the token
// and creates D. Her ubze balance stays 0.
func (s *stakingRewardsSuite) createWithoutUbze(t *testing.T) {
	t.Skip("BZE-185: the rewards balance pre-check requires the whole create fee in ubze")

	gina := interchaintest.GetAndFundTestUsers(t, s.ctx, "srzero", sdkmath.NewInt(srZeroFunds), s.bze)[0]
	require.NoError(t, testutil.WaitForBlocks(s.ctx, 2, s.bze))
	addr := gina.FormattedAddress()
	feeOpts := []helpers.TxOption{helpers.WithFees(coinStr(srFeeTxFee, s.fee)), helpers.WithGas(srFeeTxGas)}
	budget := coins(srDPrize*srDDuration, s.prz)
	s.mustExec(t, s.erin, "bank", "send", s.erin.FormattedAddress(), addr,
		budget.Add(sdk.NewInt64Coin(s.fee, srZeroFeeToks)).String())
	// gina sends all her ubze away, paying that tx in the fee token
	s.mustExecWith(t, gina, feeOpts, "bank", "send", addr, s.erin.FormattedAddress(), coinStr(srZeroFunds, chain.Denom))
	require.True(t, s.balance(t, addr, chain.Denom).IsZero(), "gina must hold no ubze")

	createFee := s.params.CreateStakingRewardFee.Amount
	res := s.mustExecWith(t, gina, feeOpts,
		"rewards", "create-staking-reward", strconv.Itoa(srDPrize), s.prz, s.stb, strconv.Itoa(srDDuration), "1", "0")
	created := s.requireTxEvent(t, res, "StakingRewardCreateEvent", map[string]string{"prize_amount": strconv.Itoa(srDPrize), "staking_denom": s.stb})
	require.NotEmpty(t, created.Attributes["reward_id"])
	s.funded = s.funded.Add(budget.AmountOf(s.prz))

	out := s.transferred(t, res, s.module, s.cpCollector, chain.Denom)
	require.True(t, out.GTE(createFee), "the community pool must get at least the create fee: %s for %s", out, createFee)
	require.True(t, out.LTE(createFee.Add(createFee.MulRaw(srFeeTolerancePct).QuoRaw(100))), "the swapped create fee %s must stay within %d%% of %s", out, srFeeTolerancePct, createFee)
	require.True(t, s.balance(t, addr, chain.Denom).IsZero(), "gina still holds no ubze")
	require.True(t, s.balance(t, addr, s.prz).IsZero(), "gina's budget is escrowed")
	s.requireEscrow(t)
}

// 13. Anyone may extend a reward: dave, not its creator, adds days and pays
// prize × days of escrow. B has no stakers yet, so no day tick touches it.
// An extension past 100 years in total fails in DeliverTx after the handler
// already moved the escrow: the whole tx is rolled back but the tx fee.
func (s *stakingRewardsSuite) update(t *testing.T) {
	creator := s.creator.FormattedAddress()
	przBefore := s.balance(t, creator, s.prz)
	res := s.mustExec(t, s.creator, "rewards", "create-staking-reward",
		strconv.Itoa(srBPrize), s.prz, s.stb, "1", strconv.Itoa(srBMinStake), strconv.Itoa(srBLock))
	created := s.requireTxEvent(t, res, "StakingRewardCreateEvent", map[string]string{"prize_amount": strconv.Itoa(srBPrize), "staking_denom": s.stb})
	s.rewardB = created.Attributes["reward_id"]
	require.NotEmpty(t, s.rewardB)
	s.funded = s.funded.AddRaw(srBPrize)
	require.Equal(t, przBefore.SubRaw(srBPrize).String(), s.balance(t, creator, s.prz).String())

	dave := s.dave.FormattedAddress()
	before := s.stakingRewardByID(t, s.rewardB)
	creatorPrz := s.balance(t, creator, s.prz)
	ubzeBefore, davePrz := s.balance(t, dave, chain.Denom), s.balance(t, dave, s.prz)
	res = s.mustExec(t, s.dave, "rewards", "update-staking-reward", s.rewardB, strconv.Itoa(srBAddDays))
	s.requireTxEvent(t, res, "StakingRewardUpdateEvent", map[string]string{
		"reward_id": s.rewardB, "duration": strconv.Itoa(1 + srBAddDays),
	})
	escrow := sdkmath.NewInt(srBPrize * srBAddDays)
	s.funded = s.funded.Add(escrow)
	require.Equal(t, escrow.String(), s.transferred(t, res, dave, s.module, s.prz).String(), "dave pays prize × added days")
	require.Equal(t, davePrz.Sub(escrow).String(), s.balance(t, dave, s.prz).String())
	require.Equal(t, ubzeBefore.Sub(s.txFee(t, res)).String(), s.balance(t, dave, chain.Denom).String(), "dave pays only the tx fee in ubze")
	require.Equal(t, creatorPrz.String(), s.balance(t, creator, s.prz).String(), "the creator pays nothing")
	want := before
	want.Duration = helpers.Int64(1 + srBAddDays)
	require.Equal(t, want, s.stakingRewardByID(t, s.rewardB), "only the duration changes")
	s.requireEscrow(t)

	s.requireRefusedMsg(t, "invalid duration", s.dave, s.rewardsMsg("MsgUpdateStakingReward", s.dave, "reward_id", s.rewardB, "duration", "0"))
	s.requireRefused(t, "staking reward not found", s.dave, "rewards", "update-staking-reward", srUnknownID, "1")

	// srMaxDuration more days pass ValidateBasic; the total does not
	state := s.moduleState(t)
	ubzeBefore, davePrz = s.balance(t, dave, chain.Denom), s.balance(t, dave, s.prz)
	require.True(t, davePrz.GTE(sdkmath.NewInt(srBPrize*srMaxDuration)), "dave must afford the escrow, so only the total refuses it")
	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.dave, []helpers.TxOption{helpers.WithGas(srTxGas)},
		"rewards", "update-staking-reward", s.rewardB, strconv.Itoa(srMaxDuration))
	require.NoError(t, err)
	require.NotZero(t, res.Code, "an extension past %d days in total must fail", srMaxDuration)
	require.Positive(t, res.Height, "the tx must reach DeliverTx: %s", res.RawLog)
	require.Contains(t, res.RawLog, fmt.Sprintf("the new duration exceeds the maximum allowed of %d days", srMaxDuration))
	require.Equal(t, davePrz.String(), s.balance(t, dave, s.prz).String(), "the escrow sent before the check must be rolled back")
	require.Equal(t, ubzeBefore.Sub(s.txFee(t, res)).String(), s.balance(t, dave, chain.Denom).String(), "the failed tx costs its fee")
	require.Equal(t, state, s.moduleState(t), "the failed extension must change neither the escrow nor the reward")
	s.requireEscrow(t)
}

// 14. Anyone may fund the stakers: dave, not the creator, sends prize coins
// that raise S by amount / staked, rounded down. A reward without stakers,
// an unknown reward, a zero amount and an amount above the balance are
// refused.
func (s *stakingRewardsSuite) distribute(t *testing.T) {
	res := s.mustExec(t, s.alice, "rewards", "join-staking", s.rewardB, strconv.Itoa(srAliceJoinB))
	s.requireTxEvent(t, res, "StakingRewardJoinEvent", map[string]string{
		"reward_id": s.rewardB, "address": s.alice.FormattedAddress(), "amount": strconv.Itoa(srAliceJoinB),
	})

	require.Equal(t, "0", s.stakingRewardByID(t, s.rewardC).staked().String(), "C has no stakers yet")
	s.requireRefused(t, "no stakers found", s.dave, "rewards", "distribute-staking-rewards", s.rewardC, "1")
	s.requireRefused(t, "staking reward not found", s.dave, "rewards", "distribute-staking-rewards", srUnknownID, "1")
	s.requireRefusedMsg(t, "amount should be greater than 0", s.dave,
		s.rewardsMsg("MsgDistributeStakingRewards", s.dave, "reward_id", s.rewardB, "amount", "0"))
	dave := s.dave.FormattedAddress()
	tooMuch := s.balance(t, dave, s.prz).AddRaw(1).String()
	s.requireRefused(t, "insufficient funds", s.dave, "rewards", "distribute-staking-rewards", s.rewardB, tooMuch)

	ubzeBefore, przBefore := s.balance(t, dave, chain.Denom), s.balance(t, dave, s.prz)
	res = s.mustExec(t, s.dave, "rewards", "distribute-staking-rewards", s.rewardB, strconv.Itoa(srDistribute))
	s.requireTxEvent(t, res, "StakingRewardDistributionEvent", map[string]string{"reward_id": s.rewardB, "amount": strconv.Itoa(srDistribute)})
	s.funded = s.funded.AddRaw(srDistribute)
	require.Equal(t, strconv.Itoa(srDistribute), s.transferred(t, res, dave, s.module, s.prz).String())
	require.Equal(t, przBefore.SubRaw(srDistribute).String(), s.balance(t, dave, s.prz).String(), "dave pays the distribution")
	require.Equal(t, ubzeBefore.Sub(s.txFee(t, res)).String(), s.balance(t, dave, chain.Denom).String())

	pre, post := s.stakingRewardAt(t, s.rewardB, res.Height-1), s.stakingRewardAt(t, s.rewardB, res.Height)
	require.Equal(t, strconv.Itoa(srAliceJoinB), pre.StakedAmount)
	// S += QuoTruncate(999,999 / 300); a day tick in the tx's block pays the
	// daily prize on top, in EndBlock
	wantS := new(big.Int).Add(decScaled(t, pre.DistributedStake), srDeltaS(sdkmath.NewInt(srDistribute), pre.staked()))
	if post.Payouts > pre.Payouts {
		require.Equal(t, int64(pre.Payouts)+1, int64(post.Payouts))
		wantS.Add(wantS, srDeltaS(parseInt(pre.PrizeAmount), pre.staked()))
		t.Logf("a day tick paid B in the distribution's block")
	}
	require.Equal(t, wantS.String(), decScaled(t, post.DistributedStake).String(), "S must grow by amount / staked, rounded down (%s)", post.DistributedStake)
	s.requireEscrow(t)
}

// 15. A participant's second join pays what is pending first, exactly, then
// restarts its S0 at the reward's S and adds the stake.
func (s *stakingRewardsSuite) topUp(t *testing.T) {
	alice := s.alice.FormattedAddress()
	przBefore, stbBefore := s.balance(t, alice, s.prz), s.balance(t, alice, s.stb)
	res := s.mustExec(t, s.alice, "rewards", "join-staking", s.rewardB, strconv.Itoa(srAliceTopUpB))
	s.requireTxEvent(t, res, "StakingRewardJoinEvent", map[string]string{"reward_id": s.rewardB, "address": alice, "amount": strconv.Itoa(srAliceTopUpB)})

	pre := s.stakingRewardAt(t, s.rewardB, res.Height-1)
	was, found := s.participationAt(t, s.alice, s.rewardB, res.Height-1)
	require.True(t, found)
	require.Equal(t, strconv.Itoa(srAliceJoinB), was.Amount)
	pending := srPending(t, was, pre)
	// at least the distribution: floor(300 × 3333.333…333) = 999,999
	require.True(t, pending.GTE(sdkmath.NewInt(srDistribute)), "pending %s", pending)
	s.paid = s.paid.Add(pending)
	require.Equal(t, pending.String(), s.transferred(t, res, s.module, alice, s.prz).String(), "the top-up pays the pending first")
	require.Equal(t, przBefore.Add(pending).String(), s.balance(t, alice, s.prz).String())
	require.Equal(t, stbBefore.SubRaw(srAliceTopUpB).String(), s.balance(t, alice, s.stb).String())

	p, found := s.participationIn(t, s.alice, s.rewardB)
	require.True(t, found)
	require.Equal(t, strconv.Itoa(srAliceJoinB+srAliceTopUpB), p.Amount, "the stake grows by the top-up")
	require.Equal(t, pre.DistributedStake, p.JoinedAt, "S0 restarts at the reward's S")
	require.Equal(t, pre.staked().AddRaw(srAliceTopUpB).String(), s.stakingRewardAt(t, s.rewardB, res.Height).StakedAmount)
	s.requireEscrow(t)
}

// 16. An exit from a reward with lock 0 pays the pending and returns the
// stake in the exit tx itself. Two exits of one user from a locked reward in
// one hour epoch (one tx: exit, join, exit) merge into a single pending
// unlock. A join after a full exit starts a fresh participation.
func (s *stakingRewardsSuite) exits(t *testing.T) {
	bob := s.bob.FormattedAddress()
	t.Run("lock 0", func(t *testing.T) {
		s.mustExec(t, s.bob, "rewards", "join-staking", s.rewardC, strconv.Itoa(srBobJoinC))
		przBefore, stbBefore := s.balance(t, bob, s.prz), s.balance(t, bob, s.stb)

		res := s.mustExec(t, s.bob, "rewards", "exit-staking", s.rewardC)
		s.requireTxEvent(t, res, "StakingRewardExitEvent", map[string]string{"reward_id": s.rewardC, "address": bob})
		pre := s.stakingRewardAt(t, s.rewardC, res.Height-1)
		was, found := s.participationAt(t, s.bob, s.rewardC, res.Height-1)
		require.True(t, found)
		pending := srPending(t, was, pre)
		s.paid = s.paid.Add(pending)
		require.Equal(t, pending.String(), s.transferred(t, res, s.module, bob, s.prz).String(), "the exit pays the pending")
		require.Equal(t, przBefore.Add(pending).String(), s.balance(t, bob, s.prz).String())
		require.Equal(t, strconv.Itoa(srBobJoinC), s.transferred(t, res, s.module, bob, s.stb).String(), "lock 0 returns the stake in the exit tx")
		require.Equal(t, stbBefore.AddRaw(srBobJoinC).String(), s.balance(t, bob, s.stb).String())
		require.Empty(t, s.unlockEntries(t, s.rewardC, s.bob), "lock 0 queues no pending unlock")
		_, found = s.participationIn(t, s.bob, s.rewardC)
		require.False(t, found)
		// C has payouts left, so its last exit does not remove it
		require.Equal(t, "0", s.stakingRewardByID(t, s.rewardC).StakedAmount)
		s.requireEscrow(t)
	})

	t.Run("two exits in one hour epoch merge", func(t *testing.T) {
		s.mustExec(t, s.bob, "rewards", "join-staking", s.rewardB, strconv.Itoa(srBobJoinB))
		przBefore, stbBefore := s.balance(t, bob, s.prz), s.balance(t, bob, s.stb)

		exit := s.rewardsMsg("MsgExitStaking", s.bob, "reward_id", s.rewardB)
		join := s.rewardsMsg("MsgJoinStaking", s.bob, "reward_id", s.rewardB, "amount", strconv.Itoa(srBobReJoinB))
		// each exit charges extra_gas_for_exit_stake on top of the three messages
		gas := 2*uint64(s.params.ExtraGasForExitStake) + 2*srTxGas
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.bob, []helpers.TxOption{helpers.WithGas(gas)}, exit, join, exit)
		require.NoError(t, err)
		require.Zero(t, res.Code, res.RawLog)
		exits := helpers.EventsWith(helpers.TxEvents(res, drEventPrefix+"StakingRewardExitEvent"), map[string]string{"reward_id": s.rewardB, "address": bob})
		require.Len(t, exits, 2, "two exits in the tx")
		s.requireTxEvent(t, res, "StakingRewardJoinEvent", map[string]string{"reward_id": s.rewardB, "address": bob, "amount": strconv.Itoa(srBobReJoinB)})

		pre := s.stakingRewardAt(t, s.rewardB, res.Height-1)
		was, found := s.participationAt(t, s.bob, s.rewardB, res.Height-1)
		require.True(t, found)
		require.Equal(t, strconv.Itoa(srBobJoinB), was.Amount)
		// the first exit pays the pending, the second one nothing (S0 = S)
		pending := srPending(t, was, pre)
		s.paid = s.paid.Add(pending)
		require.Equal(t, pending.String(), s.transferred(t, res, s.module, bob, s.prz).String())
		require.Equal(t, przBefore.Add(pending).String(), s.balance(t, bob, s.prz).String())
		require.True(t, s.transferred(t, res, s.module, bob, s.stb).IsZero(), "both stakes stay locked")
		require.Equal(t, stbBefore.SubRaw(srBobReJoinB).String(), s.balance(t, bob, s.stb).String())
		_, found = s.participationIn(t, s.bob, s.rewardB)
		require.False(t, found)
		require.Equal(t, pre.staked().SubRaw(srBobJoinB).String(), s.stakingRewardAt(t, s.rewardB, res.Height).StakedAmount)

		entries := s.unlockEntries(t, s.rewardB, s.bob)
		require.Len(t, entries, 1, "both exits must share one pending unlock")
		unlockAt := s.hourAt(t, res.Height) + srBLock*24
		require.Equal(t, fmt.Sprintf("%d/%s/%s", unlockAt, s.rewardB, bob), entries[0].Index)
		require.Equal(t, s.stb, entries[0].Denom)
		require.Equal(t, strconv.Itoa(srBobJoinB+srBobReJoinB), entries[0].Amount, "the merged unlock holds both stakes")
		s.requireEscrow(t)
	})

	t.Run("re-join after a full exit", func(t *testing.T) {
		unlock := s.unlockEntries(t, s.rewardB, s.bob)
		res := s.mustExec(t, s.bob, "rewards", "join-staking", s.rewardB, strconv.Itoa(srBobJoinB2))
		s.requireTxEvent(t, res, "StakingRewardJoinEvent", map[string]string{"reward_id": s.rewardB, "address": bob, "amount": strconv.Itoa(srBobJoinB2)})
		require.True(t, s.transferred(t, res, s.module, bob, s.prz).IsZero(), "a fresh participant has nothing pending")

		pre := s.stakingRewardAt(t, s.rewardB, res.Height-1)
		p, found := s.participationIn(t, s.bob, s.rewardB)
		require.True(t, found)
		require.Equal(t, strconv.Itoa(srBobJoinB2), p.Amount, "a fresh participation, not added to the locked stakes")
		require.Equal(t, pre.DistributedStake, p.JoinedAt, "S0 is the reward's S at the join")
		require.Equal(t, pre.staked().AddRaw(srBobJoinB2).String(), s.stakingRewardAt(t, s.rewardB, res.Height).StakedAmount)
		require.Equal(t, unlock, s.unlockEntries(t, s.rewardB, s.bob), "the re-join leaves the pending unlock alone")
		s.requireEscrow(t)
	})
}

// 17. A join below the min stake, a claim or exit by a non-participant and a
// claim or exit of an unknown reward are refused.
func (s *stakingRewardsSuite) participantRefusals(t *testing.T) {
	_, found := s.participationIn(t, s.carol, s.rewardB)
	require.False(t, found)
	s.requireRefused(t, "amount is smaller than staking reward min stake", s.carol,
		"rewards", "join-staking", s.rewardB, strconv.Itoa(srCarolBelow))
	_, found = s.participationIn(t, s.dave, s.rewardB)
	require.False(t, found)
	for _, cmd := range []string{"claim-staking-rewards", "exit-staking"} {
		s.requireRefused(t, "you are not a participant in this staking reward", s.dave, "rewards", cmd, s.rewardB)
		s.requireRefused(t, "reward with provided id not found", s.dave, "rewards", cmd, srUnknownID)
	}
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
	s.paidA = s.paidA.Add(pending)
	require.Equal(t, przBefore.Add(pending).String(), s.balance(t, carol, s.prz).String())
	s.requireQueuedUnlock(t, s.carol, res.Height, sdkmath.NewInt(srCarolStake))

	_, err := helpers.Query[json.RawMessage](s.ctx, s.bze, "rewards", "get-staking-reward", s.rewardID)
	require.Error(t, err, "the finished reward must be gone")
	for _, sr := range s.allStakingRewards(t) {
		require.NotEqual(t, s.rewardID, sr.RewardID, "the finished reward must be gone from the list")
	}
	s.requireRefused(t, "invalid reward_id", s.dave, "rewards", "delete-staking-reward", s.rewardID)

	// what the rounding left behind stays in the module: the reward's whole
	// budget minus everything it paid
	dust := sdkmath.NewInt(srPrize * srDuration).Sub(s.paidA)
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
	rewards := map[string]srInfo{}
	for _, sr := range s.allStakingRewards(t) {
		require.Equal(t, s.prz, sr.PrizeDenom, "every reward of the suite pays prz")
		rewards[sr.RewardID] = sr
		unpaid := int64(sr.Duration) - int64(sr.Payouts)
		owed = owed.Add(parseInt(sr.PrizeAmount).MulRaw(unpaid))
	}
	for _, user := range []ibc.Wallet{s.alice, s.bob, s.carol} {
		for _, p := range s.participationsAt(t, user, 0) {
			if sr, ok := rewards[p.RewardID]; ok {
				owed = owed.Add(srPending(t, p, sr))
			}
		}
	}
	require.True(t, escrow.GTE(owed), "the escrow %s must cover the unpaid budget and pending claims %s", escrow, owed)
}

func (s *stakingRewardsSuite) requireQueuedUnlock(t *testing.T, user ibc.Wallet, height int64, amount sdkmath.Int) {
	t.Helper()
	s.unlockAt[user.KeyName()] = s.hourAt(t, height) + srLock*24

	entry, found := s.pendingUnlock(t, s.unlockIndex(user))
	require.True(t, found, "no pending unlock %s", s.unlockIndex(user))
	require.Equal(t, user.FormattedAddress(), entry.Address)
	require.Equal(t, s.stk, entry.Denom)
	require.Equal(t, amount.String(), entry.Amount)
}

func (s *stakingRewardsSuite) unlockIndex(user ibc.Wallet) string {
	return fmt.Sprintf("%d/%s/%s", s.unlockAt[user.KeyName()], s.rewardID, user.FormattedAddress())
}

// hourAt returns the hour epoch at a height: the epoch a tx included there
// saw, since the epochs BeginBlock runs before the txs.
func (s *stakingRewardsSuite) hourAt(t *testing.T, height int64) int64 {
	t.Helper()
	hour, err := helpers.Query[struct {
		CurrentEpoch helpers.Int64 `json:"current_epoch"`
	}](s.ctx, s.bze, "epochs", "identifier", "hour", "--height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	return int64(hour.CurrentEpoch)
}

func (s *stakingRewardsSuite) pendingUnlock(t *testing.T, index string) (pendingUnlockInfo, bool) {
	t.Helper()
	for _, e := range s.pendingUnlocks(t) {
		if e.Index == index {
			return e, true
		}
	}
	return pendingUnlockInfo{}, false
}

// unlockEntries returns the user's pending unlocks from one reward, whatever
// their epoch (the index is <epoch>/<reward id>/<address>).
func (s *stakingRewardsSuite) unlockEntries(t *testing.T, rewardID string, user ibc.Wallet) []pendingUnlockInfo {
	t.Helper()
	var out []pendingUnlockInfo
	for _, e := range s.pendingUnlocks(t) {
		if strings.HasSuffix(e.Index, "/"+rewardID+"/"+user.FormattedAddress()) {
			out = append(out, e)
		}
	}
	return out
}

func (s *stakingRewardsSuite) pendingUnlocks(t *testing.T) []pendingUnlockInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		List []pendingUnlockInfo `json:"list"`
	}](s.ctx, s.bze, "rewards", "all-pending-unlock-participants")
	require.NoError(t, err)
	return res.List
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
	return s.stakingRewardAt(t, id, 0)
}

// stakingRewardAt reads a reward at a height (0 for the latest). At a tx's
// height - 1 it is the reward the tx saw: payouts only happen in EndBlock.
func (s *stakingRewardsSuite) stakingRewardAt(t *testing.T, id string, height int64) srInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		StakingReward srInfo `json:"staking_reward"`
	}](s.ctx, s.bze, append([]string{"rewards", "get-staking-reward", id}, srHeightFlag(height)...)...)
	require.NoError(t, err)
	require.Equal(t, id, res.StakingReward.RewardID)
	return res.StakingReward
}

func srHeightFlag(height int64) []string {
	if height <= 0 {
		return nil
	}
	return []string{"--height", strconv.FormatInt(height, 10)}
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
	return s.participationIn(t, user, s.rewardID)
}

func (s *stakingRewardsSuite) participationIn(t *testing.T, user ibc.Wallet, rewardID string) (srParticipant, bool) {
	t.Helper()
	return s.participationAt(t, user, rewardID, 0)
}

// participationAt returns the user's participation in a reward at a height
// (0 for the latest).
func (s *stakingRewardsSuite) participationAt(t *testing.T, user ibc.Wallet, rewardID string, height int64) (srParticipant, bool) {
	t.Helper()
	for _, p := range s.participationsAt(t, user, height) {
		if p.RewardID == rewardID {
			return p, true
		}
	}
	return srParticipant{}, false
}

func (s *stakingRewardsSuite) participationsAt(t *testing.T, user ibc.Wallet, height int64) []srParticipant {
	t.Helper()
	res, err := helpers.Query[struct {
		List []srParticipant `json:"list"`
	}](s.ctx, s.bze, append([]string{"rewards", "staking-reward-participant", user.FormattedAddress()}, srHeightFlag(height)...)...)
	require.NoError(t, err)
	return res.List
}

// pending is what the user can claim now: floor(stake × (S − S0)). The chain
// has no pending query for staking rewards, so it is derived from the stored
// stake, the user's S0 and the reward's S.
func (s *stakingRewardsSuite) pending(t *testing.T, user ibc.Wallet) sdkmath.Int {
	t.Helper()
	p, found := s.participation(t, user)
	require.True(t, found, "%s does not participate", user.KeyName())
	return srPending(t, p, s.stakingReward(t))
}

// srPending is floor(stake × (S − S0)) of a participation in sr, like the
// keeper's claimPending.
func srPending(t *testing.T, p srParticipant, sr srInfo) sdkmath.Int {
	t.Helper()
	require.Equal(t, sr.RewardID, p.RewardID)
	delta := new(big.Int).Sub(decScaled(t, sr.DistributedStake), decScaled(t, p.JoinedAt))
	require.GreaterOrEqual(t, delta.Sign(), 0, "S0 of a participant of %s is above S", sr.RewardID)
	return sdkmath.NewInt(mulFloor(parseInt(p.Amount).Int64(), delta))
}

// srDeltaS is what a distribution of amount adds to S, scaled by 10^18:
// amount / staked rounded down at 18 decimals (QuoTruncate).
func srDeltaS(amount, staked sdkmath.Int) *big.Int {
	return sdkmath.LegacyNewDecFromInt(amount).QuoTruncate(sdkmath.LegacyNewDecFromInt(staked)).BigInt()
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
	return s.mustExecWith(t, user, nil, args...)
}

func (s *stakingRewardsSuite) mustExecWith(t *testing.T, user ibc.Wallet, opts []helpers.TxOption, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(s.ctx, s.bze, user, opts, args...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", args, res.RawLog)
	return res
}

// requireRefused asserts the tx is rejected with reason and that it changed
// neither the rewards module's balances nor the staking rewards.
func (s *stakingRewardsSuite) requireRefused(t *testing.T, reason string, user ibc.Wallet, args ...string) {
	t.Helper()
	s.refuse(t, reason, args, func() (helpers.TxResult, error) {
		return helpers.ExecTx(s.ctx, s.bze, user, args...)
	})
}

// requireRefusedMsg asserts ValidateBasic refuses msg: the tx is signed
// without the CLI (which would run ValidateBasic itself) and CheckTx rejects
// it, so it is never included and costs the signer nothing.
func (s *stakingRewardsSuite) requireRefusedMsg(t *testing.T, reason string, user ibc.Wallet, msg map[string]any) {
	t.Helper()
	before := s.userCoins(t, user)
	res := s.refuse(t, reason, msg, func() (helpers.TxResult, error) {
		return helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(srTxGas)}, msg)
	})
	require.Zero(t, res.Height, "ValidateBasic must refuse %v in CheckTx", msg["@type"])
	require.Equal(t, before, s.userCoins(t, user), "a tx refused in CheckTx costs nothing")
}

// refuse sends a tx that must be refused with reason and asserts the rewards
// state is unchanged. Rewards with stakers change on every day tick, so a
// refusal that straddles one is sent once more: a refused tx has no effect
// to repeat.
func (s *stakingRewardsSuite) refuse(t *testing.T, reason string, what any, send func() (helpers.TxResult, error)) helpers.TxResult {
	t.Helper()
	for attempt := 1; ; attempt++ {
		day := s.currentDay(t)
		before := s.moduleState(t)
		res, err := send()
		msg, err := helpers.RejectionReason(res, err)
		require.NoError(t, err, "tx %v", what)
		require.Contains(t, msg, reason)
		after := s.moduleState(t)
		if attempt == 1 && s.currentDay(t) != day {
			t.Logf("a day tick fell during the refusal of %v; sending it again", what)
			continue
		}
		require.Equal(t, before, after, "a refused tx must not change the rewards state")
		return res
	}
}

type srModuleState struct {
	Stk, Prz, Stb string
	Rewards       []srInfo
}

func (s *stakingRewardsSuite) moduleState(t *testing.T) srModuleState {
	t.Helper()
	return srModuleState{
		Stk:     s.balance(t, s.module, s.stk).String(),
		Prz:     s.balance(t, s.module, s.prz).String(),
		Stb:     s.balance(t, s.module, s.stb).String(),
		Rewards: s.allStakingRewards(t),
	}
}

// userCoins returns the user's balances of the suite's denoms.
func (s *stakingRewardsSuite) userCoins(t *testing.T, user ibc.Wallet) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, denom := range []string{chain.Denom, s.stk, s.prz, s.stb} {
		out[denom] = s.balance(t, user.FormattedAddress(), denom).String()
	}
	return out
}

func (s *stakingRewardsSuite) currentDay(t *testing.T) int64 {
	t.Helper()
	day, err := helpers.CurrentEpoch(s.ctx, s.bze, "day")
	require.NoError(t, err)
	return day
}

// rewardsMsg is a bze.rewards message signed by user, for helpers.ExecMsgs;
// fields are proto field name / value pairs.
func (s *stakingRewardsSuite) rewardsMsg(typ string, user ibc.Wallet, fields ...string) map[string]any {
	msg := map[string]any{"@type": "/bze.rewards." + typ, "creator": user.FormattedAddress()}
	for i := 0; i+1 < len(fields); i += 2 {
		msg[fields[i]] = fields[i+1]
	}
	return msg
}

func (s *stakingRewardsSuite) transferred(t *testing.T, res helpers.TxResult, from, to, denom string) sdkmath.Int {
	t.Helper()
	got, err := helpers.Transferred(helpers.TxEvents(res, "transfer"), from, to, denom)
	require.NoError(t, err)
	return got
}

// poolAt reads a liquidity pool at a height.
func (s *stakingRewardsSuite) poolAt(t *testing.T, id string, height int64) poolInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		Pool poolInfo `json:"pool"`
	}](s.ctx, s.bze, append([]string{"tradebin", "liquidity-pool", id}, srHeightFlag(height)...)...)
	require.NoError(t, err)
	require.Equal(t, id, res.Pool.ID, "pool %s not found", id)
	return res.Pool
}

func (s *stakingRewardsSuite) minNativeLiquidity(t *testing.T) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Params struct {
			MinNativeLiquidityForModuleSwap string `json:"minNativeLiquidityForModuleSwap"`
		} `json:"params"`
	}](s.ctx, s.bze, "tradebin", "params")
	require.NoError(t, err)
	minLiquidity := parseInt(res.Params.MinNativeLiquidityForModuleSwap)
	require.True(t, minLiquidity.IsPositive(), "tradebin param minNativeLiquidityForModuleSwap missing")
	return minLiquidity
}

// srSwapOut is tradebin's swapTokens output: the pool fee ⌊in × fee⌋ comes
// off the input and out = ⌊outReserve × net / (inReserve + net)⌋.
func srSwapOut(in, inReserve, outReserve sdkmath.Int, fee sdkmath.LegacyDec) sdkmath.Int {
	net := in.Sub(sdkmath.LegacyNewDecFromInt(in).Mul(fee).TruncateInt())
	return sdkmath.LegacyNewDecFromInt(outReserve.Mul(net)).Quo(sdkmath.LegacyNewDecFromInt(inReserve.Add(net))).TruncateInt()
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
