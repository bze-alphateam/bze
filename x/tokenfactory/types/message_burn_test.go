package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgBurn(t *testing.T) {
	creator := sample.AccAddress()
	coins := "100utoken"

	msg := NewMsgBurn(creator, coins)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, coins, msg.Coins)
}

func TestMsgBurn_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()

	tests := []struct {
		name string
		msg  MsgBurn
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgBurn{
				Creator: "invalid_address",
				Coins:   "100utoken",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgBurn{
				Creator: "",
				Coins:   "100utoken",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid coins - empty",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid coins - malformed",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "invalid_coins",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid coins - no denomination",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "100",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid coins - negative",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "-100utoken",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid coins - zero",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "0utoken",
			},
			err: sdkerrors.ErrInvalidCoins,
		},

		{
			name: "invalid coins - multiple coins not supported",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "100utoken,50ustake",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "valid coins - single coin",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "100utoken",
			},
		},
		{
			name: "valid coins - large amount",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "1000000000000utoken",
			},
		},
		{
			name: "valid coins - single unit",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "1utoken",
			},
		},
		{
			name: "valid coins - different denomination",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "500ubze",
			},
		},
		{
			name: "valid coins - denomination with numbers",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "100token123",
			},
		},
		{
			name: "valid coins - long denomination",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   "250verylongdenomination",
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
