package keeper

import (
	"bytes"

	"github.com/bze-alphateam/bze/x/txfeecollector/ante"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CtxFeePayerKey mirrors the txfeecollector ante.FeePayerKey constant.
const CtxFeePayerKey = ante.FeePayerKey

// getPayerCtxDenom returns the preferred fee denom for payer: the tx fee denom when payer is the account paying the
// tx fee, otherwise "" (the fee is charged in the native denom).
func (k Keeper) getPayerCtxDenom(ctx sdk.Context, payer sdk.AccAddress) string {
	txFeePayer, ok := ctx.Value(CtxFeePayerKey).(sdk.AccAddress)
	if !ok || txFeePayer.Empty() || !bytes.Equal(txFeePayer, payer) {
		return ""
	}

	return k.getCtxDenom(ctx)
}
