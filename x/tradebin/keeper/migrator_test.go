package keeper_test

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate4to5 gives maxFillsPerOrder its default when the stored field 11 is 0 and keeps any other
// stored value.
func (suite *IntegrationTestSuite) TestMigrate4to5_MaxFillsPerOrder() {
	for _, tc := range []struct{ stored, want uint64 }{{0, 500}, {1, 1}, {750, 750}} {
		stored, want := tc.stored, tc.want
		suite.SetupTest()
		params := suite.k.GetParams(suite.ctx)
		params.MaxFillsPerOrder = stored
		suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

		migrator := keeper.NewMigrator(suite.k, nil, suite.bankMock)
		suite.Require().NoError(migrator.Migrate4to5(suite.ctx))

		got := suite.k.GetParams(suite.ctx)
		suite.Require().Equal(want, got.MaxFillsPerOrder, "stored %d", stored)
		params.MaxFillsPerOrder = want
		suite.Require().Equal(params, got)
	}
}

// Migrate4to5 sets priceLevelOwnersWindow (field 7) to 500 and keeps priceLevelExtraGas (field 8).
func (suite *IntegrationTestSuite) TestMigrate4to5_PriceLevelParams() {
	params := suite.k.GetParams(suite.ctx)
	params.PriceLevelOwnersWindow = 100
	params.PriceLevelExtraGas = 30_000
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	migrator := keeper.NewMigrator(suite.k, nil, suite.bankMock)
	suite.Require().NoError(migrator.Migrate4to5(suite.ctx))

	got := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(uint64(500), got.PriceLevelOwnersWindow)
	suite.Require().Equal(uint64(30_000), got.PriceLevelExtraGas)
}

// Migrate4to5 writes the distinct owners of every price level: three orders from two owners at one
// level count 2, a level of one order counts 1.
func (suite *IntegrationTestSuite) TestMigrate4to5_CountsPriceLevelOwners() {
	owner := sdk.AccAddress("addr1_______________")
	other := sdk.AccAddress("addr2_______________")
	buy := types.Order{Id: orderId(5), MarketId: getMarketId(), OrderType: types.OrderTypeBuy, Amount: "10", Price: "0.5", Owner: other.String()}
	for _, o := range []types.Order{
		restingSell(orderId(1), "10", owner),
		restingSell(orderId(2), "10", other),
		restingSell(orderId(4), "10", owner),
		buy,
	} {
		suite.k.SaveOrder(suite.ctx, o)
	}
	// aggregates as consensus version 4 stored them, without owners
	suite.k.SetAggregatedOrder(suite.ctx, types.AggregatedOrder{MarketId: getMarketId(), OrderType: types.OrderTypeSell, Amount: "30", Price: "1"})
	suite.k.SetAggregatedOrder(suite.ctx, types.AggregatedOrder{MarketId: getMarketId(), OrderType: types.OrderTypeBuy, Amount: "10", Price: "0.5"})

	// the state before consensus version 5 had no owner-price index
	store := suite.ctx.KVStore(suite.storeKey)
	var keys [][]byte
	it := storetypes.KVStorePrefixIterator(store, []byte(types.OwnerPriceKeyPrefix))
	for ; it.Valid(); it.Next() {
		keys = append(keys, it.Key())
	}
	suite.Require().NoError(it.Close())
	for _, key := range keys {
		store.Delete(key)
	}

	migrator := keeper.NewMigrator(suite.k, nil, suite.bankMock)
	suite.Require().NoError(migrator.Migrate4to5(suite.ctx))

	suite.requireOwners(types.OrderTypeSell, "1", 2)
	suite.requireOwners(types.OrderTypeBuy, "0.5", 1)
	sell, _ := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, "1")
	suite.Require().Equal("30", sell.Amount)
}
