package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgChangeAdmin(t *testing.T) {
	creator := sample.AccAddress()
	denom := "utoken"
	newAdmin := sample.AccAddress()

	msg := NewMsgChangeAdmin(creator, denom, newAdmin)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, denom, msg.Denom)
	require.Equal(t, newAdmin, msg.NewAdmin)
}

func TestMsgChangeAdmin_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validNewAdmin := sample.AccAddress()
	validDenom := "utoken"

	tests := []struct {
		name string
		msg  MsgChangeAdmin
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgChangeAdmin{
				Creator:  "invalid_address",
				Denom:    validDenom,
				NewAdmin: validNewAdmin,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgChangeAdmin{
				Creator:  "",
				Denom:    validDenom,
				NewAdmin: validNewAdmin,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid new admin address",
			msg: MsgChangeAdmin{
				Creator:  validCreator,
				Denom:    validDenom,
				NewAdmin: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty new admin address",
			msg: MsgChangeAdmin{
				Creator:  validCreator,
				Denom:    validDenom,
				NewAdmin: "",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "creator and new admin are the same",
			msg: MsgChangeAdmin{
				Creator:  validCreator,
				Denom:    validDenom,
				NewAdmin: validCreator,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty denom",
			msg: MsgChangeAdmin{
				Creator:  validCreator,
				Denom:    "",
				NewAdmin: validNewAdmin,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid message - typical values",
			msg: MsgChangeAdmin{
				Creator:  validCreator,
				Denom:    validDenom,
				NewAdmin: validNewAdmin,
			},
		},
		{
			name: "valid message - single character denom",
			msg: MsgChangeAdmin{
				Creator:  validCreator,
				Denom:    "u",
				NewAdmin: validNewAdmin,
			},
		},
		{
			name: "valid message - long denom",
			msg: MsgChangeAdmin{
				Creator:  validCreator,
				Denom:    "verylongdenominationname",
				NewAdmin: validNewAdmin,
			},
		},
		{
			name: "valid message - denom with numbers",
			msg: MsgChangeAdmin{
				Creator:  validCreator,
				Denom:    "token123",
				NewAdmin: validNewAdmin,
			},
		},
		{
			name: "valid message - denom with special chars",
			msg: MsgChangeAdmin{
				Creator:  validCreator,
				Denom:    "my-token_v2",
				NewAdmin: validNewAdmin,
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
