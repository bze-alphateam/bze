package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/bzeutils"
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

	// amounts must be canonical base-10 integers
	amtInt, err := bzeutils.ParseCanonicalInt(msg.Amount)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidAmount, "could not convert amount: %s", err)
	}

	if !amtInt.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount should be greater than 0")
	}

	return nil
}
