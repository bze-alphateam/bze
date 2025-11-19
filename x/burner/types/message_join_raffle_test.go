package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgJoinRaffle(t *testing.T) {
	creator := sample.AccAddress()
	denom := "utoken"

	msg := NewMsgJoinRaffle(creator, denom)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, denom, msg.Denom)
}

func TestMsgJoinRaffle_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validDenom := "utoken"

	tests := []struct {
		name string
		msg  MsgJoinRaffle
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgJoinRaffle{
				Creator: "invalid_address",
				Denom:   validDenom,
				Tickets: 1,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgJoinRaffle{
				Creator: "",
				Denom:   validDenom,
				Tickets: 1,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty denom",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   "",
				Tickets: 1,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "zero tickets",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   validDenom,
				Tickets: 0,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "tickets exceeds maximum - 51 tickets",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   validDenom,
				Tickets: 51,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "tickets exceeds maximum - 100 tickets",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   validDenom,
				Tickets: 100,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid message - 1 ticket",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   validDenom,
				Tickets: 1,
			},
		},
		{
			name: "valid message - maximum tickets (50)",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   validDenom,
				Tickets: 50,
			},
		},
		{
			name: "valid message - multiple tickets",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   validDenom,
				Tickets: 25,
			},
		},
		{
			name: "valid message - different denom",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   "ustake",
				Tickets: 10,
			},
		},
		{
			name: "valid message - denom with numbers",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   "token123",
				Tickets: 5,
			},
		},
		{
			name: "valid message - denom with special chars",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   "my-token_v2",
				Tickets: 15,
			},
		},
		{
			name: "valid message - long denom",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   "verylongdenominationname",
				Tickets: 20,
			},
		},
		{
			name: "valid message - single character denom",
			msg: MsgJoinRaffle{
				Creator: validCreator,
				Denom:   "x",
				Tickets: 30,
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
