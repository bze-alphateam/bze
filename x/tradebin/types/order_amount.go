package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/bzeutils"
)

// ParseOrderAmount parses the amount of an order message. The amount must be a positive integer in
// canonical base-10 form.
func ParseOrderAmount(amount string) (math.Int, error) {
	amt, err := bzeutils.ParseCanonicalAmount(amount)
	if err != nil {
		return math.Int{}, errorsmod.Wrap(ErrInvalidOrderAmount, err.Error())
	}

	return amt, nil
}
