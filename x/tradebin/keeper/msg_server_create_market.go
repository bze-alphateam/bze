package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateMarket(goCtx context.Context, msg *types.MsgCreateMarket) (*types.MsgCreateMarketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	_, found := k.GetMarket(ctx, msg.Base, msg.Quote)
	if found {
		return nil, types.ErrMarketAlreadyExists
	}

	//check aliases too: user can try to create a market that exists
	_, found = k.GetMarketAlias(ctx, msg.Base, msg.Quote)
	if found {
		return nil, types.ErrMarketAlreadyExists
	}

	if msg.Base == "" || msg.Quote == "" || msg.Quote == msg.Base {
		return nil, types.ErrInvalidDenom
	}

	if !k.bankKeeper.HasSupply(ctx, msg.Base) {
		return nil, types.ErrDenomHasNoSupply
	}

	if !k.bankKeeper.HasSupply(ctx, msg.Quote) {
		return nil, types.ErrDenomHasNoSupply
	}

	createMarketFee, err := sdk.ParseCoinsNormalized(k.CreateMarketFee(ctx))
	if err != nil {
		return nil, err
	}

	if createMarketFee.IsAllPositive() {
		accAddr, err := sdk.AccAddressFromBech32(msg.Creator)
		if err != nil {
			return nil, err
		}

		sendErr := k.distrKeeper.FundCommunityPool(ctx, createMarketFee, accAddr)
		if sendErr != nil {
			return nil, sendErr
		}
	}

	market := types.Market{
		Base:    msg.Base,
		Quote:   msg.Quote,
		Creator: msg.Creator,
	}
	k.SetMarket(ctx, market)

	err = k.emitMarketCreatedEvent(ctx, &market)

	return &types.MsgCreateMarketResponse{}, nil
}

func (k msgServer) emitMarketCreatedEvent(ctx sdk.Context, market *types.Market) error {
	return ctx.EventManager().EmitTypedEvent(
		&types.MarketCreatedEvent{
			Creator: market.Creator,
			Base:    market.Base,
			Quote:   market.Quote,
		},
	)
}
