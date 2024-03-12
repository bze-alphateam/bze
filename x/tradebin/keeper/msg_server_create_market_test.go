package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestCreateMarket_InvalidDenom() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	//empty base denom
	_, err := suite.msgServer.CreateMarket(goCtx, &types.MsgCreateMarket{
		Creator: "me",
		Base:    "",
		Quote:   denomBze,
	})
	suite.Require().NotNil(err)

	//empty quote denom
	_, err = suite.msgServer.CreateMarket(goCtx, &types.MsgCreateMarket{
		Creator: "me",
		Base:    denomBze,
		Quote:   "",
	})
	suite.Require().NotNil(err)

	//same denom for both
	_, err = suite.msgServer.CreateMarket(goCtx, &types.MsgCreateMarket{
		Creator: "me",
		Base:    denomBze,
		Quote:   denomBze,
	})
	suite.Require().NotNil(err)

	//denom has no supply
	_, err = suite.msgServer.CreateMarket(goCtx, &types.MsgCreateMarket{
		Creator: "me",
		Base:    denomStake,
		Quote:   denomBze,
	})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCreateMarket_MarketAlreadyExist() {
	suite.k.SetMarket(suite.ctx, market)
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CreateMarket(goCtx, &types.MsgCreateMarket{
		Creator: "me",
		Base:    denomStake,
		Quote:   denomBze,
	})
	suite.Require().NotNil(err)

	_, err = suite.msgServer.CreateMarket(goCtx, &types.MsgCreateMarket{
		Creator: "me",
		Base:    denomBze,
		Quote:   denomStake,
	})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCreateMarket_NotEnoughCoinsForFee() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	//initial balances need to be 0
	initialUserBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(initialUserBalances.IsZero())

	balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(50000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	_, err := suite.msgServer.CreateMarket(goCtx, &types.MsgCreateMarket{
		Creator: addr1.String(),
		Base:    denomStake,
		Quote:   denomBze,
	})
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCreateMarket_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	//initial balances need to be 0
	initialUserBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(initialUserBalances.IsZero())

	balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(20000000000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))

	newMarket := types.Market{
		Creator: addr1.String(),
		Base:    denomStake,
		Quote:   denomBze,
	}
	_, err := suite.msgServer.CreateMarket(goCtx, &types.MsgCreateMarket{
		Creator: addr1.String(),
		Base:    denomStake,
		Quote:   denomBze,
	})

	suite.Require().Nil(err)
	storageMarket, ok := suite.k.GetMarket(suite.ctx, newMarket.Base, newMarket.Quote)
	suite.Require().True(ok)
	suite.Require().Equal(newMarket, storageMarket)

	params := suite.k.GetParams(suite.ctx)
	fee, err := sdk.ParseCoinNormalized(params.CreateMarketFee)
	suite.Require().Nil(err)
	userNewBal := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)

	suite.Require().Equal(userNewBal.AmountOf(fee.Denom), balances.AmountOf(fee.Denom).Sub(fee.Amount))
}
