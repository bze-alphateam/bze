package ante

import (
	sdkerrors "cosmossdk.io/errors"
	storeTypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	// no need to validate basic on recheck tx, call next antehandler
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(storeTypes.ErrTxDecode, "ValidateTxFeeDenomsDecorator requires tx to be a FeeTx")
	}

	if feeTx.GetFee().Len() > 1 {
		return ctx, sdkerrors.Wrap(storeTypes.ErrInvalidRequest, "multiple denominations for same transaction fee are not supported")
	}

	if feeTx.GetFee().Empty() {
		return ctx, sdkerrors.Wrap(storeTypes.ErrInvalidRequest, "no fee supplied")
	}

	c := feeTx.GetFee()[0]
	if !vbd.tradeKeeper.IsNativeDenom(ctx, c.Denom) {
		//if trading module (keeper) is not available we do not allow anything else than the main denom
		if vbd.tradeKeeper == nil {
			return ctx, sdkerrors.Wrapf(
				storeTypes.ErrInvalidRequest,
				"invalid fee supplied. can not use %s denom as tx fee",
				c.Denom,
			)
		}

		if !vbd.tradeKeeper.CanSwapForNativeDenom(ctx, c) {
			return ctx, sdkerrors.Wrapf(
				storeTypes.ErrInvalidRequest,
				"%s can be used to pay for fees only if enough liquidity is available",
				c.Denom,
			)
		}
	}

	ctx = ctx.WithValue("fee_denom", c.Denom)

	return next(ctx, tx, simulate)
}
