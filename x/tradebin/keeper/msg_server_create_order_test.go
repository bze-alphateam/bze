package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/rand"
	"strconv"
	"time"
)

func (suite *IntegrationTestSuite) TestCreateOrder_InvalidAmount() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CreateOrder(goCtx, &types.MsgCreateOrder{
		Amount: "hdsihdshdshids",
		Price:  "1",
	})

	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "amount could not be converted to Int")
}

func (suite *IntegrationTestSuite) TestCreateOrder_AmountTooLow() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CreateOrder(goCtx, &types.MsgCreateOrder{
		Amount: "1",
		Price:  "1",
	})

	suite.Require().NotNil(err)
	//amount should be bigger than
	suite.Require().Contains(err.Error(), "amount should be bigger than")
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketNotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.CreateOrder(goCtx, &types.MsgCreateOrder{
		Amount:   "1000000",
		Price:    "1",
		MarketId: "notamarket/notatall",
	})

	suite.Require().NotNil(err)
	//market id
	suite.Require().Contains(err.Error(), "market id")
}

func (suite *IntegrationTestSuite) TestCreateOrder_InvalidOrderType() {
	suite.k.SetMarket(suite.ctx, market)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)
	_, err := suite.msgServer.CreateOrder(goCtx, &types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "1",
		MarketId:  getMarketId(),
		OrderType: "notatype",
		Creator:   addr1.String(),
	})

	suite.Require().NotNil(err)
	suite.Require().Contains(err.Error(), "order type")
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
	suite.Require().Contains(err.Error(), "bech32")
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketMaker_Buy_Success_ZeroDust() {
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

	_, ok := suite.k.GetUserDust(suite.ctx, addr1.String(), market.Quote)
	suite.Require().False(ok)

	_, ok = suite.k.GetUserDust(suite.ctx, addr1.String(), market.Base)
	suite.Require().False(ok)
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketTaker_Buy_Success_ZeroDust() {
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

	_, ok := suite.k.GetUserDust(suite.ctx, addr1.String(), market.Quote)
	suite.Require().False(ok)

	_, ok = suite.k.GetUserDust(suite.ctx, addr1.String(), market.Base)
	suite.Require().False(ok)
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketTaker_Buy_Success_WithDust() {
	suite.k.SetMarket(suite.ctx, market)
	suite.k.SetAggregatedOrder(suite.ctx, types.AggregatedOrder{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Amount:    "100000",
		Price:     "0.02331",
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
		Amount:    "87",
		Price:     "0.02331",
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

	suite.Require().Equal(userNewBal.AmountOf(fee.Denom), balances.AmountOf(fee.Denom).Sub(fee.Amount).SubRaw(3))

	feeDestinationModule := suite.app.AccountKeeper.GetModuleAddress(params.TakerFeeDestination)
	moduleBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, feeDestinationModule)
	suite.Require().False(moduleBalance.IsZero())
	suite.Require().Equal(moduleBalance.AmountOf(fee.Denom), fee.Amount)

	udQuote, ok := suite.k.GetUserDust(suite.ctx, addr1.String(), market.Quote)
	suite.Require().True(ok)
	suite.Require().Equal(udQuote.Denom, market.Quote)
	suite.Require().Equal(udQuote.Owner, addr1.String())
	suite.Require().EqualValues(udQuote.Amount, "0.972030000000000000")

	_, ok = suite.k.GetUserDust(suite.ctx, addr1.String(), market.Base)
	suite.Require().False(ok)
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

	//should never have dust on sell orders
	_, ok := suite.k.GetUserDust(suite.ctx, addr1.String(), market.Quote)
	suite.Require().False(ok)

	_, ok = suite.k.GetUserDust(suite.ctx, addr1.String(), market.Base)
	suite.Require().False(ok)
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

	//should never have dust on sell orders
	_, ok := suite.k.GetUserDust(suite.ctx, addr1.String(), market.Quote)
	suite.Require().False(ok)

	_, ok = suite.k.GetUserDust(suite.ctx, addr1.String(), market.Base)
	suite.Require().False(ok)
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketTaker_StressBalance() {
	suite.k.SetMarket(suite.ctx, market)
	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	//create initial random markets
	testDenom1 := "test1"
	testDenom2 := "test2"
	market1 := types.Market{
		Base:    testDenom1,
		Quote:   testDenom2,
		Creator: "addr1",
	}
	market2 := types.Market{
		Base:    denomStake,
		Quote:   testDenom2,
		Creator: "addr1",
	}
	market3 := types.Market{
		Base:    denomBze,
		Quote:   testDenom1,
		Creator: "addr1",
	}

	market4 := types.Market{
		Base:    denomBze,
		Quote:   testDenom2,
		Creator: "addr1",
	}
	marketsMap := make(map[string]types.Market)
	marketsMap[types.CreateMarketId(market.Base, market.Quote)] = market
	marketsMap[types.CreateMarketId(market1.Base, market1.Quote)] = market1
	marketsMap[types.CreateMarketId(market2.Base, market2.Quote)] = market2
	marketsMap[types.CreateMarketId(market3.Base, market3.Quote)] = market3
	marketsMap[types.CreateMarketId(market4.Base, market4.Quote)] = market4
	suite.k.SetMarket(suite.ctx, market1)
	suite.k.SetMarket(suite.ctx, market2)
	suite.k.SetMarket(suite.ctx, market3)
	suite.k.SetMarket(suite.ctx, market4)

	//set initial balances to 10 random accounts
	balances := sdk.NewCoins(
		newStakeCoin(999999999999999),
		newBzeCoin(999999999999999),
		sdk.NewInt64Coin(testDenom1, 999999999999999),
		sdk.NewInt64Coin(testDenom2, 999999999999999),
	)
	var creators []string
	for i := 0; i < 10; i++ {
		addr1 := sdk.AccAddress(fmt.Sprintf("addr%d_______________", i))
		acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
		suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)
		suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
		creators = append(creators, addr1.String())
	}

	for i := 0; i < 5; i++ {
		suite.randomOrderCreateMessages(suite.randomNumber(1000), creators, market)
		suite.randomOrderCreateMessages(suite.randomNumber(1000), creators, market1)
		suite.randomOrderCreateMessages(suite.randomNumber(1000), creators, market2)
		suite.randomOrderCreateMessages(suite.randomNumber(1000), creators, market3)
		suite.randomOrderCreateMessages(suite.randomNumber(1000), creators, market4)

		engine.ProcessQueueMessages(suite.ctx)
	}

	allOrders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(allOrders)
	amounts := sdk.NewCoins(sdk.NewCoin(market.Base, sdk.ZeroInt()), sdk.NewCoin(market.Quote, sdk.ZeroInt()))
	//check module balance is equal to the coins found
	foundOrderTypesByPrice := make(map[string]string)
	aggregatedOrders := make(map[string]types.AggregatedOrder)
	for _, or := range allOrders {
		//search to see if an opposite order with the same price already exists
		//if so it means the engine failed to match them when they were processed
		oppositeKey := fmt.Sprintf("%s_%s_%s", or.MarketId, or.Price, types.TheOtherOrderType(or.OrderType))
		id, ok := foundOrderTypesByPrice[oppositeKey]
		suite.Require().False(ok, fmt.Sprintf("order [%s] has the same price [%s] as [%s]", id, or.Price, or.Id))

		//save locally to check later
		key := fmt.Sprintf("%s_%s_%s", or.MarketId, or.Price, or.OrderType)
		foundOrderTypesByPrice[key] = or.Id
		amtInt, ok := sdk.NewIntFromString(or.Amount)
		suite.Require().True(ok)
		pickMarket, ok := marketsMap[or.MarketId]
		suite.Require().True(ok)
		coins, _, err := suite.k.GetOrderSdkCoin(or.OrderType, or.Price, amtInt, &pickMarket)
		suite.Require().NoError(err)

		amounts = amounts.Add(coins)
		//save in aggregated orders map so we can check it later
		agg, found := aggregatedOrders[key]
		if !found {
			agg = types.AggregatedOrder{
				MarketId:  or.MarketId,
				OrderType: or.OrderType,
				Amount:    "0",
				Price:     or.Price,
			}
		}
		aggAmt, ok := sdk.NewIntFromString(agg.Amount)
		suite.Require().True(ok)
		aggAmt = aggAmt.Add(amtInt)
		agg.Amount = aggAmt.String()
		aggregatedOrders[key] = agg
	}

	for _, agg := range aggregatedOrders {
		foundAgg, ok := suite.k.GetAggregatedOrder(suite.ctx, agg.MarketId, agg.OrderType, agg.Price)
		suite.Require().True(ok)
		suite.Require().Equal(foundAgg.Amount, agg.Amount)
	}

	moduleAddr := suite.app.AccountKeeper.GetModuleAddress(types.ModuleName)
	moduleBalance := suite.app.BankKeeper.GetAllBalances(suite.ctx, moduleAddr)
	suite.Require().Equal(moduleBalance, amounts)
}

func (suite *IntegrationTestSuite) TestCreateOrder_MarketTaker_CheckPrice_Fail() {
	suite.k.SetMarket(suite.ctx, market)
	suite.k.SetAggregatedOrder(suite.ctx, types.AggregatedOrder{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "10000",
		Price:     "1",
	})
	suite.k.SetAggregatedOrder(suite.ctx, types.AggregatedOrder{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Amount:    "10000",
		Price:     "5",
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

	//check price error on sell order
	orderMsg := types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "0.5",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err := suite.msgServer.CreateOrder(goCtx, &orderMsg)

	suite.Require().Error(err)
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 0)

	//check price error on buy
	orderMsg = types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "5.1",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err = suite.msgServer.CreateOrder(goCtx, &orderMsg)
	suite.Require().Error(err)

	qmList = suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 0)

	//add 2 new orders in order to check message queue price validator
	orderMsg = types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "4",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err = suite.msgServer.CreateOrder(goCtx, &orderMsg)
	suite.Require().NoError(err)

	orderMsg = types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "4.5",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err = suite.msgServer.CreateOrder(goCtx, &orderMsg)
	suite.Require().NoError(err)

	orderMsg = types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "3.5",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Creator:   addr1.String(),
	}
	_, err = suite.msgServer.CreateOrder(goCtx, &orderMsg)
	suite.Require().Error(err)

	orderMsg = types.MsgCreateOrder{
		Amount:    "1000000",
		Price:     "4.55",
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Creator:   addr1.String(),
	}
	_, err = suite.msgServer.CreateOrder(goCtx, &orderMsg)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) randomOrderCreateMessages(count int, creators []string, market types.Market) []types.MsgCreateOrder {
	var msgs []types.MsgCreateOrder
	orderTypes := []string{types.OrderTypeBuy, types.OrderTypeSell}
	goCtx := sdk.WrapSDKContext(suite.ctx)
	const ExecPrice = 15
	for i := 0; i < count; i++ {
		randomOrderType := orderTypes[i%2]
		randomPrice := suite.randomNumber(24) + 1 //make sure it's always higher than 0
		if randomOrderType == types.OrderTypeBuy && randomPrice > ExecPrice {
			randomPrice = ExecPrice
		} else if randomOrderType == types.OrderTypeSell && randomPrice < ExecPrice {
			randomPrice = ExecPrice
		}
		randomPriceStr := strconv.Itoa(randomPrice)
		minAmount := keeper.CalculateMinAmount(randomPriceStr)
		orderMsg := types.MsgCreateOrder{
			Amount:    minAmount.AddRaw(int64(suite.randomNumber(1000))).String(),
			Price:     randomPriceStr,
			MarketId:  types.CreateMarketId(market.Base, market.Quote),
			OrderType: randomOrderType,
			Creator:   creators[suite.randomNumber(len(creators))],
		}

		_, err := suite.msgServer.CreateOrder(goCtx, &orderMsg)
		suite.Require().NoError(err)
	}

	return msgs
}

func (suite *IntegrationTestSuite) randomNumber(to int) int {
	// Seed the random number generator to get different results each run
	// Uses the current time as the seed
	rand.Seed(time.Now().UnixNano())

	// Generate a random number between 0 and to
	num := rand.Intn(to)

	return num
}

func (suite *IntegrationTestSuite) msgOrderFillSetup(orderType string) (allPrices []string, addr1, addr2 sdk.AccAddress) {
	suite.k.SetMarket(suite.ctx, market)

	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 = sdk.AccAddress("addr1_______________")
	acc1 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc1)

	addr2 = sdk.AccAddress("addr2_______________")
	acc2 := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr2)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc2)

	//initial balances need to be 0
	user1Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr1)
	suite.Require().True(user1Balances.IsZero())

	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	suite.Require().True(user2Balances.IsZero())

	balances := sdk.NewCoins(newStakeCoin(10000000), newBzeCoin(20000000000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr2, balances))

	initialPrice := sdk.ZeroInt()
	staticAmount := "1000000"
	for i := 1; i <= 10; i++ {
		initialPrice = initialPrice.AddRaw(int64(i))
		orderMsg := types.MsgCreateOrder{
			Amount:    staticAmount,
			Price:     initialPrice.String(),
			MarketId:  getMarketId(),
			OrderType: orderType,
			Creator:   addr1.String(),
		}

		_, err := suite.msgServer.CreateOrder(goCtx, &orderMsg)
		suite.Require().NoError(err)

		allPrices = append(allPrices, initialPrice.String())
	}

	return
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_OneOrderPartialFill_Sell() {
	_, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "500000",
				Price:  "1",
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 1)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeBuy)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillSell)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee
	suite.Require().EqualValues(20000000000-500000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000+500000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 10)
	remainingOrderChecked := false
	for _, order := range all {
		//let's find our order and check if it contains the correct amount
		if order.Price != "1" {
			continue
		}

		suite.Require().EqualValues(order.Amount, fmt.Sprintf("%d", 1000000-500000))
		remainingOrderChecked = true
		break
	}

	suite.Require().True(remainingOrderChecked)
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_OneOrderPartialFill_Buy() {
	_, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "500000",
				Price:  "1",
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 1)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeSell)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillBuy)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee
	suite.Require().EqualValues(10000000-500000, user2Balances.AmountOf(denomStake).Int64())
	suite.Require().EqualValues(20000000000-100000+500000, user2Balances.AmountOf(denomBze).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 10)
	remainingOrderChecked := false
	for _, order := range all {
		//let's find our order and check if it contains the correct amount
		if order.Price != "1" {
			continue
		}

		suite.Require().EqualValues(order.Amount, fmt.Sprintf("%d", 1000000-500000))
		remainingOrderChecked = true
		break
	}

	suite.Require().True(remainingOrderChecked)
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_OneOrderFullFill_Sell() {
	_, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  "1",
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 1)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeBuy)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillSell)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee
	suite.Require().EqualValues(20000000000-1000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000+1000000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	for _, order := range all {
		//let's find our order and check if it contains the correct amount
		suite.Require().NotEqual(order.Price, "1")
	}
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_OneOrderFullFill_Buy() {
	_, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  "1",
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 1)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeSell)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillBuy)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee
	suite.Require().EqualValues(20000000000+1000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000-1000000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	for _, order := range all {
		//let's find our order and check if it contains the correct amount
		suite.Require().NotEqual(order.Price, "1000000")
	}
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_TwoOrdersOnePartialFill_Sell() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  allPrices[0],
			},
			{
				Amount: "500000",
				Price:  allPrices[1],
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	agg := suite.k.GetAllAggregatedOrder(suite.ctx)
	_ = agg
	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 2)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeBuy)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillSell)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)
	suite.Require().Equal(qmList[1].MarketId, getMarketId())
	suite.Require().Equal(qmList[1].Amount, fillOrder.Orders[1].Amount)
	suite.Require().Equal(qmList[1].OrderType, types.OrderTypeBuy)
	suite.Require().Equal(qmList[1].MessageType, types.MessageTypeFillSell)
	suite.Require().Equal(qmList[1].Price, fillOrder.Orders[1].Price)
	suite.Require().Equal(qmList[1].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-2500000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000+1500000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	remainingOrderChecked := false
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		//let's find our order and check if it contains the correct amount
		if order.Price != allPrices[1] {
			continue
		}

		suite.Require().EqualValues(order.Amount, fmt.Sprintf("%d", 1000000-500000))
		remainingOrderChecked = true
		break
	}

	suite.Require().True(remainingOrderChecked)
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_TwoOrdersOnePartialFill_Buy() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  allPrices[0],
			},
			{
				Amount: "500000",
				Price:  allPrices[1],
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	agg := suite.k.GetAllAggregatedOrder(suite.ctx)
	_ = agg
	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 2)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeSell)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillBuy)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)
	suite.Require().Equal(qmList[1].MarketId, getMarketId())
	suite.Require().Equal(qmList[1].Amount, fillOrder.Orders[1].Amount)
	suite.Require().Equal(qmList[1].OrderType, types.OrderTypeSell)
	suite.Require().Equal(qmList[1].MessageType, types.MessageTypeFillBuy)
	suite.Require().Equal(qmList[1].Price, fillOrder.Orders[1].Price)
	suite.Require().Equal(qmList[1].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000+2500000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000-1500000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	remainingOrderChecked := false
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		//let's find our order and check if it contains the correct amount
		if order.Price != allPrices[1] {
			continue
		}

		suite.Require().EqualValues(order.Amount, fmt.Sprintf("%d", 1000000-500000))
		remainingOrderChecked = true
		break
	}

	suite.Require().True(remainingOrderChecked)
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_TwoFullyFilledOrders_Buy() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  allPrices[0],
			},
			{
				Amount: "1000000",
				Price:  allPrices[1],
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	agg := suite.k.GetAllAggregatedOrder(suite.ctx)
	_ = agg
	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 2)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeSell)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillBuy)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)
	suite.Require().Equal(qmList[1].MarketId, getMarketId())
	suite.Require().Equal(qmList[1].Amount, fillOrder.Orders[1].Amount)
	suite.Require().Equal(qmList[1].OrderType, types.OrderTypeSell)
	suite.Require().Equal(qmList[1].MessageType, types.MessageTypeFillBuy)
	suite.Require().Equal(qmList[1].Price, fillOrder.Orders[1].Price)
	suite.Require().Equal(qmList[1].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000+4000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000-2000000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 8)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		suite.Require().NotEqual(order.Price, allPrices[1])
	}
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_TwoFullyFilledOrders_Sell() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1000000",
				Price:  allPrices[0],
			},
			{
				Amount: "1000000",
				Price:  allPrices[1],
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	agg := suite.k.GetAllAggregatedOrder(suite.ctx)
	_ = agg
	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 2)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeBuy)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillSell)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)
	suite.Require().Equal(qmList[1].MarketId, getMarketId())
	suite.Require().Equal(qmList[1].Amount, fillOrder.Orders[1].Amount)
	suite.Require().Equal(qmList[1].OrderType, types.OrderTypeBuy)
	suite.Require().Equal(qmList[1].MessageType, types.MessageTypeFillSell)
	suite.Require().Equal(qmList[1].Price, fillOrder.Orders[1].Price)
	suite.Require().Equal(qmList[1].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-4000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000+2000000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 8)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		suite.Require().NotEqual(order.Price, allPrices[1])
	}
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_OneFullyFilledOrderWithExtraAmount_Buy() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1500000",
				Price:  allPrices[0],
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	//check user funds were fully deducted(after message processing the remaining will be refunded)
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000-1500000, user2Balances.AmountOf(denomStake).Int64())

	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 1)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeSell)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillBuy)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances = suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000+1000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000-1000000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
	}
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_OneFullyFilledOrderWithExtraAmount_Sell() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "1500000",
				Price:  allPrices[0],
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	//check user funds were fully deducted(after message processing the remaining will be refunded)
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-100000-1500000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000, user2Balances.AmountOf(denomStake).Int64())

	agg := suite.k.GetAllAggregatedOrder(suite.ctx)
	_ = agg
	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 1)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeBuy)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillSell)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances = suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-1000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000+1000000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 9)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
	}
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_TwoFullyFilledOrdersWithExtraAmounts_Buy() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders: []*types.FillOrderItem{
			{
				Amount: "2000000",
				Price:  allPrices[0],
			},
			{
				Amount: "1500000",
				Price:  allPrices[1],
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	//check user funds were fully deducted(after message processing the remaining will be refunded)
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000-3500000, user2Balances.AmountOf(denomStake).Int64())

	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 2)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeSell)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillBuy)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)
	suite.Require().Equal(qmList[1].MarketId, getMarketId())
	suite.Require().Equal(qmList[1].Amount, fillOrder.Orders[1].Amount)
	suite.Require().Equal(qmList[1].OrderType, types.OrderTypeSell)
	suite.Require().Equal(qmList[1].MessageType, types.MessageTypeFillBuy)
	suite.Require().Equal(qmList[1].Price, fillOrder.Orders[1].Price)
	suite.Require().Equal(qmList[1].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances = suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000+4000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000-2000000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 8)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		suite.Require().NotEqual(order.Price, allPrices[1])
		suite.Require().Equal(order.Amount, "1000000")
	}
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_TwoFullyFilledOrdersWithExtraAmounts_Sell() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	//Let's fill 1 order with amount lower than the available order
	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders: []*types.FillOrderItem{
			{
				Amount: "2000000",
				Price:  allPrices[0],
			},
			{
				Amount: "1500000",
				Price:  allPrices[1],
			},
		},
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	//check user funds were fully deducted(after message processing the remaining will be refunded)
	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-6500000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000, user2Balances.AmountOf(denomStake).Int64())

	//check that the new message is saved accordingly to queue messages storage
	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, 2)
	suite.Require().Equal(qmList[0].MarketId, getMarketId())
	suite.Require().Equal(qmList[0].Amount, fillOrder.Orders[0].Amount)
	suite.Require().Equal(qmList[0].OrderType, types.OrderTypeBuy)
	suite.Require().Equal(qmList[0].MessageType, types.MessageTypeFillSell)
	suite.Require().Equal(qmList[0].Price, fillOrder.Orders[0].Price)
	suite.Require().Equal(qmList[0].Owner, fillOrder.Creator)
	suite.Require().Equal(qmList[1].MarketId, getMarketId())
	suite.Require().Equal(qmList[1].Amount, fillOrder.Orders[1].Amount)
	suite.Require().Equal(qmList[1].OrderType, types.OrderTypeBuy)
	suite.Require().Equal(qmList[1].MessageType, types.MessageTypeFillSell)
	suite.Require().Equal(qmList[1].Price, fillOrder.Orders[1].Price)
	suite.Require().Equal(qmList[1].Owner, fillOrder.Creator)

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances = suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-4000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000+2000000, user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().NotEmpty(all)
	//the same number of orders should be present (we filled only half of first order)
	suite.Require().Len(all, 8)
	for _, order := range all {
		suite.Require().NotEqual(order.Price, allPrices[0])
		suite.Require().NotEqual(order.Price, allPrices[1])
		suite.Require().Equal(order.Amount, "1000000")
	}
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_FillAllWithExtraAmounts_Sell() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
	suite.Require().NotEmpty(allPrices)
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders:    []*types.FillOrderItem{},
	}

	quoteAmount := sdk.ZeroInt()
	baseAmount := sdk.ZeroInt()
	expectedQuoteAmount := sdk.ZeroInt()
	expectedBaseAmount := sdk.ZeroInt()
	for _, pr := range allPrices {
		randAmount := 1000000 + suite.randomNumber(999999)
		fillOrder.Orders = append(fillOrder.Orders, &types.FillOrderItem{
			Price:  pr,
			Amount: fmt.Sprintf("%d", randAmount),
		})

		price, ok := sdk.NewIntFromString(pr)
		suite.Require().True(ok)
		baseAmount = baseAmount.AddRaw(int64(randAmount))
		quoteAmount = quoteAmount.Add(price.MulRaw(int64(randAmount)))

		expectedQuoteAmount = expectedQuoteAmount.Add(price.MulRaw(1000000))
		expectedBaseAmount = expectedBaseAmount.AddRaw(1000000)
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-100000-quoteAmount.Int64(), user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000, user2Balances.AmountOf(denomStake).Int64())

	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, len(allPrices))

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances = suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-100000-expectedQuoteAmount.Int64(), user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(10000000+expectedBaseAmount.Int64(), user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Empty(all)
}

func (suite *IntegrationTestSuite) TestMsgOrderFill_FillAllWithExtraAmounts_Buy() {
	allPrices, _, addr2 := suite.msgOrderFillSetup(types.OrderTypeBuy)
	suite.Require().NotEmpty(allPrices)

	//add an extra 10,000,000 STAKE to addr2 because it will fill buy orders (it sells STAKE)
	balances := sdk.NewCoins(newStakeCoin(10000000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr2, balances))
	goCtx := sdk.WrapSDKContext(suite.ctx)

	engine, err := keeper.NewProcessingEngine(suite.app.TradebinKeeper, suite.app.BankKeeper, suite.app.TradebinKeeper.Logger(suite.ctx))
	suite.Require().Nil(err)

	engine.ProcessQueueMessages(suite.ctx)

	fillOrder := types.MsgFillOrders{
		Creator:   addr2.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Orders:    []*types.FillOrderItem{},
	}

	quoteAmount := sdk.ZeroInt()
	baseAmount := sdk.ZeroInt()
	expectedQuoteAmount := sdk.ZeroInt()
	expectedBaseAmount := sdk.ZeroInt()
	for _, pr := range allPrices {
		randAmount := 1000000 + suite.randomNumber(999999)
		fillOrder.Orders = append(fillOrder.Orders, &types.FillOrderItem{
			Price:  pr,
			Amount: fmt.Sprintf("%d", randAmount),
		})

		price, ok := sdk.NewIntFromString(pr)
		suite.Require().True(ok)
		baseAmount = baseAmount.AddRaw(int64(randAmount))
		quoteAmount = quoteAmount.Add(price.MulRaw(int64(randAmount)))

		expectedQuoteAmount = expectedQuoteAmount.Add(price.MulRaw(1000000))
		expectedBaseAmount = expectedBaseAmount.AddRaw(1000000)
	}

	_, err = suite.msgServer.FillOrders(goCtx, &fillOrder)
	suite.Require().NoError(err)

	user2Balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-100000, user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(20000000-baseAmount.Int64(), user2Balances.AmountOf(denomStake).Int64())

	qmList := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qmList, len(allPrices))

	engine.ProcessQueueMessages(suite.ctx)

	//check that the order was correctly executed and the user received/spent the amounts related to the order
	user2Balances = suite.app.BankKeeper.GetAllBalances(suite.ctx, addr2)
	//subtract the market making trading fee + the coins spent on the order
	suite.Require().EqualValues(20000000000-100000+expectedQuoteAmount.Int64(), user2Balances.AmountOf(denomBze).Int64())
	suite.Require().EqualValues(20000000-expectedBaseAmount.Int64(), user2Balances.AmountOf(denomStake).Int64())

	all := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Empty(all)
}
