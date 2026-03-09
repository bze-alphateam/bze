package ante

import (
	sdkerrors "cosmossdk.io/errors"
	storeTypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	FeeDenomKey = "fee_denom"
)

// ValidateTxFeeDenomsDecorator will check if denominations used for tx fees are allowed and returns an error otherwise
type ValidateTxFeeDenomsDecorator struct {
	tradeKeeper types.TradeKeeper
}

func NewValidateTxFeeDenomsDecorator(tradeKeeper types.TradeKeeper) ValidateTxFeeDenomsDecorator {
	return ValidateTxFeeDenomsDecorator{
		tradeKeeper: tradeKeeper,
	}
}

func (vbd ValidateTxFeeDenomsDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(storeTypes.ErrTxDecode, "ValidateTxFeeDenomsDecorator requires tx to be a FeeTx")
	}

	// On ReCheckTx, skip validation but still set FeeDenomKey in context
	// so that downstream handlers (DeductFeeDecorator) can compute min gas prices
	// in the correct denomination.
	if ctx.IsReCheckTx() {
		if !feeTx.GetFee().Empty() {
			ctx = ctx.WithValue(FeeDenomKey, feeTx.GetFee()[0].Denom)
		}
		return next(ctx, tx, simulate)
	}

	if feeTx.GetFee().Len() > 1 {
		return ctx, sdkerrors.Wrap(storeTypes.ErrInvalidRequest, "multiple denominations for same transaction fee are not supported")
	}

	// Allow empty fees during genesis or simulation
	if feeTx.GetFee().Empty() {
		if simulate || ctx.BlockHeight() == 0 {
			return next(ctx, tx, simulate)
		}
		return ctx, sdkerrors.Wrap(storeTypes.ErrInvalidRequest, "no fee supplied")
	}

	c := feeTx.GetFee()[0]
	if !c.IsPositive() {
		return ctx, sdkerrors.Wrap(storeTypes.ErrInvalidRequest, "the provided transaction fee must be positive")
	}

	// Check if tradeKeeper is available before calling its methods
	if vbd.tradeKeeper == nil {
		// Without tradeKeeper, we can't validate non-native denoms
		// This should ideally not happen in production
		return ctx, sdkerrors.Wrapf(
			storeTypes.ErrInvalidRequest,
			"invalid fee supplied. can not use %s denom as tx fee",
			c.Denom,
		)
	}

	if !vbd.tradeKeeper.IsNativeDenom(ctx, c.Denom) {
		if !vbd.tradeKeeper.HasDeepLiquidityWithNativeDenom(ctx, c.Denom) {
			return ctx, sdkerrors.Wrapf(
				storeTypes.ErrInvalidRequest,
				"%s can be used to pay for fees only if enough liquidity is available",
				c.Denom,
			)
		}
	}

	ctx = ctx.WithValue(FeeDenomKey, c.Denom)

	return next(ctx, tx, simulate)
}
