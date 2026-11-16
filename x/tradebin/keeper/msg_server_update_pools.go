package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// UpdatePools is the governance message that changes the swap fee and the fee split of existing pools.
// Only Fee and FeeDest are written: the curve, reserves and LP denom never change (switching the curve
// would change the invariant). All or nothing: an unknown pool fails the message, and gov (like x/dao)
// runs it in a cache context, so the writes of the earlier entries are discarded with it. Halted pools
// can be updated, the halt only blocks trading.
// Idempotent: an entry equal to the pool's current values writes nothing and emits no event.
func (k msgServer) UpdatePools(goCtx context.Context, msg *types.MsgUpdatePools) (*types.MsgUpdatePoolsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if err := msg.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	for _, u := range msg.Updates {
		pool, found := k.GetLiquidityPool(ctx, u.PoolId)
		if !found {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "pool %s not found", u.PoolId)
		}

		oldFeeDest := types.FeeDestination{}
		if pool.FeeDest != nil {
			oldFeeDest = *pool.FeeDest
		}

		if pool.Fee.Equal(u.Fee) && pool.FeeDest != nil && feeDestEqual(oldFeeDest, u.FeeDest) {
			continue
		}

		oldFee := pool.Fee
		newFeeDest := u.FeeDest
		pool.Fee = u.Fee
		pool.FeeDest = &newFeeDest
		k.SetLiquidityPool(ctx, pool)

		err := ctx.EventManager().EmitTypedEvent(&types.PoolFeesUpdatedEvent{
			PoolId:     pool.Id,
			OldFee:     oldFee.String(),
			NewFee:     u.Fee.String(),
			OldFeeDest: oldFeeDest,
			NewFeeDest: u.FeeDest,
		})
		if err != nil {
			k.Logger().Error(err.Error())
		}
	}

	return &types.MsgUpdatePoolsResponse{}, nil
}

func feeDestEqual(a, b types.FeeDestination) bool {
	return a.Treasury.Equal(b.Treasury) && a.Burner.Equal(b.Burner) && a.Providers.Equal(b.Providers)
}
