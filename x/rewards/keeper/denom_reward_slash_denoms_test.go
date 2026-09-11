package keeper_test

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"
)

// drSlashFlows is a minimal flow recorder for the slash-denom lifecycle: every transfer succeeds
// and is tallied, balances are unlimited in the denoms the scenario uses, and the epoch keeper
// serves a mutable day/hour clock. (recordDenomRewardFlows serves a fixed denom list, so the
// factory/ibc denoms of this test need their own recorder.)
type drSlashFlows struct {
	in, out   sdk.Coins
	day, hour int64
}

func (suite *IntegrationTestSuite) recordSlashDenomFlows(denoms ...string) *drSlashFlows {
	flows := &drSlashFlows{day: 50, hour: 1200}

	rich := sdk.NewCoins()
	for _, denom := range denoms {
		rich = rich.Add(sdk.NewCoin(denom, math.NewInt(1_000_000_000_000)))
	}

	suite.bank.EXPECT().HasSupply(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
	suite.bank.EXPECT().SpendableCoins(gomock.Any(), gomock.Any()).Return(rich).AnyTimes()
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ sdk.AccAddress, _ string, amt sdk.Coins) error {
			flows.in = flows.in.Add(amt...)
			return nil
		}).
		AnyTimes()
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ sdk.AccAddress, amt sdk.Coins) error {
			flows.out = flows.out.Add(amt...)
			return nil
		}).
		AnyTimes()
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, _ sdk.AccAddress, fee sdk.Coins, _ string) (sdk.Coins, error) {
			return fee, nil
		}).
		AnyTimes()
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, identifier string) (int64, error) {
			if identifier == "day" {
				return flows.day, nil
			}
			return flows.hour, nil
		}).
		AnyTimes()

	return flows
}

// slashAdvanceDay ticks the day (and 24 hours), fires the real day-epoch hook and drains the
// EndBlock distribution queue.
func (suite *IntegrationTestSuite) slashAdvanceDay(flows *drSlashFlows) {
	flows.day++
	flows.hour += 24

	hook := suite.k.GetDenomRewardsDistributionHook()
	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "day", flows.day))
	for i := 0; i < 10; i++ {
		if _, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx); !found {
			break
		}
		suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	}
	_, found := suite.k.GetDenomRewardsDistributionQueue(suite.ctx)
	suite.Require().False(found, "distribution queue not drained")
}

func (suite *IntegrationTestSuite) slashPending(addr sdk.AccAddress, stakingDenom string) sdk.Coins {
	res, err := suite.k.DenomRewardParticipant(suite.ctx, &types.QueryDenomRewardParticipantRequest{
		Address: addr.String(), Denom: stakingDenom,
	})
	suite.Require().NoError(err)

	return sdk.NewCoins(res.Pending...)
}

// TestDenomRewardSlashDenoms_FullLifecycle drives a complete denom reward lifecycle with the
// denoms the key layout was designed around: a factory staking denom and an ibc prize denom, both
// containing "/". Every store family with a dynamic denom segment is exercised through the public
// entry points — DR record, prize accumulator, denom-first participant, address-first marker,
// participant index, schedule, the distribution cursor, and the shared pending-unlock key
// "{epoch}/dr/{denom}/{addr}" whose denom segment carries extra slashes — and escrow must conserve
// exactly (the amounts are chosen so no truncation dust is produced).
func (suite *IntegrationTestSuite) TestDenomRewardSlashDenoms_FullLifecycle() {
	stakingDenom := "factory/" + sample.AccAddress() + "/tok"
	prizeDenom := "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	suite.Require().NoError(sdk.ValidateDenom(stakingDenom))
	suite.Require().NoError(sdk.ValidateDenom(prizeDenom))

	flows := suite.recordSlashDenomFlows("ubze", stakingDenom, prizeDenom)

	creator := sdk.AccAddress("slash-creator...")
	funder := sdk.AccAddress("slash-funder....")
	addrA := sdk.AccAddress("slash-staker-a..")
	addrB := sdk.AccAddress("slash-staker-b..")

	const lockDays = 2
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
		CreateDenomRewardFee:      sdk.NewCoin("ubze", math.NewInt(25_000)),
		CreateDenomRewardPrizeFee: sdk.NewCoin("ubze", math.NewInt(7_000)),
		AddDenomRewardScheduleFee: sdk.NewCoin("ubze", math.NewInt(900)),
		MaxPrizeDenomsPerDr:       50,
		ExtraGasForDenomExit:      1_000,
		DenomRewardLock:           lockDays,
		DenomRewardMinStake:       0,
	}))

	// --- create + join ---
	_, err := suite.msgServer.CreateDenomReward(suite.ctx, types.NewMsgCreateDenomReward(creator.String(), stakingDenom))
	suite.Require().NoError(err)
	drRes, err := suite.k.DenomReward(suite.ctx, &types.QueryDenomRewardRequest{Denom: stakingDenom})
	suite.Require().NoError(err)
	suite.Require().Equal(stakingDenom, drRes.DenomReward.StakingDenom)
	suite.Require().Equal(uint32(lockDays), drRes.DenomReward.Lock)

	suite.drSecJoin(addrA, stakingDenom, 300)
	suite.drSecJoin(addrB, stakingDenom, 700) // T = 1000

	// --- money in: a 3-day schedule (700/day) and a 500 airdrop, same ibc prize denom ---
	schedRes, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(
		funder.String(), stakingDenom, prizeDenom, math.NewInt(700), "3",
	))
	suite.Require().NoError(err)
	scheduleId := schedRes.ScheduleId

	_, err = suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(
		funder.String(), stakingDenom, prizeDenom, math.NewInt(500),
	))
	suite.Require().NoError(err)
	suite.requirePrizeS(stakingDenom, prizeDenom, "0.5")

	// the airdrop is visible as pending at once: 300 × 0.5 and 700 × 0.5
	suite.Require().Equal("150", suite.slashPending(addrA, stakingDenom).AmountOf(prizeDenom).String())
	suite.Require().Equal("350", suite.slashPending(addrB, stakingDenom).AmountOf(prizeDenom).String())

	// --- day 1: S = 0.5 + 700/1000 = 1.2 ---
	suite.slashAdvanceDay(flows)
	suite.requirePrizeS(stakingDenom, prizeDenom, "1.2")

	schedules, err := suite.k.DenomRewardSchedules(suite.ctx, &types.QueryDenomRewardSchedulesRequest{Denom: stakingDenom})
	suite.Require().NoError(err)
	suite.Require().Len(schedules.List, 1)
	suite.Require().Equal(scheduleId, schedules.List[0].ScheduleId)
	suite.Require().Equal(uint32(1), schedules.List[0].Payouts)

	// the address-first marker resolves the slash denom back to the position
	positions, err := suite.k.DenomRewardParticipations(suite.ctx, &types.QueryDenomRewardParticipationsRequest{Address: addrA.String()})
	suite.Require().NoError(err)
	suite.Require().Len(positions.List, 1)
	suite.Require().Equal(stakingDenom, positions.List[0].StakingDenom)
	suite.Require().Equal("300", positions.List[0].Amount.String())

	paidA, err := suite.claimDr(addrA, stakingDenom)
	suite.Require().NoError(err)
	suite.Require().Equal("360", paidA.AmountOf(prizeDenom).String())
	paidB, err := suite.claimDr(addrB, stakingDenom)
	suite.Require().NoError(err)
	suite.Require().Equal("840", paidB.AmountOf(prizeDenom).String())

	// --- A exits under the 2-day lock: the pending-unlock key carries the slash denom ---
	suite.drSecExit(addrA, stakingDenom)
	unlockKeyA := fmt.Sprintf("%d/dr/%s/%s", flows.hour+lockDays*24, stakingDenom, addrA.String())
	pendingA, found := suite.k.GetPendingUnlockParticipant(suite.ctx, unlockKeyA)
	suite.Require().True(found, "pending unlock %q not found", unlockKeyA)
	suite.Require().Equal("300", pendingA.Amount)
	suite.Require().Equal(stakingDenom, pendingA.Denom)
	suite.Require().Equal(addrA.String(), pendingA.Address)

	_, found = suite.k.GetDenomRewardParticipant(suite.ctx, stakingDenom, addrA.String())
	suite.Require().False(found)
	suite.Require().False(hasDrParticipantMarker(*suite.k, suite.ctx, addrA.String(), stakingDenom))
	dr, _ := suite.k.GetDenomReward(suite.ctx, stakingDenom)
	suite.Require().Equal("700", dr.StakedAmount.String())
	suite.assertDrStoreIntegrity()

	// --- day 2 (T = 700): S = 2.2; B alone earns the whole 700 ---
	suite.slashAdvanceDay(flows)
	suite.requirePrizeS(stakingDenom, prizeDenom, "2.2")
	suite.Require().Equal("700", suite.slashPending(addrB, stakingDenom).AmountOf(prizeDenom).String())
	paidB, err = suite.claimDr(addrB, stakingDenom)
	suite.Require().NoError(err)
	suite.Require().Equal("700", paidB.AmountOf(prizeDenom).String())

	// --- day 3: the finishing payout deletes the schedule ---
	suite.slashAdvanceDay(flows)
	suite.requirePrizeS(stakingDenom, prizeDenom, "3.2")
	_, found = suite.k.GetDenomRewardSchedule(suite.ctx, stakingDenom, scheduleId)
	suite.Require().False(found)
	_, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardScheduleFinishEvent{}))
	suite.Require().True(ok)

	// B exits: the last 700 of prize is settled first, then the stake goes into the unlock queue
	outBefore := flows.out.AmountOf(prizeDenom)
	suite.drSecExit(addrB, stakingDenom)
	suite.Require().Equal("700", flows.out.AmountOf(prizeDenom).Sub(outBefore).String())
	unlockKeyB := fmt.Sprintf("%d/dr/%s/%s", flows.hour+lockDays*24, stakingDenom, addrB.String())
	_, found = suite.k.GetPendingUnlockParticipant(suite.ctx, unlockKeyB)
	suite.Require().True(found)

	// --- the hour epochs arrive: both stakes come back through the shared unlock pipeline ---
	suite.drainAllPendingUnlocks()
	suite.Require().Equal("1000", flows.out.AmountOf(stakingDenom).String())

	// escrow conservation, exact: stakes in == stakes out, prizes in == prizes out
	suite.Require().Equal("0", flows.in.AmountOf(stakingDenom).Sub(flows.out.AmountOf(stakingDenom)).String())
	suite.Require().Equal("2600", flows.in.AmountOf(prizeDenom).String()) // 500 airdrop + 3 × 700
	suite.Require().Equal("0", flows.in.AmountOf(prizeDenom).Sub(flows.out.AmountOf(prizeDenom)).String())

	// nothing dangling: no participants, markers or indexes; the DR and its accumulator remain
	suite.Require().Empty(suite.k.GetAllDenomRewardParticipant(suite.ctx))
	suite.Require().Empty(suite.k.GetAllDenomRewardParticipantIndex(suite.ctx))
	suite.assertDrStoreIntegrity()
	suite.Require().True(suite.k.HasDenomReward(suite.ctx, stakingDenom))
	suite.Require().Len(suite.k.GetAllDenomRewardPrizes(suite.ctx, stakingDenom), 1)
}
