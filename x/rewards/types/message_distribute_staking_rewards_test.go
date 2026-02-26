package types

import (
	"cosmossdk.io/math"
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgDistributeStakingRewards_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	tests := []struct {
		name string
		msg  MsgDistributeStakingRewards
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgDistributeStakingRewards{
				Creator: "invalid_address",
				Amount:  math.NewInt(100),
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid amount - negative",
			msg: MsgDistributeStakingRewards{
				Creator: validCreator,
				Amount:  math.NewInt(-100),
			},
			err: ErrInvalidAmount,
		},
		{
			name: "invalid amount - zero",
			msg: MsgDistributeStakingRewards{
				Creator: validCreator,
				Amount:  math.ZeroInt(),
			},
			err: ErrInvalidAmount,
		},
		{
			name: "valid message",
			msg: MsgDistributeStakingRewards{
				Creator: validCreator,
				Amount:  math.NewInt(100),
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
