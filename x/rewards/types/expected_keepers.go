package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	HasSupply(ctx context.Context, denom string) bool

	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type TradingKeeper interface {
	MarketExists(ctx sdk.Context, marketId string) bool
	CaptureAndSwapUserFee(ctx sdk.Context, payer sdk.AccAddress, fee sdk.Coins, toModule string) (coins sdk.Coins, err error)
}

type EpochKeeper interface {
	SafeGetEpochCountByIdentifier(ctx sdk.Context, identifier string) (int64, error)
}
