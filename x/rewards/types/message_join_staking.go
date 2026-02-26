package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgJoinStaking{}

func NewMsgJoinStaking(creator string, rewardId string, amount math.Int) *MsgJoinStaking {
	return &MsgJoinStaking{
		Creator:  creator,
		RewardId: rewardId,
		Amount:   amount,
	}
}

func (msg *MsgJoinStaking) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.RewardId == "" {
		return errorsmod.Wrapf(ErrInvalidRewardId, "empty reward id")
	}

	if !msg.Amount.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount should be greater than 0")
	}

	return nil
}
