package types

import (
	"regexp"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgAcceptDomain{}

func NewMsgAcceptDomain(authority string, domain string, active bool) *MsgAcceptDomain {
	return &MsgAcceptDomain{
		Authority: authority,
		Domain:    domain,
		Active:    active,
	}
}

func (msg *MsgAcceptDomain) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority (%s)", err)
	}

	RegExp := regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,12}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z
 ]{2,3})$`)

	isValidDomain := RegExp.MatchString(msg.Domain)
	if !isValidDomain {
		return errorsmod.Wrapf(ErrInvalidProposalContent, "proposal domain is invalid")
	}

	return nil
}
