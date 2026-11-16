package keeper_test

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func restingSell(id, amount string, owner sdk.AccAddress) types.Order {
	return types.Order{Id: id, MarketId: getMarketId(), OrderType: types.OrderTypeSell, Amount: amount, Price: "1", Owner: owner.String()}
}

// Two orders of one owner at one price (possible before v8.3.0): the index points at the lower id
// whatever order they are saved in, and only the order it points at removes it.
func (suite *IntegrationTestSuite) TestOwnerPriceOrder_DuplicatesKeepTheLowestId() {
	owner := sdk.AccAddress("addr1_______________")
	low := restingSell(orderId(3), "10", owner)
	high := restingSell(orderId(7), "10", owner)

	suite.k.SaveOrder(suite.ctx, high)
	suite.k.SaveOrder(suite.ctx, low)
	suite.requireOwnerPriceOrder(owner, types.OrderTypeSell, "1", low.Id)

	// a partial fill of the higher id saves it again
	high.Amount = "5"
	suite.k.SaveOrder(suite.ctx, high)
	suite.requireOwnerPriceOrder(owner, types.OrderTypeSell, "1", low.Id)

	suite.k.RemoveOrder(suite.ctx, high)
	suite.requireOwnerPriceOrder(owner, types.OrderTypeSell, "1", low.Id)

	suite.k.RemoveOrder(suite.ctx, low)
	suite.requireNoOwnerPriceOrder(owner, types.OrderTypeSell, "1")
}

// With the higher id still resting after the lowest one is gone, the entry is not repointed: the
// next save of the higher one indexes it again.
func (suite *IntegrationTestSuite) TestOwnerPriceOrder_RemainingDuplicateIsIndexedOnItsNextSave() {
	owner := sdk.AccAddress("addr1_______________")
	low := restingSell(orderId(3), "10", owner)
	high := restingSell(orderId(7), "10", owner)
	suite.k.SaveOrder(suite.ctx, low)
	suite.k.SaveOrder(suite.ctx, high)

	suite.k.RemoveOrder(suite.ctx, low)
	suite.requireNoOwnerPriceOrder(owner, types.OrderTypeSell, "1")
	_, found := suite.k.GetOrder(suite.ctx, getMarketId(), types.OrderTypeSell, high.Id)
	suite.Require().True(found)

	high.Amount = "5"
	suite.k.SaveOrder(suite.ctx, high)
	suite.requireOwnerPriceOrder(owner, types.OrderTypeSell, "1", high.Id)
}

// The key is owner, market, side and the normalised price.
func (suite *IntegrationTestSuite) TestOwnerPriceOrder_Key() {
	owner := sdk.AccAddress("addr1_______________")
	other := sdk.AccAddress("addr2_______________")
	order := types.Order{Id: orderId(1), MarketId: getMarketId(), OrderType: types.OrderTypeBuy, Amount: "10", Price: "1.5", Owner: owner.String()}
	suite.k.SaveOrder(suite.ctx, order)

	for _, price := range []string{"1.5", "1.50", "1.500000000000000000"} {
		suite.requireOwnerPriceOrder(owner, types.OrderTypeBuy, price, order.Id)
	}
	suite.requireNoOwnerPriceOrder(owner, types.OrderTypeSell, "1.5")
	suite.requireNoOwnerPriceOrder(owner, types.OrderTypeBuy, "1.6")
	suite.requireNoOwnerPriceOrder(other, types.OrderTypeBuy, "1.5")
	_, found := suite.k.GetOwnerPriceOrder(suite.ctx, owner.String(), "other/market", types.OrderTypeBuy, "1.5")
	suite.Require().False(found)

	suite.Require().Equal([]types.OrderReference{{Id: order.Id, MarketId: getMarketId(), OrderType: types.OrderTypeBuy}}, suite.k.GetAllOwnerPriceOrder(suite.ctx))
}

// Migrate4to5 builds the index from the resting orders; two orders of one owner at one price stay
// separate and the index points at the lower id.
func (suite *IntegrationTestSuite) TestMigrate4to5_BuildsOwnerPriceIndex() {
	owner := sdk.AccAddress("addr1_______________")
	other := sdk.AccAddress("addr2_______________")
	orders := []types.Order{
		restingSell(orderId(1), "10", owner),
		restingSell(orderId(2), "10", other),
		restingSell(orderId(4), "10", owner),
	}
	for _, o := range orders {
		suite.k.SaveOrder(suite.ctx, o)
	}

	// the state before consensus version 5 had no owner-price index
	store := suite.ctx.KVStore(suite.storeKey)
	var keys [][]byte
	it := storetypes.KVStorePrefixIterator(store, []byte(types.OwnerPriceKeyPrefix))
	for ; it.Valid(); it.Next() {
		keys = append(keys, it.Key())
	}
	suite.Require().NoError(it.Close())
	suite.Require().Len(keys, 2)
	for _, key := range keys {
		store.Delete(key)
	}
	suite.Require().Empty(suite.k.GetAllOwnerPriceOrder(suite.ctx))

	migrator := keeper.NewMigrator(suite.k, nil, suite.bankMock)
	suite.Require().NoError(migrator.Migrate4to5(suite.ctx))

	suite.requireOwnerPriceOrder(owner, types.OrderTypeSell, "1", orderId(1))
	suite.requireOwnerPriceOrder(other, types.OrderTypeSell, "1", orderId(2))
	suite.Require().Len(suite.k.GetAllOwnerPriceOrder(suite.ctx), 2)
	suite.Require().Equal(orders, suite.k.GetAllOrder(suite.ctx))
}
