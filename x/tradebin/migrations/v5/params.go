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

// MigrateParams gives params 7, 8 and 11 their consensus version 5 meaning.
//
// maxFillsPerOrder reuses proto field 11 of the params, which was orderBookPerBlockMessages (the
// number of queued messages executed per block) before consensus version 5. The field number is the
// same, so a non-zero stored value is read as the new param and kept as it is: the old default 500
// becomes a cap of 500 resting orders examined (filled or skipped) per order. A stored 0 becomes the
// default.
//
// priceLevelOwnersWindow (field 7) was the queue depth before the queue surcharge started
// (orderBookExtraGasWindow, default 100). It now counts distinct owners at one price level, so it is
// set to its default, 500, whatever was stored. priceLevelExtraGas (field 8) was the queue surcharge
// per message (orderBookQueueExtraGas) and keeps its stored value as the surcharge per owner; a stored
// 0 becomes the default.
func MigrateParams(ctx context.Context, k ParamsKeeper) error {
	params := k.GetParams(ctx)
	if params.MaxFillsPerOrder == 0 {
		params.MaxFillsPerOrder = v2types.DefaultMaxFillsPerOrder
	}

	params.PriceLevelOwnersWindow = v2types.DefaultPriceLevelOwnersWindow
	if params.PriceLevelExtraGas == 0 {
		params.PriceLevelExtraGas = v2types.DefaultPriceLevelExtraGas
	}

	return k.SetParams(ctx, params)
}
