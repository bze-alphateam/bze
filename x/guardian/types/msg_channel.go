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

// ValidateChannelID accepts an IBC channel identifier in the canonical form
// ibc-go generates (channel-<n>, no leading zeros): any other spelling, such as
// channel-07, can never name a real channel.
func ValidateChannelID(channelID string) error {
	sequence, err := channeltypes.ParseChannelSequence(channelID)
	if err != nil {
		return errorsmod.Wrap(ErrInvalidChannel, err.Error())
	}

	if channelID != channeltypes.FormatChannelIdentifier(sequence) {
		return errorsmod.Wrapf(ErrInvalidChannel, "%s is not canonical, use %s", channelID, channeltypes.FormatChannelIdentifier(sequence))
	}

	return nil
}
