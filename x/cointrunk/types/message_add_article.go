package types

import (
	"errors"
	"net/url"
	"strings"

	"github.com/bze-alphateam/bze/bzeutils"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddArticle = "add_article"

var _ sdk.Msg = &MsgAddArticle{}

func NewMsgAddArticle(publisher string, title string, url string, picture string) *MsgAddArticle {
	return &MsgAddArticle{
		Publisher: publisher,
		Title:     strings.TrimSpace(bzeutils.GetSanitizer().SanitizeHtml(title)),
		Url:       strings.TrimSpace(url),
		Picture:   strings.TrimSpace(picture),
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

	if len(msg.Title) < 10 || len(msg.Title) > 140 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid title: expecting between 10 and 140 characters")
	}

	_, err = msg.ParseUrl(msg.Url)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid url provided (%s)", err)
	}
	if len(msg.Url) > 2048 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid url: provided url exceeds 2048 characters")
	}

	//validate picture only if it's provided
	if msg.Picture == "" {
		return nil
	}

	_, err = msg.ParseUrl(msg.Picture)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid picture url provided (%s)", err)
	}

	if len(msg.Picture) > 2048 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid picture url: provided url exceeds 2048 chars")
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
