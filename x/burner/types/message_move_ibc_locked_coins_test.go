package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgMoveIbcLockedCoins_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgMoveIbcLockedCoins
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgMoveIbcLockedCoins{
				Creator: "invalid_address",
				Denom:   "ibc/ABC123",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "missing denom",
			msg: MsgMoveIbcLockedCoins{
				Creator: sample.AccAddress(),
				Denom:   "",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "LP token denom rejected",
			msg: MsgMoveIbcLockedCoins{
				Creator: sample.AccAddress(),
				Denom:   "ulp_some_pool",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "non-IBC denom rejected",
			msg: MsgMoveIbcLockedCoins{
				Creator: sample.AccAddress(),
				Denom:   "ubze",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "factory denom rejected",
			msg: MsgMoveIbcLockedCoins{
				Creator: sample.AccAddress(),
				Denom:   "factory/bze1abc/token",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid IBC denom",
			msg: MsgMoveIbcLockedCoins{
				Creator: sample.AccAddress(),
				Denom:   "ibc/ABC123",
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
