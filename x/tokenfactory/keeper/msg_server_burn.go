package keeper

import (
	"context"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) Burn(goCtx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	coin, err := sdk.ParseCoinNormalized(msg.Coins)
	if err != nil || !coin.IsPositive() {
		return nil, types.ErrInvalidAmount.Wrapf("coins: %s", msg.Coins)
	}

	dAuth, err := k.Keeper.GetDenomAuthority(ctx, coin.GetDenom())
	if err != nil {
		return nil, err
	}

	if msg.Creator != dAuth.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	accAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	//make sure it's not a module account
	burnFrom := k.Keeper.accountKeeper.GetAccount(ctx, accAddr)
	_, ok := burnFrom.(authtypes.ModuleAccountI)
	if ok {
		return nil, types.ErrBurnFromModuleAccount
	}

	_, _, err = types.DeconstructDenom(coin.GetDenom())
	if err != nil {
		return nil, err
	}

	//send to module
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, accAddr, types.ModuleName, sdk.NewCoins(coin))
	if err != nil {
		return nil, err
	}

	//burn the coins from module
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(coin))
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.TypeMsgBurn,
			sdk.NewAttribute(types.AttributeBurnFromAddress, msg.Creator),
			sdk.NewAttribute(types.AttributeAmount, coin.String()),
		),
	})

	return &types.MsgBurnResponse{}, nil
}
