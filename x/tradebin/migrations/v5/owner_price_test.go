package v5_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	v5 "github.com/bze-alphateam/bze/x/tradebin/migrations/v5"
	tbtestutil "github.com/bze-alphateam/bze/x/tradebin/testutil"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

func TestMigrateOwnerPriceIndex_IndexesEveryOrderInStoreOrder(t *testing.T) {
	mt := setup(t)
	k := tbtestutil.NewMockV5OwnerPriceKeeper(gomock.NewController(t))
	owner := sdk.AccAddress("owner_______________").String()
	orders := []types.Order{
		{Id: "000000000000000000000001", MarketId: marketId, OrderType: types.OrderTypeSell, Amount: "10", Price: "1", Owner: owner},
		{Id: "000000000000000000000004", MarketId: marketId, OrderType: types.OrderTypeSell, Amount: "10", Price: "1", Owner: owner},
		{Id: "000000000000000000000002", MarketId: marketId, OrderType: types.OrderTypeBuy, Amount: "5", Price: "0.5", Owner: owner},
	}

	calls := []any{k.EXPECT().GetAllOrder(gomock.Any()).Return(orders).Times(1)}
	for _, o := range orders {
		calls = append(calls, k.EXPECT().IndexOwnerPriceOrder(gomock.Any(), o).Times(1))
	}
	gomock.InOrder(calls...)

	v5.MigrateOwnerPriceIndex(mt.ctx, k)
}

func TestMigrateOwnerPriceIndex_NoOrders(t *testing.T) {
	mt := setup(t)
	k := tbtestutil.NewMockV5OwnerPriceKeeper(gomock.NewController(t))
	k.EXPECT().GetAllOrder(gomock.Any()).Return(nil).Times(1)

	v5.MigrateOwnerPriceIndex(mt.ctx, k)
}
