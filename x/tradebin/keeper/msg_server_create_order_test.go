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
		suite.randomOrderCreateMessages(suite.randomNumber(3000), creators, market)
		suite.randomOrderCreateMessages(suite.randomNumber(3000), creators, market1)
		suite.randomOrderCreateMessages(suite.randomNumber(3000), creators, market2)
		suite.randomOrderCreateMessages(suite.randomNumber(3000), creators, market3)
		suite.randomOrderCreateMessages(suite.randomNumber(3000), creators, market4)

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
		coins, err := suite.k.GetOrderCoins(or.OrderType, or.Price, amtInt, &pickMarket)
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

func (suite *IntegrationTestSuite) randomOrderCreateMessages(count int, creators []string, market types.Market) []types.MsgCreateOrder {
	var msgs []types.MsgCreateOrder
	orderTypes := []string{types.OrderTypeBuy, types.OrderTypeSell}
	goCtx := sdk.WrapSDKContext(suite.ctx)
	for i := 0; i < count; i++ {
		randomPrice := suite.randomNumber(4) + 1 //make sure it's always higher than 0
		randomPriceStr := strconv.Itoa(randomPrice)
		minAmount := keeper.CalculateMinAmount(randomPriceStr)
		randomOrderType := orderTypes[i%2]
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

	// Generate a random number between 0 and 99
	num := rand.Intn(to)

	return num
}
