package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
	"github.com/bze-alphateam/bze/testutil/sample"
)

func TestMsgCreateTradingReward_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgCreateTradingReward
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgCreateTradingReward{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgCreateTradingReward{
				Creator: sample.AccAddress(),
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

func TestMsgUpdateTradingReward_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUpdateTradingReward
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgUpdateTradingReward{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgUpdateTradingReward{
				Creator: sample.AccAddress(),
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

func TestMsgDeleteTradingReward_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgDeleteTradingReward
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgDeleteTradingReward{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgDeleteTradingReward{
				Creator: sample.AccAddress(),
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
