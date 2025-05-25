package keeper

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

func (k Keeper) getMarketAggregatedOrdersPaginated(
	ctx sdk.Context,
	market string,
	orderType string,
	pageReq *query.PageRequest,
) (orders []types.AggregatedOrder, response *query.PageResponse, err error) {

	aggOrderStore := k.getAggregatedOrderByMarketAndTypeStore(ctx, market, orderType)
	response, err = query.Paginate(aggOrderStore, pageReq, func(key []byte, value []byte) error {
		var order types.AggregatedOrder
		if err := k.cdc.Unmarshal(value, &order); err != nil {
			return err
		}

		orders = append(orders, order)
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return orders, response, nil
}
