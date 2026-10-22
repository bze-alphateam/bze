package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSetDenomConfig{}

func NewMsgSetDenomConfig(creator, denom, communityPool string, transferFee, gasSurcharge math.LegacyDec) *MsgSetDenomConfig {
	return &MsgSetDenomConfig{
		Creator:       creator,
		Denom:         denom,
		CommunityPool: communityPool,
		TransferFee:   transferFee,
		GasSurcharge:  gasSurcharge,
	}
}

// Config returns the denom config carried by the message.
func (msg *MsgSetDenomConfig) Config() DenomConfig {
	return DenomConfig{
		CommunityPool: msg.CommunityPool,
		TransferFee:   msg.TransferFee,
		GasSurcharge:  msg.GasSurcharge,
	}
}

func (msg *MsgSetDenomConfig) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	_, _, err = DeconstructDenom(msg.Denom)
	if err != nil {
		return err
	}

	return msg.Config().Validate()
}
