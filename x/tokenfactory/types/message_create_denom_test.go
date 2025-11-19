package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgCreateDenom(t *testing.T) {
	creator := sample.AccAddress()
	subdenom := "mytoken"

	msg := NewMsgCreateDenom(creator, subdenom)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, subdenom, msg.Subdenom)
}

func TestMsgCreateDenom_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validSubdenom := "mytoken"

	tests := []struct {
		name string
		msg  MsgCreateDenom
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgCreateDenom{
				Creator:  "invalid_address",
				Subdenom: validSubdenom,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgCreateDenom{
				Creator:  "",
				Subdenom: validSubdenom,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty subdenom",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "subdenom contains underscore - single",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "my_token",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "subdenom contains underscore - multiple",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "my_test_token",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "subdenom contains underscore - at start",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "_mytoken",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "subdenom contains underscore - at end",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "mytoken_",
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid message - typical subdenom",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: validSubdenom,
			},
		},
		{
			name: "valid message - single character",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "a",
			},
		},
		{
			name: "valid message - with numbers",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "token123",
			},
		},
		{
			name: "valid message - with hyphens",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "my-token",
			},
		},
		{
			name: "valid message - mixed alphanumeric and hyphens",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "test-token-123",
			},
		},
		{
			name: "valid message - long subdenom",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "verylongsubdenominationname",
			},
		},
		{
			name: "valid message - uppercase letters",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "MyToken",
			},
		},
		{
			name: "valid message - mixed case",
			msg: MsgCreateDenom{
				Creator:  validCreator,
				Subdenom: "MyTestToken123",
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
