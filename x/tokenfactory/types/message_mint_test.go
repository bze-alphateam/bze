package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgMint(t *testing.T) {
	creator := sample.AccAddress()
	coins := sdk.NewInt64Coin("utoken", 100)

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
				Coins:   sdk.NewInt64Coin("utoken", 100),
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgMint{
				Creator: "",
				Coins:   sdk.NewInt64Coin("utoken", 100),
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid coins - zero",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   sdk.NewInt64Coin("utoken", 0),
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "valid coins - single coin",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   sdk.NewInt64Coin("utoken", 100),
			},
		},
		{
			name: "valid coins - large amount",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   sdk.NewInt64Coin("utoken", 1000000000000),
			},
		},
		{
			name: "valid coins - single unit",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   sdk.NewInt64Coin("utoken", 1),
			},
		},
		{
			name: "valid coins - different denomination",
			msg: MsgMint{
				Creator: validCreator,
				Coins:   sdk.NewInt64Coin("ubze", 500),
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
