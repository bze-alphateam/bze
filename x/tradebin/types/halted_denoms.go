package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MaxDenomsPerHaltMsg bounds the denoms one MsgHaltDenoms / MsgUnhaltDenoms carries. It only keeps a
// single proposal message reasonable in size: the number of halted denoms in store is unbounded, since
// the halt check is one store lookup whatever the count.
const MaxDenomsPerHaltMsg = 1000

// ValidateHaltedDenoms checks a list of denoms as it appears in a halt / unhalt message or in genesis:
// at most max entries (0 = no cap), every entry a valid denom, no duplicates. When nativeDenom is not
// empty it is refused as well: halting it would halt every market and pool.
func ValidateHaltedDenoms(denoms []string, max int, nativeDenom string) error {
	if max > 0 && len(denoms) > max {
		return fmt.Errorf("at most %d denoms per message, got %d", max, len(denoms))
	}

	seen := make(map[string]struct{}, len(denoms))
	for _, denom := range denoms {
		if err := sdk.ValidateDenom(denom); err != nil {
			return fmt.Errorf("invalid denom %q: %w", denom, err)
		}

		if nativeDenom != "" && denom == nativeDenom {
			return fmt.Errorf("the native denom %s cannot be halted", denom)
		}

		if _, duplicate := seen[denom]; duplicate {
			return fmt.Errorf("duplicate denom %s", denom)
		}
		seen[denom] = struct{}{}
	}

	return nil
}
