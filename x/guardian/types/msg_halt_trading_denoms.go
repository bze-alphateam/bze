package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
)

var _ sdk.Msg = &MsgHaltTradingDenoms{}

// ValidateBasic does a sanity check on the provided data, with the same denom
// rules as tradebin's MsgHaltDenoms: a non-empty list of valid denoms, no
// duplicates, at most tradebin's MaxDenomsPerHaltMsg. The native denom check
// needs tradebin's params and lives in the keeper. The keeper runs it too: a
// message executed by x/gov or x/authz does not pass through the transaction
// decoder.
func (m *MsgHaltTradingDenoms) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}

	if len(m.Denoms) == 0 {
		return errorsmod.Wrap(ErrInvalidDenom, "no denoms provided")
	}

	if err := tradebintypes.ValidateHaltedDenoms(m.Denoms, tradebintypes.MaxDenomsPerHaltMsg, ""); err != nil {
		return errorsmod.Wrap(ErrInvalidDenom, err.Error())
	}

	return nil
}
