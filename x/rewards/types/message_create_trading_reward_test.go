package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgCreateTradingReward(t *testing.T) {
	creator := sample.AccAddress()
	prizeAmount := "1000"
	prizeDenom := "utoken"
	duration := "30"
	marketId := "BTC-USD"
	slots := "5"

	msg := NewMsgCreateTradingReward(creator, prizeAmount, prizeDenom, duration, marketId, slots)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, prizeAmount, msg.PrizeAmount)
	require.Equal(t, prizeDenom, msg.PrizeDenom)
	require.Equal(t, duration, msg.Duration)
	require.Equal(t, marketId, msg.MarketId)
	require.Equal(t, slots, msg.Slots)
}

func TestMsgCreateTradingReward_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validPrizeAmount := "1000"
	validPrizeDenom := "utoken"
	validDuration := "30"
	validMarketId := "BTC-USD"
	validSlots := "5"

	tests := []struct {
		name string
		msg  MsgCreateTradingReward
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgCreateTradingReward{
				Creator:     "invalid_address",
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgCreateTradingReward{
				Creator:     "",
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid prize amount - empty",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: "",
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: ErrInvalidAmount,
		},
		{
			name: "invalid prize amount - not a number",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: "invalid",
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: ErrInvalidAmount,
		},
		{
			name: "invalid prize amount - negative",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: "-100",
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: ErrInvalidAmount,
		},
		{
			name: "invalid prize amount - zero",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: "0",
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: ErrInvalidAmount,
		},
		{
			name: "invalid prize denom - empty",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  "",
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: ErrInvalidPrizeDenom,
		},
		{
			name: "invalid market id - empty",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    "",
				Slots:       validSlots,
			},
			err: ErrInvalidMarketId,
		},
		{
			name: "invalid duration - empty",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    "",
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - not a number",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    "invalid",
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - zero",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    "0",
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - negative",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    "-10",
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - above maximum",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    "366",
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid slots - empty",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       "",
			},
			err: ErrInvalidSlots,
		},
		{
			name: "invalid slots - not a number",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       "invalid",
			},
			err: ErrInvalidSlots,
		},
		{
			name: "invalid slots - zero",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       "0",
			},
			err: ErrInvalidSlots,
		},
		{
			name: "invalid slots - negative",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       "-5",
			},
			err: ErrInvalidSlots,
		},
		{
			name: "invalid slots - above maximum",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       "11",
			},
			err: ErrInvalidSlots,
		},
		{
			name: "valid message - typical values",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    validMarketId,
				Slots:       validSlots,
			},
		},
		{
			name: "valid message - minimum values",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: "1",
				PrizeDenom:  "u",
				Duration:    "1",
				MarketId:    "A",
				Slots:       "1",
			},
		},
		{
			name: "valid message - maximum values",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: "999999999999",
				PrizeDenom:  "longutoken",
				Duration:    "365",
				MarketId:    "VERY-LONG-MARKET-ID",
				Slots:       "10",
			},
		},
		{
			name: "valid message - numeric market id",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    "12345",
				Slots:       validSlots,
			},
		},
		{
			name: "valid message - market id with special chars",
			msg: MsgCreateTradingReward{
				Creator:     validCreator,
				PrizeAmount: validPrizeAmount,
				PrizeDenom:  validPrizeDenom,
				Duration:    validDuration,
				MarketId:    "BTC-USD_SPOT.v2",
				Slots:       validSlots,
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

func TestMsgCreateTradingReward_ToTradingReward(t *testing.T) {
	validCreator := sample.AccAddress()
	validMsg := MsgCreateTradingReward{
		Creator:     validCreator,
		PrizeAmount: "1000",
		PrizeDenom:  "utoken",
		Duration:    "30",
		MarketId:    "BTC-USD",
		Slots:       "5",
	}

	tr, err := validMsg.ToTradingReward()
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1000), tr.PrizeAmount)
	require.Equal(t, "utoken", tr.PrizeDenom)
	require.Equal(t, "BTC-USD", tr.MarketId)
	require.Equal(t, uint32(30), tr.Duration)
	require.Equal(t, uint32(5), tr.Slots)
}
