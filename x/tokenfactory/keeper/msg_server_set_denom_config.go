package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// SetDenomConfig replaces the denom's config as a whole. Only the current admin may send it,
// so the config freezes once the admin is renounced.
func (k msgServer) SetDenomConfig(goCtx context.Context, msg *types.MsgSetDenomConfig) (*types.MsgSetDenomConfigResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dAuth, err := k.GetDenomAuthority(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	if msg.Creator != dAuth.GetAdmin() {
		return nil, types.ErrUnauthorized
	}

	cfg := msg.Config()
	if cfg.CommunityPool != "" {
		cp, err := sdk.AccAddressFromBech32(cfg.CommunityPool)
		if err != nil {
			return nil, types.ErrInvalidDenomConfig.Wrapf("invalid community pool address: %s", err)
		}
		if k.bankKeeper.BlockedAddr(cp) {
			return nil, types.ErrInvalidDenomConfig.Wrapf("community pool %s is a blocked address", cfg.CommunityPool)
		}
	}

	if err := k.Keeper.SetDenomConfig(ctx, msg.Denom, cfg); err != nil {
		return nil, err
	}

	err = ctx.EventManager().EmitTypedEvent(&types.DenomConfigChangeEvent{
		Denom:         msg.Denom,
		CommunityPool: cfg.CommunityPool,
		TransferFee:   cfg.TransferFee,
		GasSurcharge:  cfg.GasSurcharge,
	})
	if err != nil {
		k.Logger().Error("failed to emit denom config changed event", "error", err)
	}

	return &types.MsgSetDenomConfigResponse{}, nil
}
