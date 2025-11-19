package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgPayPublisherRespect(t *testing.T) {
	creator := sample.AccAddress()
	address := sample.AccAddress()
	amount := "100utoken"

	msg := NewMsgPayPublisherRespect(creator, address, amount)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, address, msg.Address)
	require.Equal(t, amount, msg.Amount)
}

func TestMsgPayPublisherRespect_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validAddress := sample.AccAddress()
	validAmount := "100utoken"

	tests := []struct {
		name string
		msg  MsgPayPublisherRespect
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgPayPublisherRespect{
				Creator: "invalid_address",
				Address: validAddress,
				Amount:  validAmount,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgPayPublisherRespect{
				Creator: "",
				Address: validAddress,
				Amount:  validAmount,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid publisher address",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: "invalid_address",
				Amount:  validAmount,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty publisher address",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: "",
				Amount:  validAmount,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid amount - empty",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
				Amount:  "",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid amount - malformed",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
				Amount:  "invalid_amount",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid amount - no denomination",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
				Amount:  "100",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid amount - negative",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
				Amount:  "-100utoken",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid amount - zero",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
				Amount:  "0utoken",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid amount - multiple coins not supported",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
				Amount:  "100utoken,50ustake",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "valid message - typical values",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
				Amount:  validAmount,
			},
		},
		{
			name: "valid message - same creator and address",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validCreator,
				Amount:  validAmount,
			},
		},
		{
			name: "valid message - minimum amount",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
				Amount:  "1utoken",
			},
		},
		{
			name: "valid message - large amount",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
				Amount:  "1000000000000utoken",
			},
		},
		{
			name: "valid message - different denomination",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
				Amount:  "500ubze",
			},
		},
		{
			name: "valid message - denomination with numbers",
			msg: MsgPayPublisherRespect{
				Creator: validCreator,
				Address: validAddress,
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
