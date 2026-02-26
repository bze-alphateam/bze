package v2types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (msg *MsgDistributeStakingRewards) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.RewardId == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "empty reward id")
	}

	if !msg.Amount.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "amount should be greater than 0")
	}

	return nil
}
