package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// canonicalAddress decodes a bech32 account address and returns it together
// with its standard (lowercase) spelling. Bech32 is case-insensitive, so the
// same account can be spelled more than one way; every store key, stored
// address, event and hook argument in this module uses the standard spelling,
// so one account always maps to one record.
func canonicalAddress(address string) (sdk.AccAddress, string, error) {
	acc, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return nil, "", err
	}

	return acc, acc.String(), nil
}

// queryAddress returns the standard spelling of a queried address, so a query
// finds the account's records whichever spelling it was given. An address that
// does not decode is returned unchanged and simply matches nothing.
func queryAddress(address string) string {
	if _, canonical, err := canonicalAddress(address); err == nil {
		return canonical
	}

	return address
}
