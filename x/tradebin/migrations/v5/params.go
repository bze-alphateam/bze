package v5

import (
	"context"

	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

// ParamsKeeper reads and writes the module params.
type ParamsKeeper interface {
	GetParams(ctx context.Context) v2types.Params
	SetParams(ctx context.Context, params v2types.Params) error
}

// MigrateParams sets maxFillsPerOrder to its default when the stored value is 0.
//
// maxFillsPerOrder reuses proto field 11 of the params, which was orderBookPerBlockMessages (the
// number of queued messages executed per block) before consensus version 5. The field number is the
// same, so a non-zero stored value is read as the new param and kept as it is: the old default 500
// becomes a cap of 500 fills per order.
func MigrateParams(ctx context.Context, k ParamsKeeper) error {
	params := k.GetParams(ctx)
	if params.MaxFillsPerOrder != 0 {
		return nil
	}

	params.MaxFillsPerOrder = v2types.DefaultMaxFillsPerOrder

	return k.SetParams(ctx, params)
}
