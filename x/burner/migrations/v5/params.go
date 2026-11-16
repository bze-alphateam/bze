package v5

import (
	"context"

	"github.com/bze-alphateam/bze/x/burner/types"
)

// ParamsKeeper reads and writes the module params.
type ParamsKeeper interface {
	GetParams(ctx context.Context) types.Params
	SetParams(ctx context.Context, params types.Params) error
}

// MigrateParams gives the params stored before consensus version 5 the limits of MsgMoveIbcLockedCoins:
// ibc_move_max_reserve_ratio and ibc_move_cooldown_blocks are read as unset (nil / 0) and get their defaults. A value
// that is already set is kept. periodic_burning_weeks is not touched.
func MigrateParams(ctx context.Context, k ParamsKeeper) error {
	params := k.GetParams(ctx)
	if params.IbcMoveMaxReserveRatio.IsNil() || params.IbcMoveMaxReserveRatio.IsZero() {
		params.IbcMoveMaxReserveRatio = types.DefaultIbcMoveMaxReserveRatio
	}

	if params.IbcMoveCooldownBlocks == 0 {
		params.IbcMoveCooldownBlocks = types.DefaultIbcMoveCooldownBlocks
	}

	return k.SetParams(ctx, params)
}
