package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EpochKeeper defines expected keeper that can return the current epoch
type EpochKeeper interface {
	SafeGetEpochCountByIdentifier(ctx sdk.Context, identifier string) (int64, error)
}

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	HasAccount(ctx context.Context, addr sdk.AccAddress) bool
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	// GetBalancesPage returns at most limit balances of addr, the ones whose denom sorts strictly after afterDenom
	// (all of them from the first when afterDenom is empty), in the bank store's denom order, and the denom of the last
	// one ("" when none). It reads the store with a bounded iterator, never the whole account.
	GetBalancesPage(ctx context.Context, addr sdk.AccAddress, afterDenom string, limit int) (sdk.Coins, string)
	BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	HasSupply(ctx context.Context, denom string) bool
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
}

type TradeKeeper interface {
	IsNativeDenom(ctx sdk.Context, denom string) bool
	CanSwapForNativeDenom(ctx sdk.Context, coin sdk.Coin) bool
	SafeModuleAddLiquidityWithNativeDenom(ctx sdk.Context, fromModule string, coins sdk.Coins) (addedCoins sdk.Coins, refundedCoins sdk.Coins, err error)
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

// GuardianKeeper tells whether the guardian paused the protocol automation. Implemented by the x/guardian keeper.
type GuardianKeeper interface {
	// IsAutomationPaused is true while the periodic burn must not burn, lock or pool anything.
	IsAutomationPaused(ctx context.Context) bool
}
