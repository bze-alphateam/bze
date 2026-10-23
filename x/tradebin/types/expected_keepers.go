package types

import (
	"context"

	"cosmossdk.io/math"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AccountKeeper interface {
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	HasSupply(ctx context.Context, denom string) bool
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SetDenomMetaData(ctx context.Context, denomMetaData banktypes.Metadata)
	GetDenomMetaData(ctx context.Context, denom string) (banktypes.Metadata, bool)
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	GetSupply(ctx context.Context, denom string) sdk.Coin
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins

	// Methods imported from bank should be defined here
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

// DenomFeeRatesProvider reads a factory denom's DenomConfig. Implemented by the tokenfactory keeper.
type DenomFeeRatesProvider interface {
	// GetDenomCommunityPool returns the denom's community pool address, false when it has none.
	GetDenomCommunityPool(ctx sdk.Context, denom string) (sdk.AccAddress, bool)
	// GetDenomFeeRates returns the denom's transfer fee and gas surcharge ratios, zeros when none are set.
	GetDenomFeeRates(ctx sdk.Context, denom string) (transferFee, gasSurcharge math.LegacyDec)
}
