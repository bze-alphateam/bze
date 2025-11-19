package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgJoinStaking(t *testing.T) {
	creator := sample.AccAddress()
	rewardId := "reward123"
	amount := "1000"

	msg := NewMsgJoinStaking(creator, rewardId, amount)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, rewardId, msg.RewardId)
	require.Equal(t, amount, msg.Amount)
}

func TestMsgJoinStaking_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validRewardId := "reward123"
	validAmount := "1000"

	tests := []struct {
		name string
		msg  MsgJoinStaking
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgJoinStaking{
				Creator:  "invalid_address",
				RewardId: validRewardId,
				Amount:   validAmount,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgJoinStaking{
				Creator:  "",
				RewardId: validRewardId,
				Amount:   validAmount,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty reward id",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: "",
				Amount:   validAmount,
			},
			err: ErrInvalidRewardId,
		},
		{
			name: "empty amount",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: validRewardId,
				Amount:   "",
			},
			err: ErrInvalidAmount,
		},
		{
			name: "invalid amount - not a number",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: validRewardId,
				Amount:   "invalid",
			},
			err: ErrInvalidAmount,
		},
		{
			name: "invalid amount - negative",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: validRewardId,
				Amount:   "-100",
			},
			err: ErrInvalidAmount,
		},
		{
			name: "invalid amount - zero",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: validRewardId,
				Amount:   "0",
			},
			err: ErrInvalidAmount,
		},
		{
			name: "valid message - typical values",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: validRewardId,
				Amount:   validAmount,
			},
		},
		{
			name: "valid message - minimum amount",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: validRewardId,
				Amount:   "1",
			},
		},
		{
			name: "valid message - large amount",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: validRewardId,
				Amount:   "999999999999999",
			},
		},
		{
			name: "valid message - numeric reward id",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: "12345",
				Amount:   validAmount,
			},
		},
		{
			name: "valid message - alphanumeric reward id",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: "reward-123_abc",
				Amount:   validAmount,
			},
		},
		{
			name: "valid message - single character reward id",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: "a",
				Amount:   validAmount,
			},
		},
		{
			name: "valid message - long reward id",
			msg: MsgJoinStaking{
				Creator:  validCreator,
				RewardId: "very-long-reward-id-with-many-characters",
				Amount:   validAmount,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
