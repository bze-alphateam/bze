package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

var (
	_ sdk.Msg = &MsgCloseChannel{}
	_ sdk.Msg = &MsgReopenChannel{}
)

// ValidateBasic does a sanity check on the provided data. The keeper runs it
// too: a message executed by x/gov or x/authz does not pass through the
// transaction decoder.
func (m *MsgCloseChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}

	return ValidateChannelID(m.ChannelId)
}

// ValidateBasic does a sanity check on the provided data, see MsgCloseChannel.
func (m *MsgReopenChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return ValidateChannelID(m.ChannelId)
}

// ValidateChannelID accepts an IBC channel identifier (channel-<n>).
func ValidateChannelID(channelID string) error {
	if _, err := channeltypes.ParseChannelSequence(channelID); err != nil {
		return errorsmod.Wrap(ErrInvalidChannel, err.Error())
	}

	return nil
}
