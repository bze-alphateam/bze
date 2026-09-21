package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

// Every rejection test below runs with NO bank expectations: gomock panics on an unexpected call, so
// a message that captured a fee or escrowed funds before being refused would fail the test.

// haltedPool is the stake/uhalt pool (alphabetical id "stake_uhalt").
func haltedPool() types.LiquidityPool {
	return types.LiquidityPool{
		Id:      "stake_" + denomHalted,
		Base:    denomStake,
		Quote:   denomHalted,
		LpDenom: "ulp_stake_" + denomHalted,
		Creator: getTestAddress(),
		Fee:     math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
		ReserveBase:  math.NewInt(1_000_000_000_000),
		ReserveQuote: math.NewInt(2_000_000_000_000),
	}
}

// nativePool is the ubze/stake pool (id "stake_ubze"), unrelated to the halted denom.
func nativePool() types.LiquidityPool {
	return types.LiquidityPool{
		Id:      "stake_ubze",
		Base:    denomStake,
		Quote:   denomBze,
		LpDenom: "ulp_stake_ubze",
		Creator: getTestAddress(),
		Fee:     math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(5, 1),
			Burner:    math.LegacyNewDecWithPrec(3, 1),
			Providers: math.LegacyNewDecWithPrec(2, 1),
		},
		ReserveBase:  math.NewInt(3_000_000_000_000),
		ReserveQuote: math.NewInt(4_000_000_000_000),
	}
}

func (suite *IntegrationTestSuite) requirePoolUnchanged(pool types.LiquidityPool) {
	stored, found := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
	suite.Require().True(found)
	suite.Require().Equal(pool.ReserveBase, stored.ReserveBase)
	suite.Require().Equal(pool.ReserveQuote, stored.ReserveQuote)
}

func (suite *IntegrationTestSuite) TestMsgHalted_CreateMarket_Rejected() {
	suite.setHaltedDenoms(denomHalted)

	for _, m := range []types.Market{haltedBaseMarket(), haltedQuoteMarket()} {
		suite.T().Run(marketIdOf(m), func(t *testing.T) {
			_, err := suite.msgServer.CreateMarket(suite.ctx, &types.MsgCreateMarket{
				Creator: getTestAddress(),
				Base:    m.Base,
				Quote:   m.Quote,
			})
			suite.Require().ErrorIs(err, types.ErrDenomHalted)

			_, found := suite.k.GetMarket(suite.ctx, m.Base, m.Quote)
			suite.Require().False(found)
		})
	}
}

func (suite *IntegrationTestSuite) TestMsgHalted_CreateLiquidityPool_Rejected() {
	suite.setHaltedDenoms(denomHalted)

	for _, pair := range [][2]string{{denomHalted, denomBze}, {denomStake, denomHalted}} {
		suite.T().Run(pair[0]+"_"+pair[1], func(t *testing.T) {
			_, err := suite.msgServer.CreateLiquidityPool(suite.ctx, &types.MsgCreateLiquidityPool{
				Creator:      getTestAddress(),
				Base:         pair[0],
				Quote:        pair[1],
				Fee:          "0.003",
				FeeDest:      getFeeDestinationString("0.3", "0.3", "0.4"),
				InitialBase:  math.NewInt(1_000_000),
				InitialQuote: math.NewInt(1_000_000),
			})
			suite.Require().ErrorIs(err, types.ErrDenomHalted)
			suite.Require().Empty(suite.k.GetAllLiquidityPool(suite.ctx))
		})
	}
}

func (suite *IntegrationTestSuite) TestMsgHalted_CreateOrder_Rejected() {
	suite.k.SetMarket(suite.ctx, haltedBaseMarket())
	suite.k.SetMarket(suite.ctx, haltedQuoteMarket())
	suite.setHaltedDenoms(denomHalted)

	for _, m := range []types.Market{haltedBaseMarket(), haltedQuoteMarket()} {
		for _, orderType := range []string{types.OrderTypeBuy, types.OrderTypeSell} {
			suite.T().Run(marketIdOf(m)+"_"+orderType, func(t *testing.T) {
				_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
					Creator:   getTestAddress(),
					MarketId:  marketIdOf(m),
					OrderType: orderType,
					Amount:    "1000000",
					Price:     "2",
				})
				suite.Require().ErrorIs(err, types.ErrDenomHalted)
				suite.Require().Empty(suite.k.GetAllQueueMessage(suite.ctx))
			})
		}
	}
}

func (suite *IntegrationTestSuite) TestMsgHalted_FillOrders_Rejected() {
	m := haltedQuoteMarket()
	suite.k.SetMarket(suite.ctx, m)
	suite.setHaltedDenoms(denomHalted)

	for _, orderType := range []string{types.OrderTypeBuy, types.OrderTypeSell} {
		suite.T().Run(orderType, func(t *testing.T) {
			_, err := suite.msgServer.FillOrders(suite.ctx, &types.MsgFillOrders{
				Creator:   getTestAddress(),
				MarketId:  marketIdOf(m),
				OrderType: orderType,
				Orders:    []*types.FillOrderItem{{Price: "2", Amount: "1000000"}},
			})
			suite.Require().ErrorIs(err, types.ErrDenomHalted)
			suite.Require().Empty(suite.k.GetAllQueueMessage(suite.ctx))
		})
	}
}

func (suite *IntegrationTestSuite) TestMsgHalted_AddLiquidity_Rejected() {
	pool := haltedPool()
	suite.k.SetLiquidityPool(suite.ctx, pool)
	suite.setHaltedDenoms(denomHalted)

	_, err := suite.msgServer.AddLiquidity(suite.ctx, &types.MsgAddLiquidity{
		Creator:     getTestAddress(),
		PoolId:      pool.Id,
		BaseAmount:  math.NewInt(1_000_000),
		QuoteAmount: math.NewInt(2_000_000),
		MinLpTokens: math.NewInt(1),
	})
	suite.Require().ErrorIs(err, types.ErrDenomHalted)
	suite.requirePoolUnchanged(pool)
}

// A halted pool is refused wherever it sits in the route: as the first hop and as the second hop.
func (suite *IntegrationTestSuite) TestMsgHalted_MultiSwap_Rejected() {
	halted := haltedPool()
	native := nativePool()
	suite.k.SetLiquidityPool(suite.ctx, halted)
	suite.k.SetLiquidityPool(suite.ctx, native)
	suite.setHaltedDenoms(denomHalted)

	cases := []struct {
		name   string
		routes []string
		input  sdk.Coin
		minOut sdk.Coin
	}{
		{"halted pool only", []string{halted.Id}, sdk.NewInt64Coin(denomStake, 1_000_000), sdk.NewInt64Coin(denomHalted, 1)},
		{"halted pool as first hop", []string{halted.Id, native.Id}, sdk.NewInt64Coin(denomHalted, 1_000_000), sdk.NewInt64Coin(denomBze, 1)},
		{"halted pool as second hop", []string{native.Id, halted.Id}, sdk.NewInt64Coin(denomBze, 1_000_000), sdk.NewInt64Coin(denomHalted, 1)},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			_, err := suite.msgServer.MultiSwap(suite.ctx, &types.MsgMultiSwap{
				Creator:   getTestAddress(),
				Routes:    c.routes,
				Input:     c.input,
				MinOutput: c.minOut,
			})
			suite.Require().ErrorIs(err, types.ErrDenomHalted)
			suite.Require().NotErrorIs(err, types.ErrInvalidRoutes, "the halt must be reported as such, not as an invalid route")
			suite.requirePoolUnchanged(halted)
			suite.requirePoolUnchanged(native)
		})
	}
}

// With a denom halted, the unrelated market and pool keep working exactly as before.
func (suite *IntegrationTestSuite) TestMsgHalted_UnrelatedMarketAndPool_Unaffected() {
	suite.k.SetMarket(suite.ctx, market) // stake/ubze
	native := nativePool()
	suite.k.SetLiquidityPool(suite.ctx, native)
	suite.setHaltedDenoms(denomHalted)

	addr1 := sdk.AccAddress("addr1_______________")
	params := suite.k.GetParams(suite.ctx)

	// create a market on unrelated denoms (stake/ubze already exists, so use a fresh pair)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), "uatom").Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), denomBze).Return(true).Times(1)
	marketFee := sdk.NewCoins(params.CreateMarketFee)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, marketFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), marketFee).Return(nil).Times(1)
	_, err := suite.msgServer.CreateMarket(suite.ctx, &types.MsgCreateMarket{Creator: addr1.String(), Base: "uatom", Quote: denomBze})
	suite.Require().NoError(err)
	_, found := suite.k.GetMarket(suite.ctx, "uatom", denomBze)
	suite.Require().True(found)

	// maker order on the unrelated market
	makerFee := sdk.NewCoins(params.MarketMakerFee)
	paidCoins := sdk.NewCoins(sdk.NewCoin(denomBze, math.NewInt(2_000_000)))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), makerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, paidCoins).Return(nil).Times(1)
	_, err = suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Creator: addr1.String(), MarketId: getMarketId(), OrderType: types.OrderTypeBuy, Amount: "1000000", Price: "2",
	})
	suite.Require().NoError(err)
	suite.Require().Len(suite.k.GetAllQueueMessage(suite.ctx), 1)

	// swap on the unrelated pool
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, gomock.Any()).Return(nil).AnyTimes()
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, gomock.Any()).Return(nil).Times(1)
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).Return(authtypes.NewEmptyModuleAccount(types.ModuleName)).AnyTimes()
	resp, err := suite.msgServer.MultiSwap(suite.ctx, &types.MsgMultiSwap{
		Creator:   addr1.String(),
		Routes:    []string{native.Id},
		Input:     sdk.NewInt64Coin(denomBze, 1_000_000),
		MinOutput: sdk.NewInt64Coin(denomStake, 1),
	})
	suite.Require().NoError(err)
	suite.Require().True(resp.Output.IsPositive())
	stored, _ := suite.k.GetLiquidityPool(suite.ctx, native.Id)
	suite.Require().True(stored.ReserveQuote.GT(native.ReserveQuote), "the swap executed")
}

// Exits keep working: cancelling an order on a halted market queues the cancel as always.
func (suite *IntegrationTestSuite) TestMsgHalted_CancelOrder_QueuesNormally() {
	m := haltedQuoteMarket()
	suite.k.SetMarket(suite.ctx, m)
	order := suite.k.NewOrder(suite.ctx, types.Order{
		MarketId:  marketIdOf(m),
		OrderType: types.OrderTypeBuy,
		Amount:    "1000000",
		Price:     "2",
		Owner:     getTestAddress(),
	})
	suite.setHaltedDenoms(denomHalted)

	_, err := suite.msgServer.CancelOrder(suite.ctx, &types.MsgCancelOrder{
		Creator:   getTestAddress(),
		MarketId:  marketIdOf(m),
		OrderId:   order.Id,
		OrderType: order.OrderType,
	})
	suite.Require().NoError(err)

	qm := suite.k.GetAllQueueMessage(suite.ctx)
	suite.Require().Len(qm, 1)
	suite.Require().Equal(types.MessageTypeCancel, qm[0].MessageType)
	suite.Require().Equal(order.Id, qm[0].OrderId)
	suite.Require().True(suite.k.HasPendingCancel(suite.ctx, marketIdOf(m), order.OrderType, order.Id))
}

// Exits keep working: removing liquidity from a halted pool pays out both reserves as always.
func (suite *IntegrationTestSuite) TestMsgHalted_RemoveLiquidity_PaysOutNormally() {
	pool := haltedPool()
	pool.ReserveBase = math.NewInt(1000)
	pool.ReserveQuote = math.NewInt(2000)
	suite.k.SetLiquidityPool(suite.ctx, pool)
	suite.setHaltedDenoms(denomHalted)

	acc := getTestAccount()
	lpTokens := sdk.NewCoins(sdk.NewCoin(pool.LpDenom, math.NewInt(100)))
	suite.bankMock.EXPECT().GetSupply(gomock.Any(), pool.LpDenom).Return(sdk.NewCoin(pool.LpDenom, math.NewInt(1000))).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), acc, types.ModuleName, lpTokens).Return(nil).Times(1)
	suite.bankMock.EXPECT().BurnCoins(gomock.Any(), types.ModuleName, lpTokens).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, acc,
		sdk.NewCoins(sdk.NewInt64Coin(denomStake, 100), sdk.NewInt64Coin(denomHalted, 200)),
	).Return(nil).Times(1)

	resp, err := suite.msgServer.RemoveLiquidity(suite.ctx, &types.MsgRemoveLiquidity{
		Creator:  acc.String(),
		PoolId:   pool.Id,
		LpTokens: math.NewInt(100),
		MinBase:  math.NewInt(90),
		MinQuote: math.NewInt(190),
	})
	suite.Require().NoError(err)
	suite.Require().Equal(math.NewInt(100), resp.Base)
	suite.Require().Equal(math.NewInt(200), resp.Quote)

	stored, _ := suite.k.GetLiquidityPool(suite.ctx, pool.Id)
	suite.Require().Equal(math.NewInt(900), stored.ReserveBase)
	suite.Require().Equal(math.NewInt(1800), stored.ReserveQuote)
}
