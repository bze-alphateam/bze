package v2types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgUpdateStakingReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.Duration == 0 || msg.Duration > HundredYearsInDays {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "duration should be between 1 and %d days", HundredYearsInDays)
	}

	return nil
}
