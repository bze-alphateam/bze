package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgDistributeStakingRewards{}

func NewMsgDistributeStakingRewards(creator string, rewardId string, amount string) *MsgDistributeStakingRewards {
	return &MsgDistributeStakingRewards{
		Creator:  creator,
		RewardId: rewardId,
		Amount:   amount,
	}
}

func (msg *MsgDistributeStakingRewards) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	amtInt, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return errorsmod.Wrapf(ErrInvalidAmount, "could not convert amount")
	}

	if !amtInt.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount should be greater than 0")
	}

	return nil
}
