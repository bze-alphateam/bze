package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
	"github.com/bze-alphateam/bze/testutil/sample"
)

func TestMsgCreateStakingReward_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgCreateStakingReward
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgCreateStakingReward{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgCreateStakingReward{
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

func TestMsgUpdateStakingReward_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUpdateStakingReward
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgUpdateStakingReward{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgUpdateStakingReward{
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

func TestMsgDeleteStakingReward_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgDeleteStakingReward
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgDeleteStakingReward{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgDeleteStakingReward{
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
