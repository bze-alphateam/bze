package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgCleanupBoost(t *testing.T) {
	creator := sample.AccAddress()

	msg := NewMsgCleanupBoost(creator, "000000000001", 100)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, "000000000001", msg.RewardId)
	require.Equal(t, uint32(100), msg.Limit)
}

func TestMsgCleanupBoost_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()

	tests := []struct {
		name string
		msg  MsgCleanupBoost
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgCleanupBoost{
				Creator:  "invalid_address",
				RewardId: "000000000001",
				Limit:    10,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty reward_id",
			msg: MsgCleanupBoost{
				Creator:  validCreator,
				RewardId: "",
				Limit:    10,
			},
			err: ErrInvalidRewardId,
		},
		{
			name: "valid with zero limit",
			msg: MsgCleanupBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				Limit:    0,
			},
		},
		{
			name: "valid with limit",
			msg: MsgCleanupBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				Limit:    100,
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
