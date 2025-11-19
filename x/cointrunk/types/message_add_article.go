package types

import (
	"errors"
	"fmt"
	"net/url"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgAddArticle{}

func NewMsgAddArticle(publisher, title, url string) *MsgAddArticle {
	return &MsgAddArticle{
		Publisher: publisher,
		Title:     title,
		Url:       url,
	}
}

func (msg *MsgAddArticle) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Publisher)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid publisher address (%s)", err)
	}

	if len(msg.Title) < titleMinLength || len(msg.Title) > titleMaxLength {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, fmt.Sprintf("invalid title: expecting between %d and %d characters", titleMinLength, titleMaxLength))
	}

	_, err = msg.ParseUrl(msg.Url)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid url provided (%s)", err)
	}
	if len(msg.Url) > urlMaxLength {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, fmt.Sprintf("invalid url: provided url exceeds %d characters", urlMaxLength))
	}

	//validate picture only if it's provided
	if msg.Picture == "" {
		return nil
	}

	_, err = msg.ParseUrl(msg.Picture)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid picture url provided (%s)", err)
	}

	if len(msg.Picture) > urlMaxLength {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, fmt.Sprintf("invalid url: provided url exceeds %d characters", urlMaxLength))
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
