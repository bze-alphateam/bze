package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	coin, err := sdk.ParseCoinNormalized(msg.Coins)
	if err != nil || !coin.IsPositive() {
		return nil, types.ErrInvalidAmount.Wrapf("coins: %s", msg.Coins)
	}

	_, denomExists := k.bankKeeper.GetDenomMetaData(ctx, coin.GetDenom())
	if !denomExists {
		return nil, types.ErrDenomDoesNotExist.Wrapf("denom: %s", coin.GetDenom())
	}

	//check denom is a tokenfactory denom
	_, _, err = types.DeconstructDenom(coin.GetDenom())
	if err != nil {
		return nil, err
	}

	dAuth, err := k.Keeper.GetDenomAuthority(ctx, coin.GetDenom())
	if err != nil {
		return nil, err
	}

	if msg.Creator != dAuth.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(coin))
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(coin))
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.TypeMsgMint,
			sdk.NewAttribute(types.AttributeMintToAddress, msg.Creator),
			sdk.NewAttribute(types.AttributeAmount, coin.String()),
		),
	})

	return &types.MsgMintResponse{}, nil
}
