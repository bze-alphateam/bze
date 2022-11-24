package types

import (
	"errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"net/url"
	"strings"
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

	if len(strings.Trim(msg.Title, " ")) < 10 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid title: expecting at least 10 characters")
	}

	_, err = msg.ParseUrl(msg.Url)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid url provided (%s)", err)
	}

	//validate picture only if it's provided
	if msg.Picture == "" {
		return nil
	}

	_, err = msg.ParseUrl(msg.Picture)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid picture url provided (%s)", err)
	}

	return nil
}

func (msg *MsgAddArticle) ParseUrl(uri string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "https" {
		return nil, errors.New("invalid url scheme: only https accepted")
	}

	return parsed, nil
}
