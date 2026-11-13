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

// tbAggBookState is everything an order adds to the order book.
type tbAggBookState struct {
	orders      []types.Order
	aggregates  []types.AggregatedOrder
	ownerPrice  []types.OrderReference
	priceOrders []types.OrderReference
	userOrders  []types.OrderReference
	counter     uint64
	raw         map[string]string
}

func (suite *IntegrationTestSuite) tbAggCaptureBook(ctx sdk.Context) tbAggBookState {
	raw := make(map[string]string)
	it := ctx.KVStore(suite.storeKey).Iterator(nil, nil)
	defer it.Close()
	for ; it.Valid(); it.Next() {
		raw[string(it.Key())] = string(it.Value())
	}

	return tbAggBookState{
		orders:      suite.k.GetAllOrder(ctx),
		aggregates:  suite.k.GetAllAggregatedOrder(ctx),
		ownerPrice:  suite.k.GetAllOwnerPriceOrder(ctx),
		priceOrders: suite.k.GetAllPriceOrder(ctx),
		userOrders:  suite.k.GetAllUserOrder(ctx),
		counter:     suite.k.GetOrderCounter(ctx),
		raw:         raw,
	}
}

// tbAggCreateOrder runs CreateOrder and turns a panic inside the handler into its value.
func (suite *IntegrationTestSuite) tbAggCreateOrder(ctx sdk.Context, msg *types.MsgCreateOrder) (err error, panicked any) {
	defer func() {
		panicked = recover()
	}()
	_, err = suite.msgServer.CreateOrder(ctx, msg)

	return err, nil
}

// A buy resting at a price with an amount near 2^255 leaves no room in the 256-bit aggregate of that price level:
// a second such buy from another owner fails (an error or a panic of the handler) instead of resting, and in the
// cached transaction context nothing it wrote before the failure (order, indexes, owner index, counter) is kept.
func (suite *IntegrationTestSuite) TestCreateOrder_AggregateOverflowFailsAndKeepsTheBook() {
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
	before := suite.tbAggCaptureBook(suite.ctx)
	suite.Require().Len(before.orders, 1)
	suite.Require().Equal(uint64(1), before.counter)

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
	}

	// the cache context was not written: the book is exactly as before
	after := suite.tbAggCaptureBook(suite.ctx)
	suite.Require().Equal(before, after)
	agg, found = suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, price)
	suite.Require().True(found)
	suite.Require().Equal(huge.String(), agg.Amount)
	suite.Require().Equal(uint64(1), agg.Owners)
	_, found = suite.k.GetOwnerPriceOrder(suite.ctx, second.String(), getMarketId(), types.OrderTypeBuy, price)
	suite.Require().False(found)
}
