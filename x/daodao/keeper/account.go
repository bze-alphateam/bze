package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// HasAccount proxies AccountKeeper.HasAccount. Exposed for genesis import.
func (k Keeper) HasAccount(ctx context.Context, addr sdk.AccAddress) bool {
	return k.accountKeeper.HasAccount(ctx, addr)
}

// NewAccountWithAddress proxies AccountKeeper.NewAccountWithAddress.
// Exposed for genesis import.
func (k Keeper) NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return k.accountKeeper.NewAccountWithAddress(ctx, addr)
}

// SetAccount proxies AccountKeeper.SetAccount. Exposed for genesis import.
func (k Keeper) SetAccount(ctx context.Context, acc sdk.AccountI) {
	k.accountKeeper.SetAccount(ctx, acc)
}
