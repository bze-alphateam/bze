package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgStartRaffle(t *testing.T) {
	creator := sample.AccAddress()
	pot := "1000"
	duration := "30"
	chances := "100"
	ratio := "0.5"
	ticketPrice := "10"
	denom := "utoken"

	msg := NewMsgStartRaffle(creator, pot, duration, chances, ratio, ticketPrice, denom)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, pot, msg.Pot)
	require.Equal(t, duration, msg.Duration)
	require.Equal(t, chances, msg.Chances)
	require.Equal(t, ratio, msg.Ratio)
	require.Equal(t, ticketPrice, msg.TicketPrice)
	require.Equal(t, denom, msg.Denom)
}

func TestMsgStartRaffle_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validPot := "1000"
	validDuration := "30"
	validChances := "100"
	validRatio := "0.5"
	validTicketPrice := "10"
	validDenom := "utoken"

	tests := []struct {
		name string
		msg  MsgStartRaffle
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgStartRaffle{
				Creator:     "invalid_address",
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgStartRaffle{
				Creator:     "",
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid pot - empty",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         "",
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid pot - not a number",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         "invalid",
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid pot - negative",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         "-100",
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid pot - zero",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         "0",
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid duration - empty",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    "",
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid duration - not a number",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    "invalid",
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid duration - negative",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    "-10",
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid duration - zero",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    "0",
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid duration - below minimum",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    "0",
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid duration - above maximum",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    "181",
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid ratio - empty",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       "",
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid ratio - not a decimal",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       "invalid",
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid ratio - negative",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       "-0.5",
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid ratio - zero",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       "0",
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid ratio - below minimum",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       "0.005",
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid ratio - above maximum",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       "1.01",
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid chances - empty",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     "",
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid chances - not a number",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     "invalid",
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid chances - below minimum",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     "0",
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid chances - above maximum",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     "1000001",
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid ticket price - empty",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: "",
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid ticket price - not a number",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: "invalid",
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "invalid ticket price - negative",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: "-10",
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid ticket price - zero",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: "0",
				Denom:       validDenom,
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "invalid denom - ibc token",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       "ibc/ABC123",
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "valid message - minimum values",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         "1",
				Duration:    "1",
				Chances:     "1",
				Ratio:       "0.01",
				TicketPrice: "1",
				Denom:       validDenom,
			},
		},
		{
			name: "valid message - maximum values",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         "999999999",
				Duration:    "180",
				Chances:     "1000000",
				Ratio:       "1.00",
				TicketPrice: "999999999",
				Denom:       validDenom,
			},
		},
		{
			name: "valid message - typical values",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       validDenom,
			},
		},
		{
			name: "valid message - different denom",
			msg: MsgStartRaffle{
				Creator:     validCreator,
				Pot:         validPot,
				Duration:    validDuration,
				Chances:     validChances,
				Ratio:       validRatio,
				TicketPrice: validTicketPrice,
				Denom:       "ustake",
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

func TestMsgStartRaffle_ToStorageRaffle(t *testing.T) {
	validCreator := sample.AccAddress()
	validMsg := MsgStartRaffle{
		Creator:     validCreator,
		Pot:         "1000",
		Duration:    "30",
		Chances:     "100",
		Ratio:       "0.5",
		TicketPrice: "10",
		Denom:       "utoken",
	}

	raffle, err := validMsg.ToStorageRaffle()
	require.NoError(t, err)
	require.Equal(t, "1000", raffle.Pot)
	require.Equal(t, uint64(30), raffle.Duration)
	require.Equal(t, uint64(100), raffle.Chances)
	require.Equal(t, "0.5", raffle.Ratio)
	require.Equal(t, "10", raffle.TicketPrice)
	require.Equal(t, "utoken", raffle.Denom)
}

func TestMsgStartRaffle_isAllowedDenomForRaffle(t *testing.T) {
	msg := &MsgStartRaffle{}

	tests := []struct {
		name     string
		denom    string
		expected bool
	}{
		{
			name:     "allowed denom - utoken",
			denom:    "utoken",
			expected: true,
		},
		{
			name:     "allowed denom - ustake",
			denom:    "ustake",
			expected: true,
		},
		{
			name:     "not allowed denom - ibc token",
			denom:    "ibc/ABC123",
			expected: false,
		},
		{
			name:     "not allowed denom - ibc token long",
			denom:    "ibc/ABCDEF1234567890",
			expected: false,
		},
		{
			name:     "allowed denom - contains ibc but not prefix",
			denom:    "uibc",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := msg.isAllowedDenomForRaffle(tt.denom)
			require.Equal(t, tt.expected, result)
		})
	}
}
