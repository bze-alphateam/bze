package keeper_test

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// bankLedger records every coin flow crossing the module account, so a test
// can assert escrow conservation (I5) instead of pinning individual transfers.
type bankLedger struct {
	in  sdk.Coins // user -> module (escrows, stakes)
	out sdk.Coins // module -> user (payouts, stake returns)
}

// recordBankFlows replaces strict per-call expectations with recorders: any
// transfer succeeds and is added to the ledger. Balances are unlimited so the
// scenario, not the mock, decides what happens.
func recordBankFlows(bank *testutil.MockBankKeeper, epoch *testutil.MockEpochKeeper) *bankLedger {
	ledger := &bankLedger{}
	rich := sdk.NewCoins(
		sdk.NewCoin("ubze", math.NewInt(1_000_000_000_000)),
		sdk.NewCoin("uboost", math.NewInt(1_000_000_000_000)),
		sdk.NewCoin("uvdl", math.NewInt(1_000_000_000_000)),
		sdk.NewCoin("uprize", math.NewInt(1_000_000_000_000)),
	)

	bank.EXPECT().HasSupply(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
	bank.EXPECT().SpendableCoins(gomock.Any(), gomock.Any()).Return(rich).AnyTimes()
	bank.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ sdk.AccAddress, _ string, amt sdk.Coins) error {
			ledger.in = ledger.in.Add(amt...)
			return nil
		}).
		AnyTimes()
	bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ sdk.AccAddress, amt sdk.Coins) error {
			ledger.out = ledger.out.Add(amt...)
			return nil
		}).
		AnyTimes()
	epoch.EXPECT().SafeGetEpochCountByIdentifier(gomock.Any(), gomock.Any()).Return(int64(100), nil).AnyTimes()

	return ledger
}

// seedLifecycleReward stores a fresh unlocked reward whose prize denom is
// distinct from both the staking denom and every boost denom, so each ledger
// denom isolates one flow: ubze = stakes, uprize = base payouts, boost denoms
// = boost escrow/payouts.
func (suite *IntegrationTestSuite) seedLifecycleReward(rewardId string, duration uint32) {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         rewardId,
		PrizeAmount:      "1000",
		PrizeDenom:       "uprize",
		StakingDenom:     "ubze",
		Duration:         duration,
		Payouts:          0,
		StakedAmount:     "0",
		DistributedStake: "0",
	})
}

func (suite *IntegrationTestSuite) joinStaking(addr sdk.AccAddress, rewardId string, amount int64) {
	_, err := suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{
		Creator: addr.String(), RewardId: rewardId, Amount: math.NewInt(amount).String(),
	})
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) exitStaking(addr sdk.AccAddress, rewardId string) {
	_, err := suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{
		Creator: addr.String(), RewardId: rewardId,
	})
	suite.Require().NoError(err)
}

type lifecycleBoost struct {
	denom      string
	daily      int64
	days       uint32
	extendDays uint32 // 0 = no extension; applied after phase 1
}

type lifecycleCase struct {
	name     string
	stakes   []int64 // per-participant join amount (varying T)
	topUp    int64   // participant 0 tops up after phase 1 (0 = none)
	midClaim bool    // last participant claims after phase 1
	boosts   []lifecycleBoost
	phase1   int // distributed days before the gap
	gap      int // zero-staker days (A6): nothing may advance
	phase2   int // distributed days after re-join; phase1+phase2 = parent duration
}

// runBoostLifecycle drives a complete lifecycle through the public msg server
// and queue entry points and asserts the cross-cutting invariants:
// I5 (escrow conservation, residue is truncation dust only), I3/A6 (every
// boost fully emitted despite zero-staker gaps), I4 (last exit of the
// finished parent deletes all boost state) and A7 (every participant paid via
// their exit — implied by conservation with all stores empty).
func (suite *IntegrationTestSuite) runBoostLifecycle(tc lifecycleCase) {
	ledger := recordBankFlows(suite.bank, suite.epoch)
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)

	rewardId := "000000000001"
	duration := uint32(tc.phase1 + tc.phase2)
	suite.seedLifecycleReward(rewardId, duration)

	participants := make([]sdk.AccAddress, len(tc.stakes))
	for i := range tc.stakes {
		participants[i] = sdk.AccAddress(fmt.Sprintf("lifecycle-addr-%02d", i))
	}
	for i, amount := range tc.stakes {
		suite.joinStaking(participants[i], rewardId, amount)
	}

	booster := sdk.AccAddress("booster")
	for _, b := range tc.boosts {
		_, err := suite.msgServer.CreateBoost(suite.ctx, &types.MsgCreateBoost{
			Creator:     booster.String(),
			RewardId:    rewardId,
			Denom:       b.denom,
			DailyAmount: math.NewInt(b.daily).String(),
			Days:        fmt.Sprintf("%d", b.days),
		})
		suite.Require().NoError(err)
	}

	for day := 0; day < tc.phase1; day++ {
		suite.distributeOneDay()
	}

	// each settle moment truncates < 1 unit per boost — count them for the
	// dust bound
	settleOps := 0
	if tc.topUp > 0 {
		suite.joinStaking(participants[0], rewardId, tc.topUp)
		settleOps++
	}
	if tc.midClaim {
		last := participants[len(participants)-1]
		_, err := suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{
			Creator: last.String(), RewardId: rewardId,
		})
		suite.Require().NoError(err)
		settleOps++
	}
	for i, b := range tc.boosts {
		if b.extendDays == 0 {
			continue
		}
		_, err := suite.msgServer.UpdateBoost(suite.ctx, &types.MsgUpdateBoost{
			Creator:  booster.String(),
			RewardId: rewardId,
			BoostId:  fmt.Sprintf("%012d", i+1),
			Days:     fmt.Sprintf("%d", b.extendDays),
		})
		suite.Require().NoError(err)
	}

	// zero-staker gap (A6): everyone exits; skipped days consume nothing on
	// the parent or any boost
	for i := range participants {
		suite.exitStaking(participants[i], rewardId)
		settleOps++
	}
	srBeforeGap, found := suite.k.GetStakingReward(suite.ctx, rewardId)
	suite.Require().True(found)
	boostsBeforeGap := suite.k.GetRewardBoosts(suite.ctx, rewardId)
	for day := 0; day < tc.gap; day++ {
		suite.distributeOneDay()
	}
	srAfterGap, found := suite.k.GetStakingReward(suite.ctx, rewardId)
	suite.Require().True(found)
	suite.Require().Equal(srBeforeGap.Payouts, srAfterGap.Payouts)
	suite.Require().Equal(boostsBeforeGap, suite.k.GetRewardBoosts(suite.ctx, rewardId))

	for i, amount := range tc.stakes {
		suite.joinStaking(participants[i], rewardId, amount)
	}
	for day := 0; day < tc.phase2; day++ {
		suite.distributeOneDay()
	}

	// I3: the parent finished and every boost is fully emitted on or before
	// its last payout — the gap stranded nothing
	finalSr, found := suite.k.GetStakingReward(suite.ctx, rewardId)
	suite.Require().True(found)
	suite.Require().Equal(duration, finalSr.Payouts)
	for _, boost := range suite.k.GetRewardBoosts(suite.ctx, rewardId) {
		suite.Require().Equal(boost.Duration, boost.Payouts, "boost %s not fully emitted", boost.Id)
	}

	for i := range participants {
		suite.exitStaking(participants[i], rewardId)
		settleOps++
	}

	// I4: the last exit removed the finished parent and every piece of boost
	// state with it
	_, found = suite.k.GetStakingReward(suite.ctx, rewardId)
	suite.Require().False(found)
	suite.Require().Empty(suite.k.GetAllBoosts(suite.ctx))
	suite.Require().Empty(suite.k.GetAllBoostParticipant(suite.ctx))
	suite.Require().Empty(suite.k.GetAllStakingRewardParticipant(suite.ctx))

	// I5 per boost denom: escrow captured exactly, and everything escrowed
	// left again except truncation-level dust
	expectedEscrow := map[string]int64{}
	boostsPerDenom := map[string]int64{}
	for _, b := range tc.boosts {
		expectedEscrow[b.denom] += b.daily * int64(b.days+b.extendDays)
		boostsPerDenom[b.denom]++
	}
	for denom, escrowed := range expectedEscrow {
		in := ledger.in.AmountOf(denom)
		suite.Require().True(in.Equal(math.NewInt(escrowed)), "denom %s: escrowed %s, expected %d", denom, in, escrowed)

		residue := in.Sub(ledger.out.AmountOf(denom))
		dustBound := math.NewInt(int64(settleOps) * boostsPerDenom[denom])
		suite.Require().True(residue.GTE(math.ZeroInt()), "denom %s: paid out more than escrowed (residue %s)", denom, residue)
		suite.Require().True(residue.LTE(dustBound), "denom %s: residue %s exceeds truncation dust bound %s", denom, residue, dustBound)
	}

	// stakes conserve exactly: everything staked came back
	suite.Require().True(ledger.in.AmountOf("ubze").Equal(ledger.out.AmountOf("ubze")))
}

// TestBoostSecurity_EscrowConservationLifecycle is the table-driven I5 suite:
// full lifecycles with varying T sizes, boost counts (incl. two same-denom and
// one extended mid-run), day counts, top-ups, mid-run claims, zero-staker
// gaps and parent removal.
func (suite *IntegrationTestSuite) TestBoostSecurity_EscrowConservationLifecycle() {
	cases := []lifecycleCase{
		{
			name:   "single boost, one staker, exact division",
			stakes: []int64{100},
			boosts: []lifecycleBoost{{denom: "uboost", daily: 1000, days: 4}},
			phase1: 2, gap: 2, phase2: 2,
		},
		{
			name:     "same-denom pair plus extended boost, top-up and mid claim",
			stakes:   []int64{250, 1000, 5},
			topUp:    750,
			midClaim: true,
			boosts: []lifecycleBoost{
				{denom: "uboost", daily: 1000, days: 3},
				{denom: "uboost", daily: 500, days: 6},
				{denom: "uvdl", daily: 777, days: 3, extendDays: 3},
			},
			phase1: 3, gap: 1, phase2: 3,
		},
		{
			name:     "dust staker with heavy truncation",
			stakes:   []int64{1, 999999},
			midClaim: true,
			boosts:   []lifecycleBoost{{denom: "uboost", daily: 10, days: 8}},
			phase1:   4, gap: 3, phase2: 4,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.runBoostLifecycle(tc)
		})
	}
}

// TestBoostSecurity_GenesisRoundTripUnderLoad exports mid-lifecycle (active +
// finished boosts, a dormant participant with absent entries, a stamped
// late joiner), imports into a fresh chain, and runs an identical
// continuation on both: every outcome must match the unexported control run.
func (suite *IntegrationTestSuite) TestBoostSecurity_GenesisRoundTripUnderLoad() {
	ledger := recordBankFlows(suite.bank, suite.epoch)
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)

	rewardId := "000000000001"
	suite.seedLifecycleReward(rewardId, 6)

	dormant := sdk.AccAddress("dormant.")
	late := sdk.AccAddress("latecomer")
	booster := sdk.AccAddress("booster")

	// dormant joins BEFORE the boosts exist: absent entries, S0 = 0
	suite.joinStaking(dormant, rewardId, 100)

	for _, b := range []types.MsgCreateBoost{
		{Creator: booster.String(), RewardId: rewardId, Denom: "uboost", DailyAmount: "1000", Days: "6"},
		{Creator: booster.String(), RewardId: rewardId, Denom: "uvdl", DailyAmount: "300", Days: "2"},
	} {
		msg := b
		_, err := suite.msgServer.CreateBoost(suite.ctx, &msg)
		suite.Require().NoError(err)
	}

	// two days: the uvdl boost finishes, the uboost one stays active
	suite.distributeOneDay()
	suite.distributeOneDay()

	// the late joiner is stamped at the current accumulators (incl. the
	// finished boost's final value)
	suite.joinStaking(late, rewardId, 400)
	suite.distributeOneDay()

	exported := rewards.ExportGenesis(suite.ctx, *suite.k)
	suite.Require().NoError(exported.Validate())

	// fresh chain importing the mid-lifecycle export
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()
	bank2 := testutil.NewMockBankKeeper(ctrl)
	epoch2 := testutil.NewMockEpochKeeper(ctrl)
	k2, ctx2 := keepertest.RewardsKeeper(suite.T(), bank2, epoch2, testutil.NewMockTradingKeeper(ctrl), testutil.NewMockAccountKeeper(ctrl))
	ledger2 := recordBankFlows(bank2, epoch2)
	rewards.InitGenesis(ctx2, k2, *exported)
	msgServer2 := keeper.NewMsgServerImpl(k2)

	// identical continuation on both chains
	continueLifecycle := func(k keeper.Keeper, ctx sdk.Context, ms types.MsgServer) string {
		for day := 0; day < 2; day++ {
			k.EnqueueStakingRewardsDistribution(ctx)
			k.ProcessStakingRewardsDistributionQueue(ctx)
		}
		claimResponse, err := ms.ClaimStakingRewards(ctx, &types.MsgClaimStakingRewards{
			Creator: late.String(), RewardId: rewardId,
		})
		suite.Require().NoError(err)
		_, err = ms.ExitStaking(ctx, &types.MsgExitStaking{Creator: dormant.String(), RewardId: rewardId})
		suite.Require().NoError(err)

		return claimResponse.Amount
	}

	controlOutMark := ledger.out
	controlClaim := continueLifecycle(*suite.k, suite.ctx, suite.msgServer)
	importedClaim := continueLifecycle(k2, ctx2, msgServer2)

	// identical outcomes: responses, flows and stores
	suite.Require().Equal(controlClaim, importedClaim)
	suite.Require().Equal(ledger.out.Sub(controlOutMark...), ledger2.out)
	suite.Require().Empty(ledger2.in)
	suite.Require().Equal(suite.k.GetAllBoosts(suite.ctx), k2.GetAllBoosts(ctx2))
	suite.Require().Equal(suite.k.GetAllBoostParticipant(suite.ctx), k2.GetAllBoostParticipant(ctx2))
	suite.Require().Equal(suite.k.GetAllStakingRewardParticipant(suite.ctx), k2.GetAllStakingRewardParticipant(ctx2))
	suite.Require().Equal(suite.k.GetBoostsCounter(suite.ctx), k2.GetBoostsCounter(ctx2))

	controlSr, found := suite.k.GetStakingReward(suite.ctx, rewardId)
	suite.Require().True(found)
	importedSr, found := k2.GetStakingReward(ctx2, rewardId)
	suite.Require().True(found)
	suite.Require().Equal(controlSr, importedSr)
}

// TestBoostSecurity_BaseBehaviourFreeze: with zero boosts, a full
// join/distribute/claim/exit cycle produces exactly the pre-boost
// observables — no boost event, no boost state, and the precise pre-boost
// coin flows.
func (suite *IntegrationTestSuite) TestBoostSecurity_BaseBehaviourFreeze() {
	ledger := recordBankFlows(suite.bank, suite.epoch)

	rewardId := "000000000001"
	suite.seedLifecycleReward(rewardId, 2)

	staker := sdk.AccAddress("staker.")
	suite.joinStaking(staker, rewardId, 100)
	suite.distributeOneDay()
	suite.distributeOneDay()

	// base accumulator: 2 days x 1000/100 = 20 per staked unit
	claimResponse, err := suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{
		Creator: staker.String(), RewardId: rewardId,
	})
	suite.Require().NoError(err)
	suite.Require().Equal("2000", claimResponse.Amount)

	suite.exitStaking(staker, rewardId)

	// pre-boost flows, to the unit: 100 ubze staked in; 2000 uprize rewards +
	// the 100 ubze stake out; nothing else
	suite.Require().Equal(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100))), ledger.in)
	suite.Require().Equal(
		sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(2000)), sdk.NewCoin("ubze", math.NewInt(100))),
		ledger.out,
	)

	// the finished reward was removed by the last exit, exactly as before
	_, found := suite.k.GetStakingReward(suite.ctx, rewardId)
	suite.Require().False(found)

	// zero boost traces anywhere: no state, no counter movement, no events
	suite.Require().Empty(suite.k.GetAllBoosts(suite.ctx))
	suite.Require().Empty(suite.k.GetAllBoostParticipant(suite.ctx))
	suite.Require().Equal(uint64(0), suite.k.GetBoostsCounter(suite.ctx))
	baseEvents := map[string]bool{}
	for _, event := range suite.ctx.EventManager().Events() {
		suite.Require().NotContains(event.Type, "Boost")
		baseEvents[event.Type] = true
	}
	// the pre-boost event set still fires
	for _, expected := range []string{
		"StakingRewardJoinEvent", "StakingRewardDistributionEvent",
		"StakingRewardClaimEvent", "StakingRewardExitEvent",
	} {
		foundEvent := false
		for eventType := range baseEvents {
			if strings.Contains(eventType, expected) {
				foundEvent = true
				break
			}
		}
		suite.Require().True(foundEvent, "missing base event %s", expected)
	}
}
