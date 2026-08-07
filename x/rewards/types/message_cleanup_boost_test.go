package types

import (
	"math"
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgCleanupBoost(t *testing.T) {
	creator := sample.AccAddress()

	msg := NewMsgCleanupBoost(creator, "000000000001", "000000000002", 50)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, "000000000001", msg.RewardId)
	require.Equal(t, "000000000002", msg.BoostId)
	require.Equal(t, uint32(50), msg.Limit)
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
				BoostId:  "000000000002",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty reward_id",
			msg: MsgCleanupBoost{
				Creator:  validCreator,
				RewardId: "",
				BoostId:  "000000000002",
			},
			err: ErrInvalidRewardId,
		},
		{
			name: "empty boost_id",
			msg: MsgCleanupBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				BoostId:  "",
			},
			err: ErrInvalidBoostId,
		},
		{
			name: "valid with zero limit (server uses the param default)",
			msg: MsgCleanupBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				BoostId:  "000000000002",
				Limit:    0,
			},
		},
		{
			name: "valid with huge limit (server clamps)",
			msg: MsgCleanupBoost{
				Creator:  validCreator,
				RewardId: "000000000001",
				BoostId:  "000000000002",
				Limit:    math.MaxUint32,
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
