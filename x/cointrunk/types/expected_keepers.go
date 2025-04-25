package types

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type DistrKeeper interface {
	FundCommunityPool(ctx context.Context, amount sdk.Coins, sender sdk.AccAddress) error
}
