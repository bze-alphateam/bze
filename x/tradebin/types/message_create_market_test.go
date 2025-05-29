package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgCreateMarket(t *testing.T) {
	creator := sample.AccAddress()
	base := "uatom"
	quote := "uusdc"

	msg := NewMsgCreateMarket(creator, base, quote)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, base, msg.Base)
	require.Equal(t, quote, msg.Quote)
}

func TestMsgCreateMarket_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validBase := "uatom"
	validQuote := "uusdc"

	tests := []struct {
		name string
		msg  MsgCreateMarket
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgCreateMarket{
				Creator: "invalid_address",
				Base:    validBase,
				Quote:   validQuote,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgCreateMarket{
				Creator: "",
				Base:    validBase,
				Quote:   validQuote,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty base",
			msg: MsgCreateMarket{
				Creator: validCreator,
				Base:    "",
				Quote:   validQuote,
			},
			err: ErrInvalidDenom,
		},
		{
			name: "empty quote",
			msg: MsgCreateMarket{
				Creator: validCreator,
				Base:    validBase,
				Quote:   "",
			},
			err: ErrInvalidDenom,
		},
		{
			name: "both base and quote empty",
			msg: MsgCreateMarket{
				Creator: validCreator,
				Base:    "",
				Quote:   "",
			},
			err: ErrInvalidDenom,
		},
		{
			name: "base and quote are the same",
			msg: MsgCreateMarket{
				Creator: validCreator,
				Base:    "uatom",
				Quote:   "uatom",
			},
			err: ErrInvalidDenom,
		},
		{
			name: "valid message - typical denoms",
			msg: MsgCreateMarket{
				Creator: validCreator,
				Base:    validBase,
				Quote:   validQuote,
			},
		},
		{
			name: "valid message - single character denoms",
			msg: MsgCreateMarket{
				Creator: validCreator,
				Base:    "a",
				Quote:   "b",
			},
		},
		{
			name: "valid message - long denoms",
			msg: MsgCreateMarket{
				Creator: validCreator,
				Base:    "verylongbasedenomination",
				Quote:   "verylongquotedenomination",
			},
		},
		{
			name: "valid message - denoms with numbers",
			msg: MsgCreateMarket{
				Creator: validCreator,
				Base:    "token123",
				Quote:   "coin456",
			},
		},
		{
			name: "valid message - denoms with special chars",
			msg: MsgCreateMarket{
				Creator: validCreator,
				Base:    "base-token",
				Quote:   "quote_coin",
			},
		},
		{
			name: "valid message - mixed case denoms",
			msg: MsgCreateMarket{
				Creator: validCreator,
				Base:    "BaseToken",
				Quote:   "QuoteCoin",
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
