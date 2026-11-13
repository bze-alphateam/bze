package keeper_test

import (
	"fmt"
	"math/big"
	"strings"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// tbAggCreateOrder runs CreateOrder and turns a panic inside the handler into its value.
func (suite *IntegrationTestSuite) tbAggCreateOrder(ctx sdk.Context, msg *types.MsgCreateOrder) (err error, panicked any) {
	defer func() {
		panicked = recover()
	}()
	_, err = suite.msgServer.CreateOrder(ctx, msg)

	return err, nil
}

// A buy resting at a price with an amount near 2^255 leaves no room in the 256-bit aggregate of that price level:
// a second such buy from another owner fails with an overflow (an error or a panic of the handler) instead of
// resting. The transaction's writes are then discarded by baseapp, as for any failed message.
func (suite *IntegrationTestSuite) TestCreateOrder_AggregateOverflowFails() {
	suite.k.SetMarket(suite.ctx, market)
	first := sdk.AccAddress("addr1_______________")
	second := sdk.AccAddress("addr2_______________")
	price := "1"
	huge := math.NewIntFromBigInt(new(big.Int).Lsh(big.NewInt(1), 255))

	// the first order rests on its own: the aggregate holds 2^255 and one owner
	suite.placeOrders(suite.newEngine(), limitOrder(types.OrderTypeBuy, huge.String(), price, first))
	agg, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, price)
	suite.Require().True(found)
	suite.Require().Equal(huge.String(), agg.Amount)
	suite.Require().Equal(uint64(1), agg.Owners)
	suite.Require().Len(suite.k.GetAllOrder(suite.ctx), 1)
	suite.Require().Equal(uint64(1), suite.k.GetOrderCounter(suite.ctx))

	// the second buy pays its fee and escrows 2^255 quote, then reaches the book
	params := suite.k.GetParams(suite.ctx)
	makerFee := sdk.NewCoins(params.MarketMakerFee)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), second, types.ModuleName, makerFee).Return(nil).AnyTimes()
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), makerFee).Return(nil).AnyTimes()
	escrow := sdk.NewCoins(sdk.NewCoin(market.Quote, huge))
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), second, types.ModuleName, escrow).Return(nil).Times(1)

	cacheCtx, _ := suite.ctx.CacheContext()
	err, panicked := suite.tbAggCreateOrder(cacheCtx, &types.MsgCreateOrder{
		Creator:   second.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    huge.String(),
		Price:     price,
	})
	if panicked != nil {
		suite.Require().Contains(strings.ToLower(fmt.Sprint(panicked)), "overflow", "the handler failed for another reason")
	} else {
		suite.Require().Error(err, "a second 2^255 buy at the same price must not rest")
		suite.Require().Contains(strings.ToLower(err.Error()), "overflow", "the handler failed for another reason")
	}
}
