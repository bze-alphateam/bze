package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgBurn(t *testing.T) {
	creator := sample.AccAddress()
	coins := sdk.NewInt64Coin("utoken", 100)

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
				Coins:   sdk.NewInt64Coin("utoken", 100),
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgBurn{
				Creator: "",
				Coins:   sdk.NewInt64Coin("utoken", 100),
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid coins - zero",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   sdk.NewInt64Coin("utoken", 0),
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "valid coins - single coin",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   sdk.NewInt64Coin("utoken", 100),
			},
		},
		{
			name: "valid coins - large amount",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   sdk.NewInt64Coin("utoken", 1000000000000),
			},
		},
		{
			name: "valid coins - single unit",
			msg: MsgBurn{
				Creator: validCreator,
				Coins:   sdk.NewInt64Coin("utoken", 1),
			},
		},
		{
			name: "valid coins - different denomination",
			msg: MsgBurn{
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
