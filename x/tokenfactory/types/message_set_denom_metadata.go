package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

var _ sdk.Msg = &MsgSetDenomMetadata{}

func NewMsgSetDenomMetadata(creator string, metadata types.Metadata) *MsgSetDenomMetadata {
	return &MsgSetDenomMetadata{
		Creator:  creator,
		Metadata: metadata,
	}
}

func (msg *MsgSetDenomMetadata) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
