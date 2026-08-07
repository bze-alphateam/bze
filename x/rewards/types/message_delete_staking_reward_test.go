package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgDeleteStakingReward_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgDeleteStakingReward
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgDeleteStakingReward{
				Creator:  "invalid_address",
				RewardId: "000000000001",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "empty reward id",
			msg: MsgDeleteStakingReward{
				Creator: sample.AccAddress(),
			},
			err: ErrInvalidRewardId,
		}, {
			name: "valid message",
			msg: MsgDeleteStakingReward{
				Creator:  sample.AccAddress(),
				RewardId: "000000000001",
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
