package bzeutils

import (
	"fmt"

	"cosmossdk.io/math"
)

// ParseCanonicalInt parses an integer that must be written in canonical base-10 form: decimal digits
// only, an optional leading "-" for negative values, no "+" sign, no base prefix, no separators, no
// spaces and no leading zeros ("0" itself is fine). Only the form produced by math.Int.String() is
// accepted. Callers check the sign and range they need.
func ParseCanonicalInt(s string) (math.Int, error) {
	i, ok := math.NewIntFromString(s)
	if !ok {
		return math.Int{}, fmt.Errorf("could not convert %q to an integer", s)
	}

	if i.String() != s {
		return math.Int{}, fmt.Errorf("%q must be a canonical base-10 integer", s)
	}

	return i, nil
}

// ParseCanonicalAmount parses a positive amount written in canonical base-10 form (see ParseCanonicalInt).
func ParseCanonicalAmount(s string) (math.Int, error) {
	amt, err := ParseCanonicalInt(s)
	if err != nil {
		return math.Int{}, err
	}

	if !amt.IsPositive() {
		return math.Int{}, fmt.Errorf("amount %q should be greater than 0", s)
	}

	return amt, nil
}
