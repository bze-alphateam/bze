package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgUpdateStakingReward(t *testing.T) {
	creator := sample.AccAddress()
	rewardId := "reward123"
	duration := "30"

	msg := NewMsgUpdateStakingReward(creator, rewardId, duration)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, rewardId, msg.RewardId)
	require.Equal(t, duration, msg.Duration)
}

func TestMsgUpdateStakingReward_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validRewardId := "reward123"
	validDuration := "30"

	tests := []struct {
		name string
		msg  MsgUpdateStakingReward
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgUpdateStakingReward{
				Creator:  "invalid_address",
				RewardId: validRewardId,
				Duration: validDuration,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgUpdateStakingReward{
				Creator:  "",
				RewardId: validRewardId,
				Duration: validDuration,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid duration - empty",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: validRewardId,
				Duration: "",
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - not a number",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: validRewardId,
				Duration: "invalid",
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - negative",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: validRewardId,
				Duration: "-10",
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - zero",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: validRewardId,
				Duration: "0",
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - decimal",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: validRewardId,
				Duration: "10.5",
			},
			err: ErrInvalidDuration,
		},
		{
			name: "valid message - typical values",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: validRewardId,
				Duration: validDuration,
			},
		},
		{
			name: "valid message - minimum duration",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: validRewardId,
				Duration: "1",
			},
		},
		{
			name: "invalid duration - above maximum",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: validRewardId,
				Duration: "36501",
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - max uint32",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: validRewardId,
				Duration: "4294967295",
			},
			err: ErrInvalidDuration,
		},
		{
			name: "valid message - maximum duration",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: validRewardId,
				Duration: "36500",
			},
		},
		{
			name: "valid message - empty reward id",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: "",
				Duration: validDuration,
			},
		},
		{
			name: "valid message - numeric reward id",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: "12345",
				Duration: validDuration,
			},
		},
		{
			name: "valid message - alphanumeric reward id",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: "reward-123_abc",
				Duration: validDuration,
			},
		},
		{
			name: "valid message - long reward id",
			msg: MsgUpdateStakingReward{
				Creator:  validCreator,
				RewardId: "very-long-reward-id-with-many-characters",
				Duration: validDuration,
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
