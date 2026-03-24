package app

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	corestoretypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	customAnte "github.com/bze-alphateam/bze/x/txfeecollector/ante"
	"github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
)

type AnteHandlerOptions struct {
	TradeKeeper       types.TradeKeeper
	BankKeeper        types.BankKeeper
	TxCollectorKeeper *keeper.Keeper
}

type WasmAnteHandlerOptions struct {
	NodeConfig            *wasmtypes.NodeConfig
	WasmKeeper            *wasmkeeper.Keeper
	TXCounterStoreService corestoretypes.KVStoreService
}

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
func NewAnteHandler(options ante.HandlerOptions, customOptions AnteHandlerOptions, wasmOptions WasmAnteHandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "account keeper is required for ante builder")
	}

	if options.BankKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "bank keeper is required for ante builder")
	}

	if options.SignModeHandler == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		wasmkeeper.NewLimitSimulationGasDecorator(wasmOptions.NodeConfig.SimulationGasLimit), // after setup context to enforce limits early
		wasmkeeper.NewCountTXDecorator(wasmOptions.TXCounterStoreService),
		wasmkeeper.NewGasRegisterDecorator(wasmOptions.WasmKeeper.GetGasRegister()),
		wasmkeeper.NewTxContractsDecorator(),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		customAnte.NewValidateTxFeeDenomsDecorator(customOptions.TradeKeeper),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		customAnte.NewDeductFeeDecorator(customOptions.TradeKeeper, options.AccountKeeper, customOptions.BankKeeper, options.FeegrantKeeper, customOptions.TxCollectorKeeper),
		customAnte.NewCwDeployFeeDecorator(customOptions.TradeKeeper, customOptions.BankKeeper, options.FeegrantKeeper, customOptions.TxCollectorKeeper),
		ante.NewSetPubKeyDecorator(options.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
