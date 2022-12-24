package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type GovKeeper interface {
	// Methods imported from gov should be defined here
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type DistrKeeper interface {
	FundCommunityPool(ctx sdk.Context, amount sdk.Coins, sender sdk.AccAddress) error
}
