package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgPauseAutomation{}
	_ sdk.Msg = &MsgResumeAutomation{}
)

// ValidateBasic does a sanity check on the provided data. The keeper runs it
// too: a message executed by x/gov or x/authz does not pass through the
// transaction decoder.
func (m *MsgPauseAutomation) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}

	return nil
}

// ValidateBasic does a sanity check on the provided data, see MsgPauseAutomation.
func (m *MsgResumeAutomation) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return nil
}
