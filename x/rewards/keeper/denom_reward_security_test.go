package keeper_test

import (
	"context"
	"fmt"
	"strconv"
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

// This file is the cross-cutting Denom Rewards security suite (BZE-92): full-lifecycle escrow
// conservation (I4), the base-behaviour freeze of the audited SR flows, SR/DR behaviour parity,
// bounded-work proofs (I5), adversarial settle-ordering (I1), store-integrity audits (I2) and the
// genesis round-trip under load. The per-chunk regression tests live next to their handlers; the
// scenarios here span the whole feature through the public entry points.

// drSecLedger records every coin flow crossing the rewards module account, so tests assert escrow
// conservation over whole lifecycles instead of pinning individual transfers. Fee flows (captured
// via the trade keeper's capture-and-swap, forwarded module→fee-collector) are recorded separately
// from escrow so the two conservation properties can be asserted independently. The ledger also
// carries the mutable epoch clock returned by the epoch mock: day drives distributions, hour
// drives unlocks, and tests advance them explicitly.
type drSecLedger struct {
	in           sdk.Coins // user -> module (stakes, schedule budgets, airdrops)
	out          sdk.Coins // module -> user (prize payouts, stake returns)
	feeCaptured  sdk.Coins // user fees captured through CaptureAndSwapUserFee
	feeForwarded sdk.Coins // module -> CpFeeCollector forwards

	day  int64
	hour int64

	// lastS remembers every accumulator's last observed value for the I4 monotonicity audit
	lastS map[string]math.LegacyDec
}

// recordDenomRewardFlows replaces strict per-call expectations with recorders: any transfer
// succeeds and lands in the ledger, balances are unlimited, and the epoch keeper serves the
// ledger's mutable day/hour clock. The scenario, not the mock, decides what happens.
func recordDenomRewardFlows(bank *testutil.MockBankKeeper, epoch *testutil.MockEpochKeeper, trade *testutil.MockTradingKeeper) *drSecLedger {
	ledger := &drSecLedger{day: 100, hour: 2400, lastS: map[string]math.LegacyDec{}}

	rich := sdk.NewCoins()
	for _, denom := range []string{
		"ubze", "uprize", "ustake1", "ustake2", "udrstake",
		"uprizea", "uprizeb", "uprizec", "uprized", "uprizee",
	} {
		rich = rich.Add(sdk.NewCoin(denom, math.NewInt(1_000_000_000_000)))
	}

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
	bank.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, amt sdk.Coins) error {
			ledger.feeForwarded = ledger.feeForwarded.Add(amt...)
			return nil
		}).
		AnyTimes()
	trade.EXPECT().
		CaptureAndSwapUserFee(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, _ sdk.AccAddress, fee sdk.Coins, _ string) (sdk.Coins, error) {
			ledger.feeCaptured = ledger.feeCaptured.Add(fee...)
			return fee, nil
		}).
		AnyTimes()
	epoch.EXPECT().
		SafeGetEpochCountByIdentifier(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, identifier string) (int64, error) {
			if identifier == "day" {
				return ledger.day, nil
			}
			return ledger.hour, nil
		}).
		AnyTimes()

	return ledger
}

// escrow returns the ledger's escrow residue for one denom: everything sent in minus everything
// paid out. For staking denoms it must reach exactly zero at the end of a lifecycle; for prize
// denoms it must stay within the truncation-dust bound.
func (l *drSecLedger) residue(denom string) math.Int {
	return l.in.AmountOf(denom).Sub(l.out.AmountOf(denom))
}

// drAdvanceDay moves the day-epoch clock one tick, fires the real distribution hook and drains the
// EndBlock queue exactly as the chain would, then runs the I4 monotonicity audit: no accumulator
// may ever decrease, whatever the day did.
func (suite *IntegrationTestSuite) drAdvanceDay(led *drSecLedger) {
	led.day++
	led.hour += 24

	hook := suite.k.GetDenomRewardsDistributionHook()
	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "day", led.day))

	for i := 0; i < 25; i++ {
		if _, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx); !found {
			break
		}
		suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	}
	_, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found, "distribution queue not drained")

	// I4: S only ever grows
	for _, prize := range suite.k.GetAllDenomRewardPrize(suite.ctx) {
		key := prize.StakingDenom + "/" + prize.PrizeDenom
		s := prize.DistributedStake
		if prev, ok := led.lastS[key]; ok {
			suite.Require().True(s.GTE(prev), "accumulator %s decreased: %s -> %s", key, prev, s)
		}
		led.lastS[key] = s
	}
}

// drainAllPendingUnlocks discovers every pending-unlock epoch left in the shared pup/ store,
// enqueues each one and processes the generic unlock queue until the store is empty — the
// end-of-lifecycle equivalent of the hour epochs arriving.
func (suite *IntegrationTestSuite) drainAllPendingUnlocks() {
	epochs := map[int64]struct{}{}
	for _, pending := range suite.k.GetAllPendingUnlockParticipant(suite.ctx) {
		parts := strings.SplitN(pending.Index, "/", 2)
		epoch, err := strconv.ParseInt(parts[0], 10, 64)
		suite.Require().NoError(err)
		epochs[epoch] = struct{}{}
	}
	for epoch := range epochs {
		suite.k.EnqueueUnlockParticipants(suite.ctx, epoch)
	}
	for i := 0; i < 25; i++ {
		if len(suite.k.GetAllPendingUnlockParticipant(suite.ctx)) == 0 {
			return
		}
		suite.k.ProcessUnlockParticipantsQueue(suite.ctx)
	}
	suite.Require().Empty(suite.k.GetAllPendingUnlockParticipant(suite.ctx), "pending unlocks not drained")
}

// assertDrStoreIntegrity is the I2 audit: an index record may only exist while its accumulator
// AND its participant exist, and every participant carries its address-first marker. Run at any
// point of any lifecycle it must hold — exits remove indexes atomically with the position.
func (suite *IntegrationTestSuite) assertDrStoreIntegrity() {
	for _, index := range suite.k.GetAllDenomRewardParticipantIndex(suite.ctx) {
		_, found := suite.k.GetDenomRewardPrize(suite.ctx, index.StakingDenom, index.PrizeDenom)
		suite.Require().True(found, "index %s/%s/%s without its prize", index.Address, index.StakingDenom, index.PrizeDenom)
		_, found = suite.k.GetDenomRewardParticipant(suite.ctx, index.StakingDenom, index.Address)
		suite.Require().True(found, "index %s/%s/%s without its participant", index.Address, index.StakingDenom, index.PrizeDenom)
	}
	for _, participant := range suite.k.GetAllDenomRewardParticipant(suite.ctx) {
		suite.Require().True(
			hasDrParticipantMarker(*suite.k, suite.ctx, participant.Address, participant.StakingDenom),
			"participant %s/%s without its drp/a/ marker", participant.Address, participant.StakingDenom,
		)
	}
}

// --- message-driving helpers (all through the public msg server) ---

func (suite *IntegrationTestSuite) drSecJoin(addr sdk.AccAddress, denom string, amount int64) {
	_, err := suite.msgServer.JoinDenomReward(suite.ctx, &types.MsgJoinDenomReward{
		Creator: addr.String(), Denom: denom, Amount: math.NewInt(amount),
	})
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) drSecExit(addr sdk.AccAddress, denom string) {
	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{
		Creator: addr.String(), Denom: denom,
	})
	suite.Require().NoError(err)
}

// --- the escrow-conservation lifecycle (I4 end-to-end) ---

type drSecSchedule struct {
	prize      string
	daily      int64
	days       uint32
	extendDays uint32 // 0 = no extension; applied after phase 1
}

type drSecAirdrop struct {
	prize  string
	amount int64
	phase  int // 1 = after the first phase-1 day, 2 = after the first phase-2 day
}

type drSecDr struct {
	stakingDenom string
	lock         uint32  // snapshotted at creation; 0 = immediate stake return on exit
	stakes       []int64 // per-participant join amount (varying T)
	topUp        int64   // participant 0 tops up after phase 1 (0 = none)
	midClaim     bool    // last participant claims after phase 1
	schedules    []drSecSchedule
	airdrops     []drSecAirdrop
}

type drSecLifecycleCase struct {
	name      string
	maxPrizes uint32 // MaxPrizeDenomsPerDr for the case (per-DR prize-denom cap)
	phase1    int    // distributed days before the gap
	gap       int    // zero-staker days (I6): nothing may advance
	phase2    int    // distributed days after re-join; phase1+phase2 = longest schedule duration
	drs       []drSecDr
}

func (tc drSecLifecycleCase) participants(dr drSecDr) []sdk.AccAddress {
	list := make([]sdk.AccAddress, len(dr.stakes))
	for i := range dr.stakes {
		list[i] = sdk.AccAddress(fmt.Sprintf("drsec-%s-%02d....", dr.stakingDenom, i))
	}
	return list
}

// runDenomRewardLifecycle drives complete concurrent DR lifecycles through the public msg server
// and queue entry points and asserts the cross-cutting invariants: I4 (escrow conservation with
// residue bounded by truncation dust; stakes conserve exactly), I6 (zero-staker gaps advance
// nothing and strand nothing — every schedule still fully emits), I2 (store integrity at every
// phase boundary) and I1 implicitly (any settle-ordering flaw would surface as a conservation
// break). Fees run through the real capture paths and must conserve independently.
func (suite *IntegrationTestSuite) runDenomRewardLifecycle(tc drSecLifecycleCase) {
	led := recordDenomRewardFlows(suite.bank, suite.epoch, suite.trade)

	createFee := sdk.NewCoin("ubze", math.NewInt(25_000))
	prizeFee := sdk.NewCoin("ubze", math.NewInt(7_000))
	scheduleFee := sdk.NewCoin("ubze", math.NewInt(900))

	funder := sdk.AccAddress("drsec-funder....")
	settleOps := map[string]int{} // settle invocations per staking denom, for the dust bound

	// create every DR through the msg server; lock is a param snapshot, so params are set per DR
	for _, dr := range tc.drs {
		suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
			CreateDenomRewardFee:      createFee,
			CreateDenomRewardPrizeFee: prizeFee,
			AddDenomRewardScheduleFee: scheduleFee,
			MaxPrizeDenomsPerDr:       tc.maxPrizes,
			ExtraGasForDenomExit:      1_000,
			DenomRewardLock:           dr.lock,
			DenomRewardMinStake:       0,
		}))
		_, err := suite.msgServer.CreateDenomReward(suite.ctx, &types.MsgCreateDenomReward{
			Creator: funder.String(), Denom: dr.stakingDenom,
		})
		suite.Require().NoError(err)

		stored, found := suite.k.GetDenomReward(suite.ctx, dr.stakingDenom)
		suite.Require().True(found)
		suite.Require().Equal(dr.lock, stored.Lock, "lock snapshot")
	}

	// joins, schedules and phase-1 airdrops
	for _, dr := range tc.drs {
		for i, amount := range dr.stakes {
			suite.drSecJoin(tc.participants(dr)[i], dr.stakingDenom, amount)
		}
		for _, schedule := range dr.schedules {
			_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, &types.MsgCreateDenomRewardSchedule{
				Creator:     funder.String(),
				Denom:       dr.stakingDenom,
				PrizeDenom:  schedule.prize,
				DailyAmount: math.NewInt(schedule.daily),
				Duration:    fmt.Sprintf("%d", schedule.days),
			})
			suite.Require().NoError(err)
		}
	}

	for day := 0; day < tc.phase1; day++ {
		suite.drAdvanceDay(led)
		if day == 0 {
			for _, dr := range tc.drs {
				for _, airdrop := range dr.airdrops {
					if airdrop.phase != 1 {
						continue
					}
					_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, &types.MsgDistributeDenomRewards{
						Creator: funder.String(), Denom: dr.stakingDenom,
						PrizeDenom: airdrop.prize, Amount: math.NewInt(airdrop.amount),
					})
					suite.Require().NoError(err)
				}
			}
		}
	}
	suite.assertDrStoreIntegrity()

	// mid-lifecycle churn: top-ups, claims, schedule extensions
	for _, dr := range tc.drs {
		participants := tc.participants(dr)
		if dr.topUp > 0 {
			suite.drSecJoin(participants[0], dr.stakingDenom, dr.topUp)
			settleOps[dr.stakingDenom]++
		}
		if dr.midClaim {
			_, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{
				Creator: participants[len(participants)-1].String(), Denom: dr.stakingDenom,
			})
			suite.Require().NoError(err)
			settleOps[dr.stakingDenom]++
		}
		for _, schedule := range dr.schedules {
			if schedule.extendDays == 0 {
				continue
			}
			_, err := suite.msgServer.UpdateDenomRewardSchedule(suite.ctx, &types.MsgUpdateDenomRewardSchedule{
				Creator: funder.String(), Denom: dr.stakingDenom,
				ScheduleId: suite.drScheduleIdAt(dr.stakingDenom, schedule),
				Duration:   fmt.Sprintf("%d", schedule.extendDays),
			})
			suite.Require().NoError(err)
		}
	}

	// zero-staker gap (I6): everyone exits; skipped days consume nothing anywhere
	for _, dr := range tc.drs {
		for _, participant := range tc.participants(dr) {
			suite.drSecExit(participant, dr.stakingDenom)
			settleOps[dr.stakingDenom]++
		}
	}
	schedulesBeforeGap := suite.k.GetAllDenomRewardSchedule(suite.ctx)
	prizesBeforeGap := suite.k.GetAllDenomRewardPrize(suite.ctx)
	for day := 0; day < tc.gap; day++ {
		suite.drAdvanceDay(led)
	}
	suite.Require().Equal(schedulesBeforeGap, suite.k.GetAllDenomRewardSchedule(suite.ctx), "gap advanced a schedule")
	suite.Require().Equal(prizesBeforeGap, suite.k.GetAllDenomRewardPrize(suite.ctx), "gap advanced an accumulator")
	suite.assertDrStoreIntegrity()

	// stakers return; phase 2 finishes every schedule
	for _, dr := range tc.drs {
		for i, amount := range dr.stakes {
			suite.drSecJoin(tc.participants(dr)[i], dr.stakingDenom, amount)
		}
	}
	for day := 0; day < tc.phase2; day++ {
		suite.drAdvanceDay(led)
		if day == 0 {
			for _, dr := range tc.drs {
				for _, airdrop := range dr.airdrops {
					if airdrop.phase != 2 {
						continue
					}
					_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, &types.MsgDistributeDenomRewards{
						Creator: funder.String(), Denom: dr.stakingDenom,
						PrizeDenom: airdrop.prize, Amount: math.NewInt(airdrop.amount),
					})
					suite.Require().NoError(err)
				}
			}
		}
	}

	// every schedule fully emitted and deleted: the gap stranded nothing
	suite.Require().Empty(suite.k.GetAllDenomRewardSchedule(suite.ctx), "unfinished schedules left behind")

	// final claims and exits empty every position
	for _, dr := range tc.drs {
		for _, participant := range tc.participants(dr) {
			_, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{
				Creator: participant.String(), Denom: dr.stakingDenom,
			})
			if err != nil {
				// a pure-dust position claims nothing — allowed; the dust stays in escrow
				suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)
			}
			settleOps[dr.stakingDenom]++
			suite.drSecExit(participant, dr.stakingDenom)
			settleOps[dr.stakingDenom]++
		}
	}
	suite.drainAllPendingUnlocks()

	// all positions erased; the DR records survive with zero stake (DRs are never deleted)
	suite.Require().Empty(suite.k.GetAllDenomRewardParticipant(suite.ctx))
	suite.Require().Empty(suite.k.GetAllDenomRewardParticipantIndex(suite.ctx))
	for _, dr := range tc.drs {
		stored, found := suite.k.GetDenomReward(suite.ctx, dr.stakingDenom)
		suite.Require().True(found)
		suite.Require().Equal("0", stored.StakedAmount.String())
	}

	// I4 per (denom, prize): escrowed-in is exactly the budgets + airdrops, and everything
	// escrowed left again except truncation-level dust (each settle truncates < 1 unit per prize)
	for _, dr := range tc.drs {
		expectedEscrow := map[string]int64{}
		for _, schedule := range dr.schedules {
			expectedEscrow[schedule.prize] += schedule.daily * int64(schedule.days+schedule.extendDays)
		}
		for _, airdrop := range dr.airdrops {
			expectedEscrow[airdrop.prize] += airdrop.amount
		}
		dustBound := math.NewInt(int64(settleOps[dr.stakingDenom]))
		for prizeDenom, escrowed := range expectedEscrow {
			in := led.in.AmountOf(prizeDenom)
			suite.Require().True(in.Equal(math.NewInt(escrowed)), "prize %s: escrowed %s, expected %d", prizeDenom, in, escrowed)

			residue := led.residue(prizeDenom)
			suite.Require().True(residue.GTE(math.ZeroInt()), "prize %s: paid out more than escrowed (residue %s)", prizeDenom, residue)
			suite.Require().True(residue.LTE(dustBound), "prize %s: residue %s exceeds dust bound %s", prizeDenom, residue, dustBound)
		}

		// stakes conserve exactly: everything staked (two rounds of joins + top-up) came back
		suite.Require().True(led.residue(dr.stakingDenom).IsZero(),
			"stake %s: residue %s, expected exact conservation", dr.stakingDenom, led.residue(dr.stakingDenom))
	}

	// fees conserve independently: every captured fee was forwarded to the collector, and the
	// totals are exactly #DRs × create + #new-prize-denoms × prize + #schedules × schedule
	expectedFees := sdk.NewCoins()
	for _, dr := range tc.drs {
		expectedFees = expectedFees.Add(createFee)
		prizeDenoms := map[string]struct{}{}
		for _, schedule := range dr.schedules {
			expectedFees = expectedFees.Add(scheduleFee)
			prizeDenoms[schedule.prize] = struct{}{}
		}
		for _, airdrop := range dr.airdrops {
			prizeDenoms[airdrop.prize] = struct{}{}
		}
		for range prizeDenoms {
			expectedFees = expectedFees.Add(prizeFee)
		}
	}
	suite.Require().Equal(expectedFees, led.feeCaptured, "captured fees")
	suite.Require().Equal(expectedFees, led.feeForwarded, "forwarded fees")
}

// drScheduleIdAt finds the stored schedule matching a spec (by prize denom + daily amount) — specs
// don't carry ids because the counter is global across DRs.
func (suite *IntegrationTestSuite) drScheduleIdAt(stakingDenom string, spec drSecSchedule) string {
	var id string
	suite.k.IterateDenomSchedules(suite.ctx, stakingDenom, func(_ sdk.Context, schedule types.DenomRewardSchedule) bool {
		if schedule.PrizeDenom == spec.prize && schedule.DailyAmount.Equal(math.NewInt(spec.daily)) {
			id = schedule.ScheduleId
			return true
		}
		return false
	})
	suite.Require().NotEmpty(id, "schedule %s/%s not found", stakingDenom, spec.prize)
	return id
}

// TestDenomRewardSecurity_EscrowConservationLifecycle is the table-driven I4 suite: full
// lifecycles with varying T sizes, prize-denom counts up to the cap, day counts, same-denom
// schedule pairs, interleaved airdrops, top-ups, mid-run claims, extensions, zero-staker gaps,
// lock and no-lock exits, and two DRs running concurrently.
func (suite *IntegrationTestSuite) TestDenomRewardSecurity_EscrowConservationLifecycle() {
	cases := []drSecLifecycleCase{
		{
			name:      "single DR, one staker, exact division, no lock",
			maxPrizes: 50,
			phase1:    2, gap: 2, phase2: 2,
			drs: []drSecDr{{
				stakingDenom: "ustake1",
				lock:         0,
				stakes:       []int64{100},
				schedules:    []drSecSchedule{{prize: "uprizea", daily: 1000, days: 4}},
			}},
		},
		{
			name:      "same-denom schedule pair, airdrops interleaved, top-up, mid claim, extension, locked exits",
			maxPrizes: 50,
			phase1:    3, gap: 1, phase2: 3,
			drs: []drSecDr{{
				stakingDenom: "ustake1",
				lock:         7,
				stakes:       []int64{250, 1000, 5},
				topUp:        750,
				midClaim:     true,
				schedules: []drSecSchedule{
					{prize: "uprizea", daily: 1000, days: 3},
					{prize: "uprizea", daily: 500, days: 6},
					// still one day short of finishing when the extension lands after phase 1
					{prize: "uprizeb", daily: 777, days: 4, extendDays: 2},
				},
				airdrops: []drSecAirdrop{
					{prize: "uprizec", amount: 4001, phase: 1},
					{prize: "uprizea", amount: 999, phase: 2},
				},
			}},
		},
		{
			name:      "two DRs concurrently, dust-heavy staker, prize denoms at the cap",
			maxPrizes: 3,
			phase1:    4, gap: 3, phase2: 4,
			drs: []drSecDr{
				{
					stakingDenom: "ustake1",
					lock:         0,
					stakes:       []int64{1, 999999},
					midClaim:     true,
					schedules:    []drSecSchedule{{prize: "uprizea", daily: 10, days: 8}},
					airdrops: []drSecAirdrop{
						{prize: "uprizeb", amount: 123, phase: 1},
						{prize: "uprizec", amount: 55, phase: 2}, // 3rd prize denom == the cap
					},
				},
				{
					stakingDenom: "ustake2",
					lock:         2,
					stakes:       []int64{40, 60},
					topUp:        25,
					schedules: []drSecSchedule{
						{prize: "uprized", daily: 3333, days: 8},
						// unfinished at phase-1 end (4 payouts of 5), extended to the full 8
						{prize: "uprizee", daily: 100, days: 5, extendDays: 3},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.runDenomRewardLifecycle(tc)
		})
	}
}

// --- I1: adversarial settle ordering ---

// TestDenomRewardSecurity_I1_TopUpThenImmediateClaim_NoRetroactivePay attacks the
// settle-before-change ordering directly: a participant tops up ×10 right after a distribution
// and claims in the very next message. The top-up must settle at the OLD amount, so the claim
// pays nothing, and a later distribution pays everyone at the live amounts — the attacker never
// captures a single unit of pre-top-up accrual at the post-top-up amount.
func (suite *IntegrationTestSuite) TestDenomRewardSecurity_I1_TopUpThenImmediateClaim_NoRetroactivePay() {
	led := recordDenomRewardFlows(suite.bank, suite.epoch, suite.trade)
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{MaxPrizeDenomsPerDr: 50}))

	attacker := sdk.AccAddress("drsec-attacker..")
	victim := sdk.AccAddress("drsec-victim....")
	funder := sdk.AccAddress("drsec-funder....")

	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ustake1", Lock: 0, MinStake: 0, StakedAmount: math.NewInt(0)})
	suite.drSecJoin(attacker, "ustake1", 100)
	suite.drSecJoin(victim, "ustake1", 300)

	// day of accrual: 400 escrowed on T = 400 → S = 1
	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, &types.MsgDistributeDenomRewards{
		Creator: funder.String(), Denom: "ustake1", PrizeDenom: "uprizea", Amount: math.NewInt(400),
	})
	suite.Require().NoError(err)

	// the attack: top up 100 → 1000, then claim immediately
	suite.drSecJoin(attacker, "ustake1", 900) // settles 100 × 1 = 100 first (I1)
	_, err = suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{
		Creator: attacker.String(), Denom: "ustake1",
	})
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim, "top-up left claimable accrual behind")
	suite.Require().True(led.out.AmountOf("uprizea").Equal(math.NewInt(100)), "attacker settled at the wrong amount")

	// next distribution pays at the live amounts: T = 1300, S += 1
	_, err = suite.msgServer.DistributeDenomRewards(suite.ctx, &types.MsgDistributeDenomRewards{
		Creator: funder.String(), Denom: "ustake1", PrizeDenom: "uprizea", Amount: math.NewInt(1300),
	})
	suite.Require().NoError(err)

	claimed, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{
		Creator: attacker.String(), Denom: "ustake1",
	})
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin("uprizea", 1000)), sdk.Coins(claimed.Amounts))

	claimed, err = suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{
		Creator: victim.String(), Denom: "ustake1",
	})
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin("uprizea", 600)), sdk.Coins(claimed.Amounts))

	// exact conservation: 1700 in, 100 + 1000 + 600 out, zero residue
	suite.Require().True(led.residue("uprizea").IsZero())
}

// --- I2: store integrity under a mixed sequence ---

// TestDenomRewardSecurity_I2_NoOrphanIndexes runs a mixed join/airdrop/schedule/exit sequence and
// audits after every step: no index may outlive its prize or its participant, and every
// participant carries its marker. The exit's atomic removal of indexes+position+marker is what
// the audit catches if it ever regresses.
func (suite *IntegrationTestSuite) TestDenomRewardSecurity_I2_NoOrphanIndexes() {
	recordDenomRewardFlows(suite.bank, suite.epoch, suite.trade)
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{MaxPrizeDenomsPerDr: 50}))

	early := sdk.AccAddress("drsec-early.....")
	late := sdk.AccAddress("drsec-late......")
	funder := sdk.AccAddress("drsec-funder....")

	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ustake1", Lock: 0, MinStake: 0, StakedAmount: math.NewInt(0)})

	suite.drSecJoin(early, "ustake1", 100) // joins before any prize exists: zero stamps
	suite.assertDrStoreIntegrity()

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, &types.MsgDistributeDenomRewards{
		Creator: funder.String(), Denom: "ustake1", PrizeDenom: "uprizea", Amount: math.NewInt(500),
	})
	suite.Require().NoError(err)
	_, err = suite.msgServer.CreateDenomRewardSchedule(suite.ctx, &types.MsgCreateDenomRewardSchedule{
		Creator: funder.String(), Denom: "ustake1", PrizeDenom: "uprizeb", DailyAmount: math.NewInt(100), Duration: "2",
	})
	suite.Require().NoError(err)

	suite.drSecJoin(late, "ustake1", 400) // stamped on both accumulators
	suite.assertDrStoreIntegrity()
	suite.Require().Len(suite.k.GetAllDenomRewardParticipantIndex(suite.ctx), 2, "late joiner must be stamped on every prize")

	// early's exit settles (paying its uprizea accrual) and removes its position atomically
	suite.drSecExit(early, "ustake1")
	suite.assertDrStoreIntegrity()
	for _, index := range suite.k.GetAllDenomRewardParticipantIndex(suite.ctx) {
		suite.Require().NotEqual(early.String(), index.Address, "exit left an index behind")
	}
	suite.Require().False(hasDrParticipantMarker(*suite.k, suite.ctx, early.String(), "ustake1"))

	suite.drSecExit(late, "ustake1")
	suite.assertDrStoreIntegrity()
	suite.Require().Empty(suite.k.GetAllDenomRewardParticipantIndex(suite.ctx))
	suite.Require().Empty(suite.k.GetAllDenomRewardParticipant(suite.ctx))
}

// --- I5: bounded work ---

// TestDenomRewardSecurity_I5_BoundedWork_GasIndependentOfParticipants proves nothing iterates
// participants: a distribution day, a claim and an exit consume EXACTLY the same gas whether the
// DR has 2 or 300 participants. Any hidden participant iteration would scale the gas with the
// participant count and fail the equality.
func (suite *IntegrationTestSuite) TestDenomRewardSecurity_I5_BoundedWork_GasIndependentOfParticipants() {
	measure := func(nParticipants int) (distGas, claimGas, exitGas uint64) {
		suite.SetupTest()
		led := recordDenomRewardFlows(suite.bank, suite.epoch, suite.trade)
		suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
			MaxPrizeDenomsPerDr:  50,
			ExtraGasForDenomExit: 1_000,
		}))

		// same T and same measured participant regardless of the crowd size, so every measured
		// operation moves identical amounts and writes identical bytes
		measured := sdk.AccAddress("drsec-measured..")
		suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ustake1", Lock: 0, MinStake: 0, StakedAmount: math.NewInt(100000)})
		suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ustake1", PrizeDenom: "uprizea", DistributedStake: math.LegacyMustNewDecFromStr("0")})
		suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: measured.String(), StakingDenom: "ustake1", Amount: math.NewInt(50000)})
		for i := 1; i < nParticipants; i++ {
			suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{
				Address:      sdk.AccAddress(fmt.Sprintf("drsec-crowd-%04d", i)).String(),
				StakingDenom: "ustake1",
				Amount:       math.NewInt(7),
			})
		}
		suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
			ScheduleId: "000000000001", StakingDenom: "ustake1", PrizeDenom: "uprizea",
			DailyAmount: math.NewInt(100000), Duration: 5, Payouts: 0,
		})

		gas := func(op func()) uint64 {
			before := suite.ctx.GasMeter().GasConsumed()
			op()
			return suite.ctx.GasMeter().GasConsumed() - before
		}

		distGas = gas(func() { suite.drAdvanceDay(led) })
		claimGas = gas(func() {
			_, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{
				Creator: measured.String(), Denom: "ustake1",
			})
			suite.Require().NoError(err)
		})
		exitGas = gas(func() { suite.drSecExit(measured, "ustake1") })

		return distGas, claimGas, exitGas
	}

	smallDist, smallClaim, smallExit := measure(2)
	bigDist, bigClaim, bigExit := measure(300)

	suite.Require().Equal(smallDist, bigDist, "distribution gas scales with participants")
	suite.Require().Equal(smallClaim, bigClaim, "claim gas scales with participants")
	suite.Require().Equal(smallExit, bigExit, "exit gas scales with participants")
}

// --- SR-parity (mandatory group 1) ---

// TestDenomRewardSecurity_SrParity asserts the DR mechanisms behave EQUAL to their audited SR
// counterparts, side by side in the same store: dust semantics, zero-staker day skips, escrow
// amounts, fee capture flows and the unlock epoch computation.
func (suite *IntegrationTestSuite) TestDenomRewardSecurity_SrParity() {
	suite.Run("dust: no pay, no stamp, cumulative accrual pays later", func() {
		suite.SetupTest()
		led := recordDenomRewardFlows(suite.bank, suite.epoch, suite.trade)
		user := sdk.AccAddress("drsec-parity-01.")

		// identical numbers on both systems: amount 100, S moves 0 → 0.005 → 0.01
		suite.k.SetStakingReward(suite.ctx, types.StakingReward{
			RewardId: "000000000001", PrizeDenom: "uprize", StakingDenom: "ubze",
			PrizeAmount: "1", Duration: 10, StakedAmount: "100", DistributedStake: "0.005",
		})
		suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
			Address: user.String(), RewardId: "000000000001", Amount: "100", JoinedAt: "0",
		})
		suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ustake1", Lock: 0, MinStake: 0, StakedAmount: math.NewInt(100)})
		suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ustake1", PrizeDenom: "uprizea", DistributedStake: math.LegacyMustNewDecFromStr("0.005")})
		suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: user.String(), StakingDenom: "ustake1", Amount: math.NewInt(100)})

		// both claims: positive pending truncating to zero → same error, no snapshot advance
		_, srErr := suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{Creator: user.String(), RewardId: "000000000001"})
		_, drErr := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: user.String(), Denom: "ustake1"})
		suite.Require().ErrorIs(srErr, types.ErrNoRewardsToClaim)
		suite.Require().ErrorIs(drErr, types.ErrNoRewardsToClaim)

		srParticipant, _ := suite.k.GetStakingRewardParticipant(suite.ctx, user.String(), "000000000001")
		suite.Require().Equal("0", srParticipant.JoinedAt, "SR dust claim advanced JoinedAt")
		_, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, user.String(), "ustake1", "uprizea")
		suite.Require().False(found, "DR dust claim stamped an index")

		// accrual doubles: both pay exactly the same single whole unit
		sr, _ := suite.k.GetStakingReward(suite.ctx, "000000000001")
		sr.DistributedStake = "0.01"
		suite.k.SetStakingReward(suite.ctx, sr)
		prize, _ := suite.k.GetDenomRewardPrize(suite.ctx, "ustake1", "uprizea")
		prize.DistributedStake = math.LegacyMustNewDecFromStr("0.01")
		suite.k.SetDenomRewardPrize(suite.ctx, prize)

		srClaim, err := suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{Creator: user.String(), RewardId: "000000000001"})
		suite.Require().NoError(err)
		drClaim, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: user.String(), Denom: "ustake1"})
		suite.Require().NoError(err)
		suite.Require().Equal(srClaim.Amount, drClaim.Amounts[0].Amount.String(), "dust payout parity")
		suite.Require().Equal(led.out.AmountOf("uprize"), led.out.AmountOf("uprizea"), "coin flow parity")
	})

	suite.Run("zero-staker day: both skip without consuming a payout", func() {
		suite.SetupTest()
		led := recordDenomRewardFlows(suite.bank, suite.epoch, suite.trade)

		suite.k.SetStakingReward(suite.ctx, types.StakingReward{
			RewardId: "000000000001", PrizeDenom: "uprize", StakingDenom: "ubze",
			PrizeAmount: "1000", Duration: 5, Payouts: 0, StakedAmount: "0", DistributedStake: "0",
		})
		suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ustake1", StakedAmount: math.NewInt(0)})
		suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ustake1", PrizeDenom: "uprizea", DistributedStake: math.LegacyMustNewDecFromStr("0")})
		suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
			ScheduleId: "000000000001", StakingDenom: "ustake1", PrizeDenom: "uprizea",
			DailyAmount: math.NewInt(1000), Duration: 5, Payouts: 0,
		})

		suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
		suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)
		suite.drAdvanceDay(led)

		sr, _ := suite.k.GetStakingReward(suite.ctx, "000000000001")
		schedule, _ := suite.k.GetDenomRewardSchedule(suite.ctx, "ustake1", "000000000001")
		suite.Require().Equal(uint32(0), sr.Payouts, "SR consumed a zero-staker day")
		suite.Require().Equal(uint32(0), schedule.Payouts, "DR consumed a zero-staker day")
		suite.Require().Equal("0", sr.DistributedStake)
		prize, _ := suite.k.GetDenomRewardPrize(suite.ctx, "ustake1", "uprizea")
		suite.Require().True(math.LegacyMustNewDecFromStr("0").Equal(prize.DistributedStake))
	})

	suite.Run("escrow and fee capture: identical amounts through identical paths", func() {
		suite.SetupTest()
		led := recordDenomRewardFlows(suite.bank, suite.epoch, suite.trade)
		creator := sdk.AccAddress("drsec-parity-02.")

		fee := sdk.NewCoin("ubze", math.NewInt(25_000))
		suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
			CreateStakingRewardFee: fee,
			CreateDenomRewardFee:   fee,
			MaxPrizeDenomsPerDr:    50,
		}))

		// SR: create escrows prize_amount × duration and captures the creation fee
		_, err := suite.msgServer.CreateStakingReward(suite.ctx, &types.MsgCreateStakingReward{
			Creator: creator.String(), PrizeAmount: "100", PrizeDenom: "uprize",
			StakingDenom: "ubze", Duration: "5", MinStake: "0", Lock: "0",
		})
		suite.Require().NoError(err)
		srEscrow := led.in.AmountOf("uprize")
		srFee := led.feeCaptured.AmountOf("ubze")

		// DR: create captures the same fee; a schedule escrows daily × duration up front
		_, err = suite.msgServer.CreateDenomReward(suite.ctx, &types.MsgCreateDenomReward{Creator: creator.String(), Denom: "ustake1"})
		suite.Require().NoError(err)
		_, err = suite.msgServer.CreateDenomRewardSchedule(suite.ctx, &types.MsgCreateDenomRewardSchedule{
			Creator: creator.String(), Denom: "ustake1", PrizeDenom: "uprizea", DailyAmount: math.NewInt(100), Duration: "5",
		})
		suite.Require().NoError(err)

		suite.Require().True(srEscrow.Equal(led.in.AmountOf("uprizea")), "escrow parity: SR %s vs DR %s", srEscrow, led.in.AmountOf("uprizea"))
		suite.Require().True(srFee.Equal(led.feeCaptured.AmountOf("ubze").Sub(srFee)), "creation fee parity")
		suite.Require().Equal(led.feeCaptured, led.feeForwarded, "every captured fee reaches the collector")
	})

	suite.Run("unlock: same lock produces the same epoch in the shared store, one drain pays both", func() {
		suite.SetupTest()
		led := recordDenomRewardFlows(suite.bank, suite.epoch, suite.trade)
		user := sdk.AccAddress("drsec-parity-03.")

		suite.k.SetStakingReward(suite.ctx, types.StakingReward{
			RewardId: "000000000001", PrizeDenom: "uprize", StakingDenom: "ubze",
			PrizeAmount: "1", Duration: 10, StakedAmount: "100", DistributedStake: "0", Lock: 3,
		})
		suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{
			Address: user.String(), RewardId: "000000000001", Amount: "100", JoinedAt: "0",
		})
		suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ustake1", Lock: 3, MinStake: 0, StakedAmount: math.NewInt(70)})
		suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: user.String(), StakingDenom: "ustake1", Amount: math.NewInt(70)})

		_, err := suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{Creator: user.String(), RewardId: "000000000001"})
		suite.Require().NoError(err)
		suite.drSecExit(user, "ustake1")

		// both entries sit at the SAME epoch (hour + 3×24), keyed collision-free
		unlockEpoch := led.hour + 3*24
		entries := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, unlockEpoch)
		suite.Require().Len(entries, 2, "SR and DR unlocks must share the epoch prefix")

		// one generic drain pays both stakes in full
		suite.drainAllPendingUnlocks()
		suite.Require().True(led.out.AmountOf("ubze").Equal(math.NewInt(100)))
		suite.Require().True(led.out.AmountOf("ustake1").Equal(math.NewInt(70)))
	})
}

// --- base-behaviour freeze ---

// srObservables captures every observable output of a fixed SR lifecycle: the claim response, the
// exact coin flows and the SR event types in order. Two runs are behaviour-identical iff their
// observables are equal.
type srObservables struct {
	claimAmount  string
	ubzeIn       math.Int
	ubzeOut      math.Int
	uprizeOut    math.Int
	srEventTypes []string
}

// runSrLifecycleObservables drives the fixed SR lifecycle — join 100, two 1000-payout
// distribution days, claim, locked exit, unlock drain — and returns its observables. When withDr
// is true, a full DR world (separate staking/prize denoms) runs interleaved with it: a DR
// schedule distributing on the same day ticks, a DR participant, and a DR exit landing in the
// shared unlock store at the SAME epoch as the SR exit.
func (suite *IntegrationTestSuite) runSrLifecycleObservables(withDr bool) srObservables {
	suite.SetupTest()
	led := recordDenomRewardFlows(suite.bank, suite.epoch, suite.trade)

	staker := sdk.AccAddress("drsec-freeze-sr.")
	drStaker := sdk.AccAddress("drsec-freeze-dr.")
	funder := sdk.AccAddress("drsec-funder....")

	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId: "000000000001", PrizeAmount: "1000", PrizeDenom: "uprize",
		StakingDenom: "ubze", Duration: 2, Payouts: 0, StakedAmount: "0", DistributedStake: "0", Lock: 2,
	})

	if withDr {
		suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
			CreateDenomRewardPrizeFee: sdk.NewCoin("ubze", math.NewInt(7_000)),
			MaxPrizeDenomsPerDr:       50,
			DenomRewardLock:           2, // same lock as the SR: both unlocks land on one epoch
		}))
		_, err := suite.msgServer.CreateDenomReward(suite.ctx, &types.MsgCreateDenomReward{Creator: funder.String(), Denom: "udrstake"})
		suite.Require().NoError(err)
		suite.drSecJoin(drStaker, "udrstake", 200)
		_, err = suite.msgServer.CreateDenomRewardSchedule(suite.ctx, &types.MsgCreateDenomRewardSchedule{
			Creator: funder.String(), Denom: "udrstake", PrizeDenom: "uprized", DailyAmount: math.NewInt(500), Duration: "2",
		})
		suite.Require().NoError(err)
	}

	_, err := suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{
		Creator: staker.String(), RewardId: "000000000001", Amount: "100",
	})
	suite.Require().NoError(err)

	for day := 0; day < 2; day++ {
		suite.k.EnqueueStakingRewardsDistribution(suite.ctx)
		suite.k.ProcessStakingRewardsDistributionQueue(suite.ctx)
		suite.drAdvanceDay(led) // fires the DR day hook; a no-op without DR schedules
	}

	claim, err := suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{
		Creator: staker.String(), RewardId: "000000000001",
	})
	suite.Require().NoError(err)

	_, err = suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{Creator: staker.String(), RewardId: "000000000001"})
	suite.Require().NoError(err)
	if withDr {
		suite.drSecExit(drStaker, "udrstake") // same hour epoch → shared unlock store and queue
	}
	suite.drainAllPendingUnlocks()

	var srEventTypes []string
	for _, event := range suite.ctx.EventManager().Events() {
		if strings.Contains(event.Type, "StakingReward") {
			srEventTypes = append(srEventTypes, event.Type)
		}
	}

	return srObservables{
		claimAmount:  claim.Amount,
		ubzeIn:       led.in.AmountOf("ubze"),
		ubzeOut:      led.out.AmountOf("ubze"),
		uprizeOut:    led.out.AmountOf("uprize"),
		srEventTypes: srEventTypes,
	}
}

// TestDenomRewardSecurity_BaseBehaviourFreeze_ZeroDrState: with zero DR state, a full SR
// join/distribute/claim/exit/unlock cycle produces exactly the pre-DR observables — the precise
// pre-DR coin flows, no DR event, no DR state and no DR queue. The DR day hook and EndBlock
// processor run on every tick and must leave no trace.
func (suite *IntegrationTestSuite) TestDenomRewardSecurity_BaseBehaviourFreeze_ZeroDrState() {
	observed := suite.runSrLifecycleObservables(false)

	// pre-DR flows, to the unit: 100 ubze staked in; 2 days × 1000 uprize claimed; stake unlocked
	suite.Require().Equal("2000", observed.claimAmount)
	suite.Require().True(observed.ubzeIn.Equal(math.NewInt(100)))
	suite.Require().True(observed.ubzeOut.Equal(math.NewInt(100)))
	suite.Require().True(observed.uprizeOut.Equal(math.NewInt(2000)))

	// zero DR traces anywhere: no state, no counter movement, no queue, no events
	suite.Require().Empty(suite.k.GetAllDenomReward(suite.ctx))
	suite.Require().Empty(suite.k.GetAllDenomRewardPrize(suite.ctx))
	suite.Require().Empty(suite.k.GetAllDenomRewardParticipant(suite.ctx))
	suite.Require().Empty(suite.k.GetAllDenomRewardParticipantIndex(suite.ctx))
	suite.Require().Empty(suite.k.GetAllDenomRewardSchedule(suite.ctx))
	suite.Require().Equal(uint64(0), suite.k.GetDenomRewardScheduleCounter(suite.ctx))
	_, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found)
	for _, event := range suite.ctx.EventManager().Events() {
		suite.Require().NotContains(event.Type, "DenomReward", "DR event emitted with zero DR state")
	}
}

// TestDenomRewardSecurity_BaseBehaviourFreeze_SrUnaffectedByActiveDrs: the same SR lifecycle runs
// with a full DR world active — schedules distributing on the same day ticks, participants, and a
// DR exit unlocking at the same epoch through the shared store and queue. Every SR observable
// (claim response, coin flows, event sequence) must be identical to the zero-DR run, and the DR
// flows must never touch the SR denoms.
func (suite *IntegrationTestSuite) TestDenomRewardSecurity_BaseBehaviourFreeze_SrUnaffectedByActiveDrs() {
	control := suite.runSrLifecycleObservables(false)
	withDr := suite.runSrLifecycleObservables(true)

	suite.Require().Equal(control, withDr, "active DRs changed an SR observable")

	// the DR world really ran: its own denoms moved (200 stake in+out, 2×500 schedule payouts)
	dr, found := suite.k.GetDenomReward(suite.ctx, "udrstake")
	suite.Require().True(found)
	suite.Require().Equal("0", dr.StakedAmount.String())
}

// --- genesis round-trip under load ---

// TestDenomRewardSecurity_GenesisRoundTripUnderLoad builds a mid-lifecycle DR state through the
// public entry points — an active and a finished schedule, an airdrop-created prize, a dormant
// participant with zero stamps (joined before any prize existed) and a stamped late joiner —
// exports it, imports into a fresh chain and runs an identical continuation on both: every
// payout, response and store must match the unexported control run.
func (suite *IntegrationTestSuite) TestDenomRewardSecurity_GenesisRoundTripUnderLoad() {
	led := recordDenomRewardFlows(suite.bank, suite.epoch, suite.trade)
	// the export carries params, and genesis validation requires a fully valid set — so the
	// genesis test starts from the defaults and only adjusts the DR knobs it needs
	params := types.DefaultParams()
	params.CreateDenomRewardFee = sdk.NewCoin("ubze", math.NewInt(25_000))
	params.CreateDenomRewardPrizeFee = sdk.NewCoin("ubze", math.NewInt(7_000))
	params.DenomRewardLock = 0 // lock 0: continuation exits pay immediately on both chains
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	dormant := sdk.AccAddress("drsec-dormant...")
	late := sdk.AccAddress("drsec-late......")
	funder := sdk.AccAddress("drsec-funder....")

	_, err := suite.msgServer.CreateDenomReward(suite.ctx, &types.MsgCreateDenomReward{Creator: funder.String(), Denom: "ustake1"})
	suite.Require().NoError(err)

	// dormant joins BEFORE any prize exists: zero index records, pure lazy-zero exposure
	suite.drSecJoin(dormant, "ustake1", 100)

	for _, schedule := range []types.MsgCreateDenomRewardSchedule{
		{Creator: funder.String(), Denom: "ustake1", PrizeDenom: "uprizea", DailyAmount: math.NewInt(1000), Duration: "6"},
		{Creator: funder.String(), Denom: "ustake1", PrizeDenom: "uprizeb", DailyAmount: math.NewInt(300), Duration: "2"},
	} {
		msg := schedule
		_, err = suite.msgServer.CreateDenomRewardSchedule(suite.ctx, &msg)
		suite.Require().NoError(err)
	}

	// two days: the uprizeb schedule finishes, the uprizea one stays active
	suite.drAdvanceDay(led)
	suite.drAdvanceDay(led)

	// the late joiner is stamped at the current accumulators (incl. the finished schedule's final
	// S); an airdrop then creates a THIRD prize born after the late join
	suite.drSecJoin(late, "ustake1", 400)
	suite.drAdvanceDay(led)
	_, err = suite.msgServer.DistributeDenomRewards(suite.ctx, &types.MsgDistributeDenomRewards{
		Creator: funder.String(), Denom: "ustake1", PrizeDenom: "uprizec", Amount: math.NewInt(555),
	})
	suite.Require().NoError(err)

	exported := rewards.ExportGenesis(suite.ctx, *suite.k)
	suite.Require().NoError(exported.Validate())

	// fresh chain importing the mid-lifecycle export, with its epoch clock aligned
	mockCtrl := gomock.NewController(suite.T())
	defer mockCtrl.Finish()
	bank2 := testutil.NewMockBankKeeper(mockCtrl)
	epoch2 := testutil.NewMockEpochKeeper(mockCtrl)
	trade2 := testutil.NewMockTradingKeeper(mockCtrl)
	k2, ctx2 := keepertest.RewardsKeeper(suite.T(), bank2, epoch2, trade2, testutil.NewMockAccountKeeper(mockCtrl))
	led2 := recordDenomRewardFlows(bank2, epoch2, trade2)
	led2.day, led2.hour = led.day, led.hour
	rewards.InitGenesis(ctx2, k2, *exported)
	msgServer2 := keeper.NewMsgServerImpl(&k2)

	// identical continuation on both chains: three more days finish the active schedule, the late
	// joiner claims everything, the dormant participant exits
	continueLifecycle := func(k keeper.Keeper, ctx sdk.Context, ms types.MsgServer, l *drSecLedger) sdk.Coins {
		for day := 0; day < 3; day++ {
			l.day++
			l.hour += 24
			suite.Require().NoError(k.GetDenomRewardsDistributionHook().AfterEpochEnd(ctx, "day", l.day))
			for i := 0; i < 5; i++ {
				k.ProcessDenomRewardsDistributionQueue(ctx)
			}
		}

		claim, err := ms.ClaimDenomRewards(ctx, &types.MsgClaimDenomRewards{Creator: late.String(), Denom: "ustake1"})
		suite.Require().NoError(err)
		_, err = ms.ExitDenomReward(ctx, &types.MsgExitDenomReward{Creator: dormant.String(), Denom: "ustake1"})
		suite.Require().NoError(err)

		return sdk.Coins(claim.Amounts)
	}

	controlOutMark := led.out
	controlClaim := continueLifecycle(*suite.k, suite.ctx, suite.msgServer, led)
	importedClaim := continueLifecycle(k2, ctx2, msgServer2, led2)

	// identical outcomes: responses, flows and every store
	suite.Require().Equal(controlClaim, importedClaim)
	suite.Require().Equal(led.out.Sub(controlOutMark...), led2.out, "imported chain paid different amounts")
	suite.Require().Empty(led2.in, "imported chain escrowed coins the control did not")
	suite.Require().Equal(suite.k.GetAllDenomReward(suite.ctx), k2.GetAllDenomReward(ctx2))
	suite.Require().Equal(suite.k.GetAllDenomRewardPrize(suite.ctx), k2.GetAllDenomRewardPrize(ctx2))
	suite.Require().Equal(suite.k.GetAllDenomRewardParticipant(suite.ctx), k2.GetAllDenomRewardParticipant(ctx2))
	suite.Require().Equal(suite.k.GetAllDenomRewardParticipantIndex(suite.ctx), k2.GetAllDenomRewardParticipantIndex(ctx2))
	suite.Require().Equal(suite.k.GetAllDenomRewardSchedule(suite.ctx), k2.GetAllDenomRewardSchedule(ctx2))
	suite.Require().Equal(suite.k.GetDenomRewardScheduleCounter(suite.ctx), k2.GetDenomRewardScheduleCounter(ctx2))

	// both chains re-export identically after the identical continuation
	suite.Require().Equal(rewards.ExportGenesis(suite.ctx, *suite.k), rewards.ExportGenesis(ctx2, k2))
}
