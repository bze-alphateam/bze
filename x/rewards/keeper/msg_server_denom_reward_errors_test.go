package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	keeper2 "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"github.com/bze-alphateam/bze/x/rewards/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// Error branches of the denom reward handlers: every bank / trade-keeper / epoch failure must
// surface as the handler's error and must stop the handler before it writes the record the
// failed transfer was paying for. (Transfers that already happened earlier in the same handler
// are rolled back by the tx; what these tests pin is that no state is written after a failure.)

var errMockFailure = fmt.Errorf("mock failure")

// --- CreateDenomReward ---

func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_FeeCaptureError() {
	creator := sdk.AccAddress("dr-err-creator-1")
	fee := sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(25000)))
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{CreateDenomRewardFee: fee[0]}))

	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)
	suite.richBalance(creator)
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, creator, fee, types.ModuleName).
		Return(nil, errMockFailure).Times(1)

	_, err := suite.msgServer.CreateDenomReward(suite.ctx, types.NewMsgCreateDenomReward(creator.String(), "ubze"))
	suite.Require().ErrorIs(err, errMockFailure)
	suite.Require().False(suite.k.HasDenomReward(suite.ctx, "ubze"))
}

func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_FeeForwardError() {
	creator := sdk.AccAddress("dr-err-creator-2")
	fee := sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(25000)))
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{CreateDenomRewardFee: fee[0]}))

	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)
	suite.richBalance(creator)
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, creator, fee, types.ModuleName).
		Return(fee, nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, fee).
		Return(errMockFailure).Times(1)

	_, err := suite.msgServer.CreateDenomReward(suite.ctx, types.NewMsgCreateDenomReward(creator.String(), "ubze"))
	suite.Require().ErrorIs(err, errMockFailure)
	suite.Require().False(suite.k.HasDenomReward(suite.ctx, "ubze"))
}

// --- JoinDenomReward ---

func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_EscrowError() {
	joiner := sdk.AccAddress("dr-err-joiner-1")
	suite.seedDenomReward("ubze", 1000)
	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 500))

	suite.bank.EXPECT().SpendableCoins(suite.ctx, joiner).Return(coins).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, joiner, types.ModuleName, coins).
		Return(errMockFailure).Times(1)

	_, err := suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(500)))
	suite.Require().ErrorIs(err, errMockFailure)

	_, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", joiner.String())
	suite.Require().False(found)
	suite.Require().False(hasDrParticipantMarker(*suite.k, suite.ctx, joiner.String(), "ubze"))
	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("1000", dr.StakedAmount.String())
}

// --- DistributeDenomRewards ---

func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_ExistingPrize_EscrowError() {
	sender := sdk.AccAddress("dr-err-sender-1")
	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uprize", 50))

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(sender)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, sender, types.ModuleName, coins).
		Return(errMockFailure).Times(1)

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(50)))
	suite.Require().ErrorIs(err, errMockFailure)
	suite.requirePrizeS("ubze", "uprize", "0")
}

func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_NewPrize_FeeCaptureError() {
	sender := sdk.AccAddress("dr-err-sender-2")
	suite.setDrMoneyInParams(50)
	suite.seedDenomReward("ubze", 100)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(sender)
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, sender, drPrizeFee, types.ModuleName).
		Return(nil, errMockFailure).Times(1)

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(50)))
	suite.Require().ErrorIs(err, errMockFailure)
	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().False(found, "a failed prize fee must not create the accumulator")
}

func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_NewPrize_FeeForwardError() {
	sender := sdk.AccAddress("dr-err-sender-3")
	suite.setDrMoneyInParams(50)
	suite.seedDenomReward("ubze", 100)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(sender)
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, sender, drPrizeFee, types.ModuleName).
		Return(drPrizeFee, nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, drPrizeFee).
		Return(errMockFailure).Times(1)

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(50)))
	suite.Require().ErrorIs(err, errMockFailure)
	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().False(found)
}

// --- CreateDenomRewardSchedule ---

func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_ScheduleFeeCaptureError() {
	funder := sdk.AccAddress("dr-err-funder-1")
	suite.setDrMoneyInParams(50)
	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(funder)
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, funder, drScheduleFee, types.ModuleName).
		Return(nil, errMockFailure).Times(1)

	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(funder.String(), "ubze", "uprize", math.NewInt(100), "5"))
	suite.Require().ErrorIs(err, errMockFailure)
	suite.Require().Empty(suite.k.GetAllDenomRewardSchedule(suite.ctx))
	suite.Require().Equal(uint64(0), suite.k.GetDenomRewardScheduleCounter(suite.ctx))
}

func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_ScheduleFeeForwardError() {
	funder := sdk.AccAddress("dr-err-funder-2")
	suite.setDrMoneyInParams(50)
	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(funder)
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, funder, drScheduleFee, types.ModuleName).
		Return(drScheduleFee, nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, drScheduleFee).
		Return(errMockFailure).Times(1)

	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(funder.String(), "ubze", "uprize", math.NewInt(100), "5"))
	suite.Require().ErrorIs(err, errMockFailure)
	suite.Require().Empty(suite.k.GetAllDenomRewardSchedule(suite.ctx))
}

func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_BudgetEscrowError() {
	funder := sdk.AccAddress("dr-err-funder-3")
	suite.setDrMoneyInParams(50)
	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)
	budget := sdk.NewCoins(sdk.NewInt64Coin("uprize", 500))

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(funder)
	suite.expectFeeCapture(funder, drScheduleFee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, funder, types.ModuleName, budget).
		Return(errMockFailure).Times(1)

	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(funder.String(), "ubze", "uprize", math.NewInt(100), "5"))
	suite.Require().ErrorIs(err, errMockFailure)
	suite.Require().Empty(suite.k.GetAllDenomRewardSchedule(suite.ctx))
	suite.Require().Equal(uint64(0), suite.k.GetDenomRewardScheduleCounter(suite.ctx))
}

// --- UpdateDenomRewardSchedule ---

func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Update_EscrowError() {
	funder := sdk.AccAddress("dr-err-funder-4")
	suite.seedDenomRewardWithPrize("ubze", "uprize", 100)
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId: "000000000001", StakingDenom: "ubze", PrizeDenom: "uprize",
		DailyAmount: math.NewInt(100), Duration: 5, Payouts: 1,
	})
	extra := sdk.NewCoins(sdk.NewInt64Coin("uprize", 200))

	suite.richBalance(funder)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, funder, types.ModuleName, extra).
		Return(errMockFailure).Times(1)

	_, err := suite.msgServer.UpdateDenomRewardSchedule(suite.ctx, types.NewMsgUpdateDenomRewardSchedule(funder.String(), "ubze", "000000000001", "2"))
	suite.Require().ErrorIs(err, errMockFailure)
	schedule, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(5), schedule.Duration)
}

// --- ExitDenomReward ---

func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_LockZero_StakeReturnError() {
	exiter := sdk.AccAddress("dr-err-exiter-1")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 0, StakedAmount: math.NewInt(500)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: exiter.String(), StakingDenom: "ubze", Amount: math.NewInt(500)})

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, exiter, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(errMockFailure).Times(1)

	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().ErrorIs(err, errMockFailure)

	// the position is only erased after the stake left the module: still fully intact here
	participant, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", exiter.String())
	suite.Require().True(found)
	suite.Require().Equal("500", participant.Amount.String())
	suite.Require().True(hasDrParticipantMarker(*suite.k, suite.ctx, exiter.String(), "ubze"))
	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("500", dr.StakedAmount.String())
}

func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_LockPositive_EpochError() {
	exiter := sdk.AccAddress("dr-err-exiter-2")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, StakedAmount: math.NewInt(500)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: exiter.String(), StakingDenom: "ubze", Amount: math.NewInt(500)})

	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(0), errMockFailure).Times(1)

	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().ErrorIs(err, errMockFailure)

	_, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", exiter.String())
	suite.Require().True(found)
	suite.Require().Empty(suite.k.GetAllPendingUnlockParticipant(suite.ctx))
	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("500", dr.StakedAmount.String())
}

// --- nil trade keeper with a positive fee ---

// nilTradeKeeperSetup builds a keeper without a trade keeper, the configuration in which a
// positive fee param cannot be captured. Mirrors TestMsgServerDenomReward_CreateDenomReward_NilTradeKeeper.
func (suite *IntegrationTestSuite) nilTradeKeeperSetup() (keeper.Keeper, sdk.Context, types.MsgServer, *testutil.MockBankKeeper) {
	t := suite.T()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	mockEpoch := testutil.NewMockEpochKeeper(mockCtrl)
	mockAcc := testutil.NewMockAccountKeeper(mockCtrl)

	k, ctx := keeper2.RewardsKeeper(t, mockBank, mockEpoch, nil, mockAcc)

	return k, ctx, keeper.NewMsgServerImpl(k), mockBank
}

func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_NewPrize_NilTradeKeeper() {
	k, ctx, msgServer, bank := suite.nilTradeKeeperSetup()
	sender := sdk.AccAddress("dr-err-sender-4")
	suite.Require().NoError(k.SetParams(ctx, types.Params{
		CreateDenomRewardPrizeFee: sdk.NewCoin("ubze", math.NewInt(25_000)),
		MaxPrizeDenomsPerDr:       50,
	}))
	k.SetDenomReward(ctx, types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(100)})

	bank.EXPECT().HasSupply(ctx, "uprize").Return(true).Times(1)
	bank.EXPECT().SpendableCoins(ctx, sender).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1_000_000)), sdk.NewCoin("uprize", math.NewInt(1_000_000)))).AnyTimes()

	_, err := msgServer.DistributeDenomRewards(ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(50)))
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "trade keeper is not available")
	_, found := k.GetDenomRewardPrize(ctx, "ubze", "uprize")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_ExistingPrize_NilTradeKeeper() {
	k, ctx, msgServer, bank := suite.nilTradeKeeperSetup()
	funder := sdk.AccAddress("dr-err-funder-5")
	suite.Require().NoError(k.SetParams(ctx, types.Params{
		AddDenomRewardScheduleFee: sdk.NewCoin("ubze", math.NewInt(5_000)),
		MaxPrizeDenomsPerDr:       50,
	}))
	k.SetDenomReward(ctx, types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(100)})
	k.SetDenomRewardPrize(ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyZeroDec()})

	bank.EXPECT().HasSupply(ctx, "uprize").Return(true).Times(1)
	bank.EXPECT().SpendableCoins(ctx, funder).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1_000_000)), sdk.NewCoin("uprize", math.NewInt(1_000_000)))).AnyTimes()

	_, err := msgServer.CreateDenomRewardSchedule(ctx, types.NewMsgCreateDenomRewardSchedule(funder.String(), "ubze", "uprize", math.NewInt(100), "5"))
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "trade keeper is not available")
	suite.Require().Empty(k.GetAllDenomRewardSchedule(ctx))
}
