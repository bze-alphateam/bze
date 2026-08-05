package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) setBoostTestParams(fee sdk.Coin, maxBoosts uint32) {
	params := types.DefaultParams()
	params.CreateBoostFee = fee
	params.MaxBoostsPerReward = maxBoosts
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) seedStakingReward(rewardId string, duration, payouts uint32) types.StakingReward {
	sr := types.StakingReward{
		RewardId:         rewardId,
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         duration,
		Payouts:          payouts,
		StakedAmount:     "0",
		DistributedStake: "0",
	}
	suite.k.SetStakingReward(suite.ctx, sr)

	return sr
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostSuccessWithFee() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(100)), 10)
	suite.seedStakingReward("000000000001", 30, 0)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uboost").Return(true).Times(1)
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(50000)), sdk.NewCoin("ubze", math.NewInt(1000)))).
		Times(1)
	//escrow = daily_amount 1000 x 10 days
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(10000)))).
		Return(nil).
		Times(1)
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, creator, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100))), types.ModuleName).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100))), nil).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, gomock.Any(), sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100)))).
		Return(nil).
		Times(1)

	msg := &types.MsgCreateBoost{
		Creator:     creator.String(),
		RewardId:    "000000000001",
		Denom:       "uboost",
		DailyAmount: "1000",
		Days:        "10",
	}

	response, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Equal("000000000001", response.BoostId)

	boost, found := suite.k.GetBoost(suite.ctx, "000000000001", response.BoostId)
	suite.Require().True(found)
	suite.Require().Equal("000000000001", boost.RewardId)
	suite.Require().Equal("uboost", boost.Denom)
	suite.Require().Equal("1000", boost.DailyAmount)
	suite.Require().Equal(uint32(10), boost.Duration)
	suite.Require().Equal(uint32(0), boost.Payouts)
	suite.Require().Equal("0", boost.DistributedStake)
	suite.Require().Equal(creator.String(), boost.Creator)
	suite.Require().Equal(uint64(1), suite.k.GetBoostsCounter(suite.ctx))
}

// TestMsgServerBoost_CreateBoostIdsIncrement creates two boosts with the SAME
// denom on one reward: both must be created (no uniqueness rule) with
// incrementing ids.
func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostIdsIncrement() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)
	suite.seedStakingReward("000000000001", 30, 0)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uboost").Return(true).Times(2)
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(50000)))).
		Times(2)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(5000)))).
		Return(nil).
		Times(2)

	msg := &types.MsgCreateBoost{
		Creator:     creator.String(),
		RewardId:    "000000000001",
		Denom:       "uboost",
		DailyAmount: "1000",
		Days:        "5",
	}

	first, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().Equal("000000000001", first.BoostId)

	second, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().Equal("000000000002", second.BoostId)

	//both same-denom boosts coexist
	suite.Require().Len(suite.k.GetRewardBoosts(suite.ctx, "000000000001"), 2)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostNilRequest() {
	response, err := suite.msgServer.CreateBoost(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(sdkerrors.ErrInvalidRequest, err)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostInvalidCreator() {
	msg := &types.MsgCreateBoost{
		Creator:     "invalid-address",
		RewardId:    "000000000001",
		Denom:       "uboost",
		DailyAmount: "1000",
		Days:        "10",
	}

	response, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostMissingReward() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)

	msg := &types.MsgCreateBoost{
		Creator:     creator.String(),
		RewardId:    "000000000009",
		Denom:       "uboost",
		DailyAmount: "1000",
		Days:        "10",
	}

	response, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrInvalidRewardId)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostDenomWithoutSupply() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)
	suite.seedStakingReward("000000000001", 30, 0)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uboost").Return(false).Times(1)

	msg := &types.MsgCreateBoost{
		Creator:     creator.String(),
		RewardId:    "000000000001",
		Denom:       "uboost",
		DailyAmount: "1000",
		Days:        "10",
	}

	response, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrInvalidBoostDenom)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostInvalidDays() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)
	suite.seedStakingReward("000000000001", 30, 0)

	for _, days := range []string{"0", "-3", "not-a-number"} {
		msg := &types.MsgCreateBoost{
			Creator:     creator.String(),
			RewardId:    "000000000001",
			Denom:       "uboost",
			DailyAmount: "1000",
			Days:        days,
		}

		response, err := suite.msgServer.CreateBoost(suite.ctx, msg)
		suite.Require().Error(err, "days=%s", days)
		suite.Require().Nil(response)
		suite.Require().ErrorIs(err, types.ErrInvalidBoostDays)
	}
}

// TestMsgServerBoost_CreateBoostDaysTooLong covers the fresh-parent case and
// the payouts > 0 case: days must fit parent.duration - parent.payouts.
func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostDaysTooLong() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)
	suite.seedStakingReward("000000000001", 30, 0)
	suite.seedStakingReward("000000000002", 30, 10)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uboost").Return(true).AnyTimes()

	//fresh parent: 31 > 30 - 0
	msg := &types.MsgCreateBoost{
		Creator:     creator.String(),
		RewardId:    "000000000001",
		Denom:       "uboost",
		DailyAmount: "1000",
		Days:        "31",
	}
	response, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrInvalidBoostDays)

	//partially paid parent: 21 > 30 - 10
	msg.RewardId = "000000000002"
	msg.Days = "21"
	response, err = suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrInvalidBoostDays)

	//exactly the remainder is accepted: 20 == 30 - 10
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(50000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(20000)))).
		Return(nil).
		Times(1)
	msg.Days = "20"
	response, err = suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
}

// TestMsgServerBoost_CreateBoostCapReached rejects creation at exactly the cap
// (finished boosts count too).
func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostCapReached() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)
	suite.seedStakingReward("000000000001", 30, 0)

	for i := 1; i <= 10; i++ {
		boost := newBoost("000000000001", fmt.Sprintf("%012d", i), "uboost")
		if i == 1 {
			//a finished boost occupies a slot as well
			boost.Payouts = boost.Duration
		}
		suite.k.SetBoost(suite.ctx, boost)
	}

	suite.bank.EXPECT().HasSupply(suite.ctx, "uboost").Return(true).Times(1)

	msg := &types.MsgCreateBoost{
		Creator:     creator.String(),
		RewardId:    "000000000001",
		Denom:       "uboost",
		DailyAmount: "1000",
		Days:        "10",
	}

	response, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrBoostCapReached)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostInvalidAmount() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)
	suite.seedStakingReward("000000000001", 30, 0)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uboost").Return(true).Times(2)

	for _, amount := range []string{"0", "-1000"} {
		msg := &types.MsgCreateBoost{
			Creator:     creator.String(),
			RewardId:    "000000000001",
			Denom:       "uboost",
			DailyAmount: amount,
			Days:        "10",
		}

		response, err := suite.msgServer.CreateBoost(suite.ctx, msg)
		suite.Require().Error(err, "amount=%s", amount)
		suite.Require().Nil(response)
		suite.Require().ErrorIs(err, types.ErrInvalidAmount)
	}
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_CreateBoostInsufficientFunds() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(100)), 10)
	suite.seedStakingReward("000000000001", 30, 0)

	suite.bank.EXPECT().HasSupply(suite.ctx, "uboost").Return(true).Times(1)
	//user holds the escrow but not the fee on top of it
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(10000)))).
		Times(1)

	msg := &types.MsgCreateBoost{
		Creator:     creator.String(),
		RewardId:    "000000000001",
		Denom:       "uboost",
		DailyAmount: "1000",
		Days:        "10",
	}

	response, err := suite.msgServer.CreateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)

	//nothing was persisted and no id was consumed
	suite.Require().Empty(suite.k.GetRewardBoosts(suite.ctx, "000000000001"))
	suite.Require().Equal(uint64(0), suite.k.GetBoostsCounter(suite.ctx))
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_UpdateBoostSuccess() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(100)), 10)
	suite.seedStakingReward("000000000001", 30, 10)

	boost := newBoost("000000000001", "000000000001", "uboost")
	boost.Payouts = 5
	boost.DistributedStake = "1.5"
	boost.Duration = 10
	suite.k.SetBoost(suite.ctx, boost)

	//extension is fee-free: escrow only, daily_amount 1000 x 5 extra days
	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(5000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(5000)))).
		Return(nil).
		Times(1)

	msg := &types.MsgUpdateBoost{
		Creator:  creator.String(),
		RewardId: "000000000001",
		BoostId:  "000000000001",
		Days:     "5",
	}

	response, err := suite.msgServer.UpdateBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	updated, found := suite.k.GetBoost(suite.ctx, "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(15), updated.Duration)
	suite.Require().Equal(uint32(5), updated.Payouts)
	suite.Require().Equal("1.5", updated.DistributedStake)
}

// TestMsgServerBoost_UpdateBoostReArmsFinished extends a finished boost:
// payouts == duration before, payouts < duration after, accumulator untouched.
func (suite *IntegrationTestSuite) TestMsgServerBoost_UpdateBoostReArmsFinished() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)
	suite.seedStakingReward("000000000001", 30, 10)

	boost := newBoost("000000000001", "000000000001", "uboost")
	boost.Duration = 10
	boost.Payouts = 10
	boost.DistributedStake = "2.75"
	suite.k.SetBoost(suite.ctx, boost)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(3000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(3000)))).
		Return(nil).
		Times(1)

	msg := &types.MsgUpdateBoost{
		Creator:  creator.String(),
		RewardId: "000000000001",
		BoostId:  "000000000001",
		Days:     "3",
	}

	response, err := suite.msgServer.UpdateBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	updated, found := suite.k.GetBoost(suite.ctx, "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(13), updated.Duration)
	suite.Require().True(updated.Payouts < updated.Duration)
	suite.Require().Equal("2.75", updated.DistributedStake)
}

// TestMsgServerBoost_UpdateBoostAllowedAtCap proves extension consumes no
// slot: it succeeds while the reward sits at exactly max_boosts_per_reward.
func (suite *IntegrationTestSuite) TestMsgServerBoost_UpdateBoostAllowedAtCap() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)
	suite.seedStakingReward("000000000001", 30, 0)

	for i := 1; i <= 10; i++ {
		boost := newBoost("000000000001", fmt.Sprintf("%012d", i), "uboost")
		if i == 5 {
			//leave headroom under the parent's remainder for the extension
			boost.Duration = 10
		}
		suite.k.SetBoost(suite.ctx, boost)
	}

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(2000)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(2000)))).
		Return(nil).
		Times(1)

	msg := &types.MsgUpdateBoost{
		Creator:  creator.String(),
		RewardId: "000000000001",
		BoostId:  "000000000005",
		Days:     "2",
	}

	response, err := suite.msgServer.UpdateBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_UpdateBoostNilRequest() {
	response, err := suite.msgServer.UpdateBoost(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Equal(sdkerrors.ErrInvalidRequest, err)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_UpdateBoostMissingReward() {
	creator := sdk.AccAddress("creator")

	msg := &types.MsgUpdateBoost{
		Creator:  creator.String(),
		RewardId: "000000000009",
		BoostId:  "000000000001",
		Days:     "5",
	}

	response, err := suite.msgServer.UpdateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrInvalidRewardId)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_UpdateBoostMissingBoost() {
	creator := sdk.AccAddress("creator")
	suite.seedStakingReward("000000000001", 30, 0)

	msg := &types.MsgUpdateBoost{
		Creator:  creator.String(),
		RewardId: "000000000001",
		BoostId:  "000000000009",
		Days:     "5",
	}

	response, err := suite.msgServer.UpdateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrInvalidBoostId)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_UpdateBoostInvalidDays() {
	creator := sdk.AccAddress("creator")
	suite.seedStakingReward("000000000001", 30, 0)
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000001", "uboost"))

	for _, days := range []string{"0", "-3", "not-a-number"} {
		msg := &types.MsgUpdateBoost{
			Creator:  creator.String(),
			RewardId: "000000000001",
			BoostId:  "000000000001",
			Days:     days,
		}

		response, err := suite.msgServer.UpdateBoost(suite.ctx, msg)
		suite.Require().Error(err, "days=%s", days)
		suite.Require().Nil(response)
		suite.Require().ErrorIs(err, types.ErrInvalidBoostDays)
	}
}

// TestMsgServerBoost_UpdateBoostOverExtension covers A12: the extended
// remainder must not exceed the parent's remaining payouts, including the
// parent.payouts > 0 case. The boost must stay unchanged after the rejection.
func (suite *IntegrationTestSuite) TestMsgServerBoost_UpdateBoostOverExtension() {
	creator := sdk.AccAddress("creator")
	suite.setBoostTestParams(sdk.NewCoin("ubze", math.NewInt(0)), 10)
	//parent remaining = 30 - 10 = 20
	suite.seedStakingReward("000000000001", 30, 10)

	//boost remaining = 10 - 0 = 10; extending by 11 makes it 21 > 20
	boost := newBoost("000000000001", "000000000001", "uboost")
	boost.Duration = 10
	suite.k.SetBoost(suite.ctx, boost)

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(50000)))).
		Times(2)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, gomock.Any()).
		Return(nil).
		Times(2)

	msg := &types.MsgUpdateBoost{
		Creator:  creator.String(),
		RewardId: "000000000001",
		BoostId:  "000000000001",
		Days:     "11",
	}

	response, err := suite.msgServer.UpdateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().ErrorIs(err, types.ErrInvalidBoostDays)

	//the store still holds the pre-extension schedule
	unchanged, found := suite.k.GetBoost(suite.ctx, "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(uint32(10), unchanged.Duration)

	//exactly the parent's remainder is accepted: 10 + 10 = 20
	msg.Days = "10"
	response, err = suite.msgServer.UpdateBoost(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
}

func (suite *IntegrationTestSuite) TestMsgServerBoost_UpdateBoostInsufficientFunds() {
	creator := sdk.AccAddress("creator")
	suite.seedStakingReward("000000000001", 30, 0)
	suite.k.SetBoost(suite.ctx, newBoost("000000000001", "000000000001", "uboost"))

	suite.bank.EXPECT().
		SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("uboost", math.NewInt(1)))).
		Times(1)

	msg := &types.MsgUpdateBoost{
		Creator:  creator.String(),
		RewardId: "000000000001",
		BoostId:  "000000000001",
		Days:     "5",
	}

	response, err := suite.msgServer.UpdateBoost(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
}
