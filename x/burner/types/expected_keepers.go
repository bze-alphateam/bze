package types

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EpochKeeper defines expected keeper that can return the current epoch
type EpochKeeper interface {
	GetEpochCountByIdentifier(ctx sdk.Context, identifier string) int64
}

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	HasAccount(ctx context.Context, addr sdk.AccAddress) bool
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	HasSupply(ctx context.Context, denom string) bool
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}
