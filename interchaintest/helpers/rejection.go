package helpers

import (
	"errors"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TxFee returns the fee a transaction paid in denom, read from the "fee"
// attribute of its "tx" events. It is an error when the tx has no fee attribute.
func TxFee(res TxResult, denom string) (sdkmath.Int, error) {
	for _, e := range TxEvents(res, "tx") {
		v, ok := e.Attributes["fee"]
		if !ok {
			continue
		}
		coins, err := sdk.ParseCoinsNormalized(v)
		if err != nil {
			return sdkmath.Int{}, fmt.Errorf("parse fee %q: %w", v, err)
		}
		return coins.AmountOf(denom), nil
	}
	return sdkmath.Int{}, fmt.Errorf("tx %s has no fee attribute", res.TxHash)
}

// RejectionReason returns the chain's message for a rejected transaction: the
// simulation output when `--gas auto` simulation failed, or the raw log of a
// tx that came back with a non-zero code. It is an error when the tx was
// accepted (code 0) or when err is an infrastructure failure, so suites can
// assert rejections with one call whichever way the chain refused the tx.
func RejectionReason(res TxResult, err error) (string, error) {
	var simErr *SimulationError
	switch {
	case errors.As(err, &simErr):
		return simErr.Output, nil
	case err != nil:
		return "", err
	case res.Code != 0:
		return res.RawLog, nil
	default:
		return "", fmt.Errorf("tx %s was accepted at height %d, want a rejection", res.TxHash, res.Height)
	}
}
