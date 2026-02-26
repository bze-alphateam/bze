package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgCreateOrder(t *testing.T) {
	creator := sample.AccAddress()
	orderType := OrderTypeBuy
	amount := math.NewInt(1000)
	price := math.LegacyMustNewDecFromStr("1.5")
	marketId := "BTC-USD"

	msg := NewMsgCreateOrder(creator, orderType, amount, price, marketId)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, orderType, msg.OrderType)
	require.Equal(t, amount, msg.Amount)
	require.Equal(t, price, msg.Price)
	require.Equal(t, marketId, msg.MarketId)
}

func TestMsgCreateOrder_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validOrderType := OrderTypeBuy
	validAmount := math.NewInt(1000)
	validPrice := math.LegacyMustNewDecFromStr("1.5")
	validMarketId := "BTC-USD"

	tests := []struct {
		name string
		msg  MsgCreateOrder
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgCreateOrder{
				Creator:   "invalid_address",
				OrderType: validOrderType,
				Amount:    validAmount,
				Price:     validPrice,
				MarketId:  validMarketId,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgCreateOrder{
				Creator:   "",
				OrderType: validOrderType,
				Amount:    validAmount,
				Price:     validPrice,
				MarketId:  validMarketId,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid order type - empty",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: "",
				Amount:    validAmount,
				Price:     validPrice,
				MarketId:  validMarketId,
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "invalid order type - wrong value",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: "invalid",
				Amount:    validAmount,
				Price:     validPrice,
				MarketId:  validMarketId,
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "invalid order type - uppercase",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: "BUY",
				Amount:    validAmount,
				Price:     validPrice,
				MarketId:  validMarketId,
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "invalid order type - mixed case",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: "Buy",
				Amount:    validAmount,
				Price:     validPrice,
				MarketId:  validMarketId,
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "invalid amount - negative",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    math.NewInt(-1000),
				Price:     validPrice,
				MarketId:  validMarketId,
			},
			err: ErrInvalidOrderAmount,
		},
		{
			name: "invalid amount - zero",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    math.NewInt(0),
				Price:     validPrice,
				MarketId:  validMarketId,
			},
			err: ErrInvalidOrderAmount,
		},
		{
			name: "invalid price - negative",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    validAmount,
				Price:     math.LegacyMustNewDecFromStr("-1.5"),
				MarketId:  validMarketId,
			},
			err: ErrInvalidOrderPrice,
		},
		{
			name: "invalid price - zero",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    validAmount,
				Price:     math.LegacyMustNewDecFromStr("0"),
				MarketId:  validMarketId,
			},
			err: ErrInvalidOrderPrice,
		},
		{
			name: "invalid price - zero decimal",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    validAmount,
				Price:     math.LegacyMustNewDecFromStr("0.0"),
				MarketId:  validMarketId,
			},
			err: ErrInvalidOrderPrice,
		},
		{
			name: "valid message - buy order",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: OrderTypeBuy,
				Amount:    validAmount,
				Price:     validPrice,
				MarketId:  validMarketId,
			},
		},
		{
			name: "valid message - sell order",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: OrderTypeSell,
				Amount:    validAmount,
				Price:     validPrice,
				MarketId:  validMarketId,
			},
		},
		{
			name: "valid message - minimum amount",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    math.NewInt(1),
				Price:     validPrice,
				MarketId:  validMarketId,
			},
		},
		{
			name: "valid message - large amount",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    math.NewInt(999999999999999),
				Price:     validPrice,
				MarketId:  validMarketId,
			},
		},
		{
			name: "valid message - small price",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    validAmount,
				Price:     math.LegacyMustNewDecFromStr("0.000001"),
				MarketId:  validMarketId,
			},
		},
		{
			name: "valid message - large price",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    validAmount,
				Price:     math.LegacyMustNewDecFromStr("999999.999999"),
				MarketId:  validMarketId,
			},
		},
		{
			name: "valid message - integer price",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    validAmount,
				Price:     math.LegacyMustNewDecFromStr("100"),
				MarketId:  validMarketId,
			},
		},
		{
			name: "invalid message - empty market id",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    validAmount,
				Price:     validPrice,
				MarketId:  "",
			},
			err: ErrInvalidOrderMarketId,
		},
		{
			name: "valid message - numeric market id",
			msg: MsgCreateOrder{
				Creator:   validCreator,
				OrderType: validOrderType,
				Amount:    validAmount,
				Price:     validPrice,
				MarketId:  "12345",
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

func TestTheOtherOrderType(t *testing.T) {
	tests := []struct {
		name      string
		orderType string
		expected  string
	}{
		{
			name:      "buy returns sell",
			orderType: OrderTypeBuy,
			expected:  OrderTypeSell,
		},
		{
			name:      "sell returns buy",
			orderType: OrderTypeSell,
			expected:  OrderTypeBuy,
		},
		{
			name:      "invalid type returns buy",
			orderType: "invalid",
			expected:  OrderTypeBuy,
		},
		{
			name:      "empty type returns buy",
			orderType: "",
			expected:  OrderTypeBuy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TheOtherOrderType(tt.orderType)
			require.Equal(t, tt.expected, result)
		})
	}
}
