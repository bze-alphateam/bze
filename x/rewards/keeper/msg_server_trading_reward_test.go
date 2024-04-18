package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/rewards/types"
	types2 "github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (suite *IntegrationTestSuite) TestCreateTradingReward_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	_, err := suite.msgServer.CreateTradingReward(goCtx, nil)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

func (suite *IntegrationTestSuite) TestCreateTradingReward_InvalidCreator() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	req := &types.MsgCreateTradingReward{Creator: ""}

	_, err := suite.msgServer.CreateTradingReward(goCtx, req)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCreateTradingReward_InvalidTradingReward() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	tests := []struct {
		name string
		msg  types.MsgCreateTradingReward
	}{
		{
			name: "empty prize amount",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "",
			},
		},
		{
			name: "zero prize amount",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "0",
			},
		},
		{
			name: "negative prize amount",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "-10",
			},
		},
		{
			name: "empty prize denom",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "10",
				PrizeDenom:  "",
			},
		},
		{
			name: "missing market id",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "10",
				PrizeDenom:  "ubze",
				MarketId:    "not_a_market_id",
			},
		},
		{
			name: "invalid duration",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "10",
				PrizeDenom:  "ubze",
				MarketId:    "stake/ubze",
				Duration:    "",
			},
		},
		{
			name: "zero duration",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "10",
				PrizeDenom:  "ubze",
				Duration:    "0",
				MarketId:    "stake/ubze",
			},
		},
		{
			name: "negative duration",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "10",
				PrizeDenom:  "ubze",
				Duration:    "-11",
				MarketId:    "stake/ubze",
			},
		},
		{
			name: "duration too high",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "10",
				PrizeDenom:  "ubze",
				Duration:    "3213132131231",
				MarketId:    "stake/ubze",
			},
		},
		{
			name: "invalid slots",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "10",
				PrizeDenom:  "ubze",
				Duration:    "100",
				MarketId:    "stake/ubze",
				Slots:       "",
			},
		},
		{
			name: "zero slots",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "10",
				PrizeDenom:  "ubze",
				Duration:    "100",
				MarketId:    "stake/ubze",
				Slots:       "0",
			},
		},
		{
			name: "negative slots",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "10",
				PrizeDenom:  "ubze",
				Duration:    "100",
				MarketId:    "stake/ubze",
				Slots:       "-3",
			},
		},
		{
			name: "too many slots slots",
			msg: types.MsgCreateTradingReward{
				Creator:     addr1.String(),
				PrizeAmount: "10",
				PrizeDenom:  "ubze",
				Duration:    "100",
				MarketId:    "stake/ubze",
				Slots:       "101",
			},
		},
	}

	for _, tt := range tests {
		_, err := suite.msgServer.CreateTradingReward(goCtx, &tt.msg)
		suite.Require().NotNil(err, tt.name)
	}
}

func (suite *IntegrationTestSuite) TestCreateTradingReward_MissingSupply() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	msg := types.MsgCreateTradingReward{
		Creator:     addr1.String(),
		PrizeAmount: "10",
		PrizeDenom:  "ubzesd",
		Duration:    "100",
		MarketId:    "stake/ubze",
		Slots:       "1",
	}

	_, err := suite.msgServer.CreateTradingReward(goCtx, &msg)
	suite.Require().ErrorIs(err, types.ErrInvalidPrizeDenom)
}

func (suite *IntegrationTestSuite) TestCreateTradingReward_NotEnoughBalance() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	//initial balances need to be 0
	initialUserBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(initialUserBalances.IsZero())

	balances := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	//create market
	suite.app.TradebinKeeper.SetMarket(suite.ctx, types2.Market{
		Base:    "stake",
		Quote:   "ubze",
		Creator: "",
	})

	msg := types.MsgCreateTradingReward{
		Creator:     addr1.String(),
		PrizeAmount: "10",
		PrizeDenom:  "ubze",
		Duration:    "100",
		MarketId:    "stake/ubze",
		Slots:       "100",
	}

	_, err := suite.msgServer.CreateTradingReward(goCtx, &msg)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestCreateTradingReward_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	//initial balances need to be 0
	initialUserBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(initialUserBalances.IsZero())

	balances := sdk.NewCoins(sdk.NewInt64Coin("ubze", 10000002000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	suite.app.TradebinKeeper.SetMarket(suite.ctx, types2.Market{
		Base:    "stake",
		Quote:   "ubze",
		Creator: "",
	})

	msg := types.MsgCreateTradingReward{
		Creator:     addr1.String(),
		PrizeAmount: "200",
		PrizeDenom:  "ubze",
		Duration:    "100",
		MarketId:    "stake/ubze",
		Slots:       "10",
	}

	res, err := suite.msgServer.CreateTradingReward(goCtx, &msg)
	suite.Require().NoError(err)

	storeTradingReward, ok := suite.k.GetPendingTradingReward(suite.ctx, res.RewardId)
	suite.Require().True(ok)

	suite.Require().EqualValues(msg.PrizeAmount, storeTradingReward.PrizeAmount)
	suite.Require().EqualValues(msg.PrizeDenom, storeTradingReward.PrizeDenom)
	suite.Require().EqualValues(uint32(100), storeTradingReward.Duration)
	suite.Require().EqualValues(msg.MarketId, storeTradingReward.MarketId)
	suite.Require().EqualValues(uint32(10), storeTradingReward.Slots)

	expectedRemainingBalance := sdk.NewCoins(sdk.NewInt64Coin("ubze", 0))
	actualRemainingBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(actualRemainingBalance.IsEqual(expectedRemainingBalance))
}
