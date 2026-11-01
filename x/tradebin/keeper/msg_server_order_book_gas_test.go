package keeper_test

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// TestMsgCreateOrder_TakerFillGasIsMeteredInTx measures the gas of one MsgCreateOrder that fills one
// resting maker order: the matching executes in the transaction, so it is charged to the
// transaction's gas meter.
func (suite *IntegrationTestSuite) TestMsgCreateOrder_TakerFillGasIsMeteredInTx() {
	suite.k.SetMarket(suite.ctx, market)
	maker := sdk.AccAddress("addr1_______________")
	taker := sdk.AccAddress("addr2_______________")
	params := suite.k.GetParams(suite.ctx)

	// the maker order rests on the suite context
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeSell, "1000000", "2", maker))

	takerFee := sdk.NewCoins(params.MarketTakerFee)
	escrow := sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 2_000_000))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), taker, types.ModuleName, takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), takerFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), taker, types.ModuleName, escrow).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, maker, escrow).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, taker, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 1_000_000))).Return(nil).Times(1)

	// only the taker message runs on the fresh gas meter
	gasCtx := suite.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	_, err := suite.msgServer.CreateOrder(gasCtx, &types.MsgCreateOrder{
		Creator:   taker.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "1000000",
		Price:     "2",
	})
	suite.Require().NoError(err)

	gasUsed := gasCtx.GasMeter().GasConsumed()
	suite.T().Logf("gas consumed by one MsgCreateOrder filling one resting maker order: %d", gasUsed)
	suite.Require().Greater(gasUsed, uint64(0))

	// the fill happened in that same call
	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
	suite.Require().Empty(suite.k.GetAllAggregatedOrder(suite.ctx))
	suite.Require().Len(suite.k.GetAllHistoryOrder(suite.ctx), 1)
}
