package keeper_test

import (
	"cosmossdk.io/math"
	keeper2 "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"go.uber.org/mock/gomock"
)

// setBoostParams overwrites params with a controlled boost creation fee and cap.
// A fee <= 0 disables the fee flow (no trade keeper interaction).
func (suite *IntegrationTestSuite) setBoostParams(fee int64, cap uint32) {
	p := types.DefaultParams()
	if fee <= 0 {
		p.CreateBoostFee = sdk.NewCoin("ubze", math.ZeroInt())
	} else {
		p.CreateBoostFee = sdk.NewCoin("ubze", math.NewInt(fee))
	}
	p.MaxBoostsPerReward = cap
	suite.Require().NoError(suite.k.SetParams(suite.ctx, p))
}

// seedReward stores a staking reward with the given remaining-emissions window.
func (suite *IntegrationTestSuite) seedReward(id string, duration, payouts uint32) {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         id,
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         duration,
		Payouts:          payouts,
		MinStake:         1,
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
	})
}

func (suite *IntegrationTestSuite) TestCreateBoost_Success_WithFee() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0)
	suite.setBoostParams(100, 10)

	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000000)))).Times(1)
	// escrow = daily_amount(1000) * days(3) = 3000
	suite.bank.EXPECT().SendCoinsFromAccountToModule(
		suite.ctx, creator, types.ModuleName,
		sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(3000))),
	).Return(nil).Times(1)
	// fee capture + swap of 100 ubze
	suite.trade.EXPECT().CaptureAndSwapUserFee(
		suite.ctx, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100))), types.ModuleName,
	).Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100))), nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromModuleToModule(
		suite.ctx, types.ModuleName, gomock.Any(), sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100))),
	).Return(nil).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "1000", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Equal(uint64(1), resp.Uid)

	boost, found := suite.k.GetBoost(suite.ctx, "000000000001", "ubze")
	suite.Require().True(found)
	suite.Require().Equal(uint64(1), boost.Uid)
	suite.Require().Equal("1000", boost.DailyAmount)
	suite.Require().Equal(uint32(3), boost.DaysLeft)
	suite.Require().Equal("0", boost.SBoost)
	suite.Require().Equal("", boost.FinalizeCursor)
	suite.Require().Equal(creator.String(), boost.Creator)
	suite.Require().Equal(uint64(1), suite.k.GetBoostCounter(suite.ctx))
}

func (suite *IntegrationTestSuite) TestCreateBoost_NilRequest() {
	resp, err := suite.msgServer.CreateBoost(suite.ctx, nil)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_InvalidCreator() {
	suite.seedReward("000000000001", 5, 0)
	msg := types.NewMsgCreateBoost("not-an-address", "000000000001", "ubze", "1000", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_RewardNotFound() {
	creator := sdk.AccAddress("creator")
	msg := types.NewMsgCreateBoost(creator.String(), "000000000099", "ubze", "1000", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidRewardId)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_DenomWithoutSupply() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0)
	suite.bank.EXPECT().HasSupply(suite.ctx, "nosupply").Return(false).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "nosupply", "1000", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidBoostDenom)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_DaysExceedRemaining_PayoutsZero() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0) // remaining = 5
	suite.setBoostParams(0, 10)
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "1000", "6")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidBoostDays)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_DaysExceedRemaining_PayoutsPositive() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 3) // remaining = 2
	suite.setBoostParams(0, 10)
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "1000", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidBoostDays)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_ZeroDays() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0)
	suite.setBoostParams(0, 10)
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "1000", "0")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidBoostDays)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_DuplicateRecord() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0)
	suite.setBoostParams(0, 10)
	// an existing record for the same (reward, denom) blocks creation
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "ubze", 7))
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "1000", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrBoostAlreadyExists)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_CapReachedAtExactlyTen() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0)
	suite.setBoostParams(0, 10)
	// seed exactly 10 existing boosts with distinct denoms
	for i := 0; i < 10; i++ {
		suite.k.SetBoost(suite.ctx, newBoost("000000000001", "seed"+string(rune('0'+i)), uint64(i+1)))
	}
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "1000", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrBoostCapReached)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_AllowedAtNineExisting() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0)
	suite.setBoostParams(0, 10) // fee disabled -> no trade mocks
	for i := 0; i < 9; i++ {
		suite.k.SetBoost(suite.ctx, newBoost("000000000001", "seed"+string(rune('0'+i)), uint64(i+1)))
	}
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000000)))).Times(1)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(
		suite.ctx, creator, types.ModuleName,
		sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(3000))),
	).Return(nil).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "1000", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Len(suite.k.GetRewardBoosts(suite.ctx, "000000000001"), 10)
}

func (suite *IntegrationTestSuite) TestCreateBoost_ZeroDailyAmount() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0)
	suite.setBoostParams(0, 10)
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "0", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidAmount)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_NegativeDailyAmount() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0)
	suite.setBoostParams(0, 10)
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "-5", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidAmount)
	suite.Require().Nil(resp)
}

func (suite *IntegrationTestSuite) TestCreateBoost_InsufficientFunds() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0)
	suite.setBoostParams(0, 10) // no fee
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)
	// escrow 3000 needed, only 100 spendable
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100)))).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "1000", "3")
	resp, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(resp)
	suite.Require().Contains(err.Error(), "user balance is too low")
}

func (suite *IntegrationTestSuite) TestCreateBoost_NilTradeKeeperWithFee() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	mockEpoch := testutil.NewMockEpochKeeper(mockCtrl)
	mockAcc := testutil.NewMockAccountKeeper(mockCtrl)

	k, ctx := keeper2.RewardsKeeper(t, mockBank, mockEpoch, nil, mockAcc)
	msgServer := keeper.NewMsgServerImpl(k)

	creator := sdk.AccAddress("creator")
	k.SetStakingReward(ctx, types.StakingReward{
		RewardId: "000000000001", PrizeAmount: "1000", PrizeDenom: "ubze",
		StakingDenom: "ubze", Duration: 5, Payouts: 0, MinStake: 1, Lock: 7,
		StakedAmount: "0", DistributedStake: "0",
	})
	p := types.DefaultParams()
	p.CreateBoostFee = sdk.NewCoin("ubze", math.NewInt(100))
	p.MaxBoostsPerReward = 10
	suite.Require().NoError(k.SetParams(ctx, p))

	mockBank.EXPECT().HasSupply(ctx, "ubze").Return(true).Times(1)
	mockBank.EXPECT().SpendableCoins(ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000000)))).Times(1)
	mockBank.EXPECT().SendCoinsFromAccountToModule(
		ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(3000))),
	).Return(nil).Times(1)

	msg := types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "1000", "3")
	resp, err := msgServer.CreateBoost(ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(resp)
	suite.Require().Contains(err.Error(), "trade keeper is not available")
}

func (suite *IntegrationTestSuite) TestCreateBoost_UidIncrementsAcrossCreations() {
	creator := sdk.AccAddress("creator")
	suite.seedReward("000000000001", 5, 0)
	suite.setBoostParams(0, 10) // no fee

	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)
	suite.bank.EXPECT().HasSupply(suite.ctx, "uother").Return(true).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000000)), sdk.NewCoin("uother", math.NewInt(1000000)))).Times(2)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(
		suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(3000))),
	).Return(nil).Times(1)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(
		suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uother", math.NewInt(3000))),
	).Return(nil).Times(1)

	resp1, err := suite.msgServer.CreateBoost(suite.ctx, types.NewMsgCreateBoost(creator.String(), "000000000001", "ubze", "1000", "3"))
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1), resp1.Uid)

	resp2, err := suite.msgServer.CreateBoost(suite.ctx, types.NewMsgCreateBoost(creator.String(), "000000000001", "uother", "1000", "3"))
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(2), resp2.Uid)

	suite.Require().Equal(uint64(2), suite.k.GetBoostCounter(suite.ctx))
}
