package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddArticle = "add_article"

var _ sdk.Msg = &MsgAddArticle{}

func NewMsgAddArticle(publisher string, title string, url string, picture string) *MsgAddArticle {
	return &MsgAddArticle{
		Publisher: publisher,
		Title:     title,
		Url:       url,
		Picture:   picture,
	}
}

func (msg *MsgAddArticle) Route() string {
	return RouterKey
}

func (msg *MsgAddArticle) Type() string {
	return TypeMsgAddArticle
}

func (msg *MsgAddArticle) GetSigners() []sdk.AccAddress {
	publisher, err := sdk.AccAddressFromBech32(msg.Publisher)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{publisher}
}

func (msg *MsgAddArticle) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddArticle) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Publisher)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid publisher address (%s)", err)
	}
	return nil
}
