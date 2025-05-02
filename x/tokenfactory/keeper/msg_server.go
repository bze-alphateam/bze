package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) CreateDenom(goCtx context.Context, msg *types.MsgCreateDenom) (*types.MsgCreateDenomResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	denom, err := k.validateCreateDenom(ctx, msg.Creator, msg.Subdenom)
	if err != nil {
		return nil, err
	}

	err = k.chargeForCreateDenom(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	err = k.CreateDenomAfterValidation(ctx, msg.Creator, denom)

	return &types.MsgCreateDenomResponse{NewDenom: denom}, err
}

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

	return &types.MsgMintResponse{}, nil
}

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

	return &types.MsgBurnResponse{}, nil
}

func (k msgServer) ChangeAdmin(goCtx context.Context, msg *types.MsgChangeAdmin) (*types.MsgChangeAdminResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dAuth, err := k.Keeper.GetDenomAuthority(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	if msg.NewAdmin != "" {
		//validate new admin address only if one was provided
		_, err = sdk.AccAddressFromBech32(msg.NewAdmin)
		if err != nil {
			return nil, err
		}
	}

	if msg.Creator != dAuth.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	dAuth.Admin = msg.NewAdmin
	err = k.Keeper.SetDenomAuthority(ctx, msg.Denom, dAuth)
	if err != nil {
		return nil, err
	}

	err = ctx.EventManager().EmitTypedEvent(&types.DenomAdminChangeEvent{
		Admin:    msg.Creator,
		NewAdmin: msg.NewAdmin,
		Denom:    msg.Denom,
	})

	if err != nil {
		k.Logger().Error("failed to emit admin changed event", "error", err)
	}

	return &types.MsgChangeAdminResponse{}, nil
}

func (k msgServer) SetDenomMetadata(goCtx context.Context, msg *types.MsgSetDenomMetadata) (*types.MsgSetDenomMetadataResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Defense in depth validation of metadata
	err := msg.Metadata.Validate()
	if err != nil {
		return nil, err
	}

	dAuth, err := k.Keeper.GetDenomAuthority(ctx, msg.Metadata.Base)
	if err != nil {
		return nil, err
	}

	if msg.Creator != dAuth.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	k.Keeper.bankKeeper.SetDenomMetaData(ctx, msg.Metadata)

	err = ctx.EventManager().EmitTypedEvent(&types.DenomMetadataChangeEvent{Denom: msg.Metadata.Base})
	if err != nil {
		k.Logger().Error("failed to emit metadata changed event", "error", err)
	}

	return &types.MsgSetDenomMetadataResponse{}, nil
}
