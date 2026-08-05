package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgUpdateBoost(t *testing.T) {
	creator := sample.AccAddress()

	msg := NewMsgUpdateBoost(creator, "000000000001", "000000000002", "30")

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, "000000000001", msg.RewardId)
	require.Equal(t, "000000000002", msg.BoostId)
	require.Equal(t, "30", msg.Days)
}

func TestMsgUpdateBoost_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()

	tests := []struct {
		name string
		msg  MsgUpdateBoost
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgUpdateBoost{
				Creator:  "invalid_address",
				RewardId: "000000000001",
				BoostId:  "000000000002",
				Days:     "30",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty reward_id",
			msg: MsgUpdateBoost{
				Creator:  validCreator,
				RewardId: "",
				BoostId:  "000000000002",
				Days:     "30",
			},
			err: ErrInvalidRewardId,
		},
		{
			name: "empty boost_id",
			msg: MsgUpdateBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				BoostId:  "",
				Days:     "30",
			},
			err: ErrInvalidBoostId,
		},
		{
			name: "non-numeric days",
			msg: MsgUpdateBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				BoostId:  "000000000002",
				Days:     "xyz",
			},
			err: ErrInvalidBoostDays,
		},
		{
			name: "zero days",
			msg: MsgUpdateBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				BoostId:  "000000000002",
				Days:     "0",
			},
			err: ErrInvalidBoostDays,
		},
		{
			name: "negative days",
			msg: MsgUpdateBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				BoostId:  "000000000002",
				Days:     "-3",
			},
			err: ErrInvalidBoostDays,
		},
		{
			name: "days too large",
			msg: MsgUpdateBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				BoostId:  "000000000002",
				Days:     "36501",
			},
			err: ErrInvalidBoostDays,
		},
		{
			name: "valid",
			msg: MsgUpdateBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				BoostId:  "000000000002",
				Days:     "30",
			},
		},
		{
			name: "valid max days",
			msg: MsgUpdateBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				BoostId:  "000000000002",
				Days:     "36500",
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
