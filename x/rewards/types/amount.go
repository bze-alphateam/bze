package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// ParseCanonicalInt parses an integer that must be written in canonical base-10 form: decimal digits
// only, an optional leading "-" for negative values, no "+" sign, no base prefix, no separators and no
// leading zeros ("0" itself is fine). Callers check the sign they need.
//
// Amounts handled by this module are stored and re-read as strings, so only the form produced by
// math.Int.String() is accepted: every later parse of the same string then yields the same value.
func ParseCanonicalInt(s string) (math.Int, error) {
	amt, ok := math.NewIntFromString(s)
	if !ok {
		return math.Int{}, fmt.Errorf("could not convert %q to an integer", s)
	}

	if amt.String() != s {
		return math.Int{}, fmt.Errorf("amount %q must be a canonical base-10 integer", s)
	}

	return amt, nil
}

// ParseCanonicalAmount parses a positive amount written in canonical base-10 form (see ParseCanonicalInt).
func ParseCanonicalAmount(s string) (math.Int, error) {
	amt, err := ParseCanonicalInt(s)
	if err != nil {
		return math.Int{}, err
	}

	if !amt.IsPositive() {
		return math.Int{}, fmt.Errorf("amount should be greater than 0")
	}

	return amt, nil
}
