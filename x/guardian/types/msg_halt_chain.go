package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MaxHaltReasonLength is the longest MsgHaltChain reason, in bytes.
const MaxHaltReasonLength = 256

// HaltPlanPrefix starts the name of every halt plan; the height follows it.
const HaltPlanPrefix = "emergency-halt-"

var _ sdk.Msg = &MsgHaltChain{}

// HaltPlanName is the name of the upgrade plan that halts the chain at height.
// A binary that restarts the chain registers an upgrade handler under it.
func HaltPlanName(height int64) string {
	return fmt.Sprintf("%s%d", HaltPlanPrefix, height)
}

// ValidateBasic does a sanity check on the provided data, see MsgBlockDenom.
func (m *MsgHaltChain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}

	if m.Reason == "" {
		return errorsmod.Wrap(ErrInvalidHaltReason, "reason is required")
	}

	if len(m.Reason) > MaxHaltReasonLength {
		return errorsmod.Wrapf(ErrInvalidHaltReason, "reason is %d bytes, at most %d", len(m.Reason), MaxHaltReasonLength)
	}

	return nil
}
