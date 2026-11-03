package v830

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
)

// removedBurnerFeeCollectorName is the module account name of txfeecollector_burner, removed in v8.3.0: burn-bound
// fees go straight to the burner module. Its ModuleAccount stays in auth state, unused and no longer blocked.
const removedBurnerFeeCollectorName = "txfeecollector_burner"

// BankKeeper is the part of the bank keeper the handler uses.
type BankKeeper interface {
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}

// sweepBurnerFeeCollector moves every coin left at the removed txfeecollector_burner address to the burner module,
// which handles each kind its own way on its next run. The name is no longer a registered module account, so the
// coins leave as a plain account send.
func sweepBurnerFeeCollector(ctx sdk.Context, bank BankKeeper) error {
	addr := authtypes.NewModuleAddress(removedBurnerFeeCollectorName)
	coins := bank.GetAllBalances(ctx, addr)
	if coins.IsZero() {
		ctx.Logger().Info("txfeecollector_burner holds nothing to sweep")
		return nil
	}

	if err := bank.SendCoinsFromAccountToModule(ctx, addr, burnertypes.ModuleName, coins); err != nil {
		return err
	}
	ctx.Logger().Info("swept txfeecollector_burner to the burner module", "coins", coins.String())

	return nil
}
