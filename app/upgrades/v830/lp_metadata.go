package v830

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
)

// TradebinKeeper is the part of the tradebin keeper the handler uses.
type TradebinKeeper interface {
	GetAllLiquidityPool(ctx sdk.Context) []tradebintypes.LiquidityPool
}

// MetadataKeeper is the part of the bank keeper that reads and writes denom metadata.
type MetadataKeeper interface {
	GetDenomMetaData(ctx context.Context, denom string) (banktypes.Metadata, bool)
	SetDenomMetaData(ctx context.Context, denomMetaData banktypes.Metadata)
}

// backfillLpMetadata gives every pool's LP share denom the name, symbol and description that pool creation left
// blank before v8.3.0. The bank module refuses metadata without a name or symbol, so the blank entries made exported
// genesis fail validation. Only blank fields are filled; the denom units, base and display stay as stored. The LP
// denom is read from the pool, never derived: pre-hash pools keep their legacy "ulp_<base>_<quote>" denom. A pool
// without metadata, or whose metadata still fails validation after the fill, is logged and left as it is.
func backfillLpMetadata(ctx sdk.Context, tradebin TradebinKeeper, bank MetadataKeeper) {
	for _, pool := range tradebin.GetAllLiquidityPool(ctx) {
		denom := pool.GetLpDenom()
		md, found := bank.GetDenomMetaData(ctx, denom)
		if !found {
			ctx.Logger().Info("LP denom has no metadata, skipped", "pool", pool.GetId(), "denom", denom)
			continue
		}
		if md.Name != "" && md.Symbol != "" {
			continue
		}

		if md.Name == "" {
			md.Name = tradebintypes.LpDenomName(pool.GetId())
		}
		if md.Symbol == "" {
			md.Symbol = tradebintypes.LpDenomSymbol(pool.GetId())
		}
		if md.Description == "" {
			md.Description = tradebintypes.LpDenomDescription(pool.GetId())
		}
		if err := md.Validate(); err != nil {
			ctx.Logger().Error("LP denom metadata still invalid, skipped", "pool", pool.GetId(), "denom", denom, "error", err)
			continue
		}

		bank.SetDenomMetaData(ctx, md)
		ctx.Logger().Info("backfilled LP denom metadata", "pool", pool.GetId(), "denom", denom)
	}
}
