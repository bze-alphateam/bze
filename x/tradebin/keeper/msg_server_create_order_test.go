package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestCreateOrder_InvalidAmount() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CreateOrder(goCtx, &types.MsgCreateOrder{
		Amount: "as2",
	})

	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCreateOrder_AmountTooLow() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CreateOrder(goCtx, &types.MsgCreateOrder{
		Amount: "10",
		Price:  "1",
	})

	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketNotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CreateOrder(goCtx, &types.MsgCreateOrder{
		Amount:   "1000000",
		Price:    "1",
		MarketId: "notamarket/notatall",
	})

	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCreateOrder_InvalidOrderType() {
	suite.k.SetMarket(suite.ctx, market)
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CreateOrder(goCtx, &types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: "notatype",
	})

	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCreateOrder_InvalidCreator() {
	suite.k.SetMarket(suite.ctx, market)
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CreateOrder(goCtx, &types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   "notanaddress",
	})

	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketMaker_Buy_Success() {
	suite.k.SetMarket(suite.ctx, market)
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	//initial balances need to be 0
	initialUserBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(initialUserBalances.IsZero())

	balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(20000000000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "2",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(goCtx, &orderMsg)

	suite.Require().Nil(err)
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 1)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, orderMsg.Amount)
	suite.Require().Equal(qmList[0].OrderType, orderMsg.OrderType)
	suite.Require().Equal(qmList[0].MessageType, orderMsg.OrderType)
	suite.Require().Equal(qmList[0].Price, orderMsg.Price)
	suite.Require().Equal(qmList[0].Owner, orderMsg.Creator)

	params := suite.k.GetParams(suite.ctx)
	fee, err := sdk.ParseCoinNormalized(params.MarketMakerFee)
	suite.Require().Nil(err)
	userNewBal := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)

	suite.Require().Equal(userNewBal.AmountOf(fee.Denom), balances.AmountOf(fee.Denom).Sub(fee.Amount).SubRaw(2000000))

	feeDestinationModule := suite.app.AccountKeeper.GetModuleAddress(params.MakerFeeDestination)
	moduleBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, feeDestinationModule)
	suite.Require().False(moduleBalance.IsZero())
	suite.Require().Equal(moduleBalance.AmountOf(fee.Denom), fee.Amount)
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketTaker_Buy_Success() {
	suite.k.SetMarket(suite.ctx, market)
	suite.k.SetAggregatedOrder(suite.ctx, types.AggregatedOrder{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Amount:    "10000",
		Price:     "2",
	})

	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	//initial balances need to be 0
	initialUserBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(initialUserBalances.IsZero())

	balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(20000000000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "2",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(goCtx, &orderMsg)

	suite.Require().Nil(err)
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 1)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, orderMsg.Amount)
	suite.Require().Equal(qmList[0].OrderType, orderMsg.OrderType)
	suite.Require().Equal(qmList[0].MessageType, orderMsg.OrderType)
	suite.Require().Equal(qmList[0].Price, orderMsg.Price)
	suite.Require().Equal(qmList[0].Owner, orderMsg.Creator)

	params := suite.k.GetParams(suite.ctx)
	fee, err := sdk.ParseCoinNormalized(params.MarketTakerFee)
	suite.Require().Nil(err)
	userNewBal := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)

	suite.Require().Equal(userNewBal.AmountOf(fee.Denom), balances.AmountOf(fee.Denom).Sub(fee.Amount).SubRaw(2000000))

	feeDestinationModule := suite.app.AccountKeeper.GetModuleAddress(params.TakerFeeDestination)
	moduleBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, feeDestinationModule)
	suite.Require().False(moduleBalance.IsZero())
	suite.Require().Equal(moduleBalance.AmountOf(fee.Denom), fee.Amount)
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketMaker_Sell_Success() {
	suite.k.SetMarket(suite.ctx, market)
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	//initial balances need to be 0
	initialUserBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(initialUserBalances.IsZero())

	balances := sdk.NewCoins(newStakeCoin(10000000), newBzeCoin(20000000000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(goCtx, &orderMsg)

	suite.Require().Nil(err)
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 1)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, orderMsg.Amount)
	suite.Require().Equal(qmList[0].OrderType, orderMsg.OrderType)
	suite.Require().Equal(qmList[0].MessageType, orderMsg.OrderType)
	suite.Require().Equal(qmList[0].Price, orderMsg.Price)
	suite.Require().Equal(qmList[0].Owner, orderMsg.Creator)

	params := suite.k.GetParams(suite.ctx)
	fee, err := sdk.ParseCoinNormalized(params.MarketMakerFee)
	suite.Require().Nil(err)
	userNewBal := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)

	suite.Require().Equal(userNewBal.AmountOf(fee.Denom), balances.AmountOf(fee.Denom).Sub(fee.Amount))
	suite.Require().Equal(userNewBal.AmountOf(market.Base), balances.AmountOf(market.Base).SubRaw(1000000))

	feeDestinationModule := suite.app.AccountKeeper.GetModuleAddress(params.MakerFeeDestination)
	moduleBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, feeDestinationModule)
	suite.Require().False(moduleBalance.IsZero())
	suite.Require().Equal(moduleBalance.AmountOf(fee.Denom), fee.Amount)
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketTaker_Sell_Success() {
	suite.k.SetMarket(suite.ctx, market)
	suite.k.SetAggregatedOrder(suite.ctx, types.AggregatedOrder{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "10000",
		Price:     "1",
	})

	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	//initial balances need to be 0
	initialUserBalances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(initialUserBalances.IsZero())

	balances := sdk.NewCoins(newStakeCoin(10000000), newBzeCoin(20000000000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(goCtx, &orderMsg)

	suite.Require().Nil(err)
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 1)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, orderMsg.Amount)
	suite.Require().Equal(qmList[0].OrderType, orderMsg.OrderType)
	suite.Require().Equal(qmList[0].MessageType, orderMsg.OrderType)
	suite.Require().Equal(qmList[0].Price, orderMsg.Price)
	suite.Require().Equal(qmList[0].Owner, orderMsg.Creator)

	params := suite.k.GetParams(suite.ctx)
	fee, err := sdk.ParseCoinNormalized(params.MarketTakerFee)
	suite.Require().Nil(err)
	userNewBal := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)

	suite.Require().Equal(userNewBal.AmountOf(fee.Denom), balances.AmountOf(fee.Denom).Sub(fee.Amount))
	suite.Require().Equal(userNewBal.AmountOf(market.Base), balances.AmountOf(market.Base).SubRaw(1000000))

	//TODO: better testing in case fee destination is changed to community pool
	feeDestinationModule := suite.app.AccountKeeper.GetModuleAddress(params.TakerFeeDestination)
	moduleBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, feeDestinationModule)
	suite.Require().False(moduleBalance.IsZero())
	suite.Require().Equal(moduleBalance.AmountOf(fee.Denom), fee.Amount)
}
