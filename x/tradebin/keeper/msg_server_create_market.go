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

	k.SetMarket(ctx, types.Market{
		Base:    msg.Base,
		Quote:   msg.Quote,
		Creator: msg.Creator,
	})

	return &types.MsgCreateMarketResponse{}, nil
}