package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
)

const (
	denomMainnet = "ubze"
	denomTestnet = "utbz"
)

func NewAnteHandler(options ante.HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "account keeper is required for ante builder")
	}

	if options.BankKeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "bank keeper is required for ante builder")
	}

	if options.SignModeHandler == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}

	sigGasConsumer := options.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = ante.DefaultSigVerificationGasConsumer
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewRejectExtensionOptionsDecorator(),
		ante.NewMempoolFeeDecorator(),
		ante.NewValidateBasicDecorator(),
		NewValidateTxFeeDenomsDecorator(), //use our own validate basic to enforce denoms that should be used for fees
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper),
		ante.NewSetPubKeyDecorator(options.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, sigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}

// ValidateTxFeeDenomsDecorator will check if denominations used for tx fees are allowed and returns an error otherwise
type ValidateTxFeeDenomsDecorator struct{}

func NewValidateTxFeeDenomsDecorator() ValidateTxFeeDenomsDecorator {
	return ValidateTxFeeDenomsDecorator{}
}

func (vbd ValidateTxFeeDenomsDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// no need to validate basic on recheck tx, call next antehandler
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "ValidateTxFeeDenomsDecorator requires tx to be a FeeTx")
	}

	for _, c := range feeTx.GetFee() {
		if !vbd.isAllowedDenom(c.Denom) {
			return ctx, sdkerrors.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"invalid fee supplied. cannot pay fee in %s denomination. Allowed denomination is %s for mainnet and %s for testnet",
				c.Denom,
				denomMainnet,
				denomTestnet,
			)
		}
	}

	return next(ctx, tx, simulate)
}

func (vbd ValidateTxFeeDenomsDecorator) isAllowedDenom(denom string) bool {
	return denom == denomMainnet || denom == denomTestnet
}
