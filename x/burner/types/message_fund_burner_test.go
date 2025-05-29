package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgFundBurner(t *testing.T) {
	creator := sample.AccAddress()
	amount := "100utoken"

	msg := NewMsgFundBurner(creator, amount)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, amount, msg.Amount)
}

func TestMsgFundBurner_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()

	tests := []struct {
		name string
		msg  MsgFundBurner
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgFundBurner{
				Creator: "invalid_address",
				Amount:  "100utoken",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgFundBurner{
				Creator: "",
				Amount:  "100utoken",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid amount - empty",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid amount - malformed",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "invalid_amount",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid amount - no denomination",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "100",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid amount - negative",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "-100utoken",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid amount - zero",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "0utoken",
			},
			err: sdkerrors.ErrInvalidCoins,
		},

		{
			name: "valid amount - single coin",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "100utoken",
			},
		},
		{
			name: "valid amount - large amount",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "1000000000000utoken",
			},
		},
		{
			name: "valid amount - multiple coins",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "100utoken,50ustake",
			},
		},
		{
			name: "valid amount - multiple coins different order",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "50ustake,100utoken",
			},
		},
		{
			name: "valid amount - single unit",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "1utoken",
			},
		},
		{
			name: "valid amount - different denomination",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "500ubze",
			},
		},
		{
			name: "valid amount - denomination with numbers",
			msg: MsgFundBurner{
				Creator: validCreator,
				Amount:  "100token123",
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
