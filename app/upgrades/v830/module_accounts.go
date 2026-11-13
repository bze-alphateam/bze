package v830

import (
	"context"
	"fmt"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	daotypes "github.com/bze-alphateam/bze/x/dao/types"
)

// AccountKeeper is the part of the auth keeper the handler uses.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	NewAccount(ctx context.Context, acc sdk.AccountI) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
}

// newModuleAccount is a module account added in v8.3.0, with the permissions the app registers for it.
type newModuleAccount struct {
	name        string
	permissions []string
}

// newModuleAccounts lists the module accounts that are new in v8.3.0. x/guardian and ratelimit have none.
var newModuleAccounts = []newModuleAccount{
	{name: daotypes.ModuleName},
}

// ensureNewModuleAccounts makes every module account that is new in v8.3.0 exist as a module account. The
// addresses were not blocked before v8.3.0, so an account may already sit at one:
//   - no account: the module account is created with a new account number;
//   - a module account with the expected name and permissions: left as it is;
//   - a module account with another name or other permissions: an error;
//   - a plain BaseAccount: replaced by the module account, keeping its address, account number and sequence;
//     its balances, held by x/bank, are untouched;
//   - any other account type: left in place and logged. An error here would abort the upgrade and stop the chain.
//
// It never asks the account keeper for the module account itself, which panics on a plain account.
func ensureNewModuleAccounts(ctx sdk.Context, ak AccountKeeper) error {
	for _, macc := range newModuleAccounts {
		if err := ensureModuleAccount(ctx, ak, macc.name, macc.permissions); err != nil {
			return err
		}
	}

	return nil
}

func ensureModuleAccount(ctx sdk.Context, ak AccountKeeper, name string, permissions []string) error {
	addr := authtypes.NewModuleAddress(name)
	acc := ak.GetAccount(ctx, addr)
	if acc == nil {
		ak.SetAccount(ctx, ak.NewAccount(ctx, authtypes.NewEmptyModuleAccount(name, permissions...)))
		ctx.Logger().Info("created module account", "name", name, "address", addr.String())

		return nil
	}

	switch existing := acc.(type) {
	case sdk.ModuleAccountI:
		if existing.GetName() != name || !samePermissions(existing.GetPermissions(), permissions) {
			return fmt.Errorf(
				"module account at %s is %q with permissions %v, expected %q with permissions %v",
				addr, existing.GetName(), existing.GetPermissions(), name, permissions,
			)
		}

		return nil
	case *authtypes.BaseAccount:
		base := authtypes.NewBaseAccount(addr, nil, existing.GetAccountNumber(), existing.GetSequence())
		macc := authtypes.NewModuleAccount(base, name, permissions...)
		if err := macc.Validate(); err != nil {
			return fmt.Errorf("module account %s: %w", name, err)
		}
		ak.SetAccount(ctx, macc)
		ctx.Logger().Info(
			"replaced plain account with module account",
			"name", name, "address", addr.String(), "account_number", existing.GetAccountNumber(),
		)

		return nil
	default:
		ctx.Logger().Error(
			"unsupported account type at module address, left unchanged",
			"name", name, "address", addr.String(), "type", fmt.Sprintf("%T", acc),
		)

		return nil
	}
}

// samePermissions compares two permission lists regardless of order; nil and empty are equal.
func samePermissions(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)

	return slices.Equal(x, y)
}
