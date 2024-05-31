package types

import (
	"encoding/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

const TypeMsgSetDenomMetadata = "set_denom_metadata"

var _ sdk.Msg = &MsgSetDenomMetadata{}

func NewMsgSetDenomMetadata(creator string, metadata string) *MsgSetDenomMetadata {
	mdByte := []byte(metadata)
	mdStruct := types.Metadata{}
	err := json.Unmarshal(mdByte, &mdStruct)
	if err != nil {
		return nil
	}

	return &MsgSetDenomMetadata{
		Creator:  creator,
		Metadata: mdStruct,
	}
}

func (msg *MsgSetDenomMetadata) Route() string {
	return RouterKey
}

func (msg *MsgSetDenomMetadata) Type() string {
	return TypeMsgSetDenomMetadata
}

func (msg *MsgSetDenomMetadata) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgSetDenomMetadata) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSetDenomMetadata) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
