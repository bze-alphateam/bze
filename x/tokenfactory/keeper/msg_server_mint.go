package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	_, denomExists := k.bankKeeper.GetDenomMetaData(ctx, msg.Denom)
	if !denomExists {
		return nil, types.ErrDenomDoesNotExist.Wrapf("denom: %s", msg.Denom)
	}

	dAuth, err := k.Keeper.GetDenomAuthority(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	if msg.Creator != dAuth.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	//check denom is a tokenfactory denom
	_, _, err = types.DeconstructDenom(msg.Denom)
	if err != nil {
		return nil, err
	}

	amountInt, ok := sdk.NewIntFromString(msg.Amount)
	if !ok || !amountInt.IsPositive() {
		return nil, types.ErrInvalidAmount.Wrapf("converting amount [%s] to int [%d] failed", msg.Amount, amountInt.Int64())
	}

	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	coin := sdk.NewCoin(msg.Denom, amountInt)
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
			sdk.NewAttribute(types.AttributeAmount, msg.Amount),
		),
	})

	return &types.MsgMintResponse{}, nil
}
