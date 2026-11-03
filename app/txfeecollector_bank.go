package app

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// txfeecollectorBankKeeper is the bank keeper the txfeecollector module receives: the SDK bank keeper plus the bounded
// page read of an account's balances its EndBlock passes use.
type txfeecollectorBankKeeper struct {
	bankkeeper.BaseKeeper
}

// ProvideTxfeecollectorBankKeeper wraps the SDK bank keeper for the txfeecollector module. It returns the interface
// type so depinject never picks it for another module's bank keeper input.
func ProvideTxfeecollectorBankKeeper(bk bankkeeper.BaseKeeper) txfeecollectortypes.BankKeeper {
	return txfeecollectorBankKeeper{BaseKeeper: bk}
}

// GetBalancesPage reads at most limit balances of addr whose denom sorts strictly after afterDenom, straight from the
// bank store's balances map. The iterator is closed before it returns, so the caller can write to the store.
func (k txfeecollectorBankKeeper) GetBalancesPage(ctx context.Context, addr sdk.AccAddress, afterDenom string, limit int) (sdk.Coins, string) {
	if limit <= 0 {
		return nil, ""
	}

	rng := collections.NewPrefixedPairRange[sdk.AccAddress, string](addr)
	if afterDenom != "" {
		rng = rng.StartExclusive(afterDenom)
	}

	var page sdk.Coins
	err := k.Balances.Walk(ctx, rng, func(key collections.Pair[sdk.AccAddress, string], amount math.Int) (bool, error) {
		page = append(page, sdk.NewCoin(key.K2(), amount))

		return len(page) >= limit, nil
	})
	if err != nil {
		panic(err)
	}

	if len(page) == 0 {
		return nil, ""
	}

	return page, page[len(page)-1].Denom
}
