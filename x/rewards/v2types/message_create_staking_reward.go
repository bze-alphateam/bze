package v2types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TenYearsInDays     = 365 * 10
	HundredYearsInDays = 365 * 100
)

func (msg *MsgCreateStakingReward) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if !msg.PrizeAmount.IsPositive() {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "amount should be greater than 0")
	}

	if msg.PrizeDenom == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "prize denom cannot be empty")
	}

	if msg.StakingDenom == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "staking denom cannot be empty")
	}

	if msg.MinStake.IsNegative() {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "min stake cannot be negative")
	}

	if msg.Duration == 0 || msg.Duration > HundredYearsInDays {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "duration should be between 1 and %d days", HundredYearsInDays)
	}

	if msg.Lock > TenYearsInDays {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "locking time should be between 0 and %d days", TenYearsInDays)
	}

	return nil
}
