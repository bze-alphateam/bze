package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateOrder(goCtx context.Context, msg *types.MsgCreateOrder) (*types.MsgCreateOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	market, found := k.GetMarketById(ctx, msg.MarketId)
	if !found {
		return nil, types.ErrMarketNotFound.Wrapf("market id: %s", msg.MarketId)
	}

	//calculate needed funds for this order
	coin, err := k.getOrderNeededCoins(msg, &market)
	if err != nil {
		return nil, err
	}

	accAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	//capture user funds for this order
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, accAddr, types.ModuleName, sdk.NewCoins(coin))
	if err != nil {
		return nil, err
	}

	qm := types.QueueMessage{
		MarketId:    msg.MarketId,
		MessageType: msg.OrderType,
		Amount:      coin.Amount.Int64(),
		Price:       msg.Price,
	}

	k.SetQueueMessage(ctx, qm)

	_ = ctx

	return &types.MsgCreateOrderResponse{}, nil
}
