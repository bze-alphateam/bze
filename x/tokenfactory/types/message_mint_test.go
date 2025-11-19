package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgMint(t *testing.T) {
	creator := sample.AccAddress()
	coins := "100utoken"

	msg := NewMsgMint(creator, coins)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, coins, msg.Coins)
}

func TestMsgMint_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()

	tests := []struct {
		name string
		msg  MsgMint
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgMint{
				Creator: "invalid_address",
				Coins:   "100utoken",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgMint{
				Creator: "",
				Coins:   "100utoken",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid coins - empty",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   "",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid coins - malformed",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   "invalid_coins",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid coins - no denomination",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   "100",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid coins - negative",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   "-100utoken",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid coins - zero",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   "0utoken",
			},
			err: sdkerrors.ErrInvalidCoins,
		},

		{
			name: "valid coins - single coin",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   "100utoken",
			},
		},
		{
			name: "valid coins - large amount",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   "1000000000000utoken",
			},
		},
		{
			name: "valid coins - single unit",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   "1utoken",
			},
		},
		{
			name: "valid coins - different denomination",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   "500ubze",
			},
		},
		{
			name: "valid coins - denomination with numbers",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   "100token123",
			},
		},
		{
			name: "valid coins - long denomination",
			msg: MsgMint{
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
