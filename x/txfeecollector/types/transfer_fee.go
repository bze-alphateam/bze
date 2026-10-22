package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// FactoryDenomPrefix is the prefix of the tokenfactory denoms, the only ones that can carry a transfer fee or a gas
// surcharge.
const FactoryDenomPrefix = "factory/"

type skipTransferFeeKey struct{}

// WithSkipTransferFee returns a context on which the transfer-fee send restriction charges nothing. Use it only on a
// context local to the send that must not pay (the fee send itself, the tx fee deduction), never on a context passed
// on to other handlers.
func WithSkipTransferFee(ctx sdk.Context) sdk.Context {
	return ctx.WithValue(skipTransferFeeKey{}, true)
}

// SkipTransferFee reports whether the context was marked by WithSkipTransferFee.
func SkipTransferFee(ctx context.Context) bool {
	skip, _ := ctx.Value(skipTransferFeeKey{}).(bool)

	return skip
}
