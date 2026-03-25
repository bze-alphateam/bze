package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgFillOrders(t *testing.T) {
	creator := sample.AccAddress()
	marketId := "BTC-USD"
	orderType := OrderTypeBuy
	orders := []*FillOrderItem{
		{Price: "1.5", Amount: "1000"},
		{Price: "1.6", Amount: "2000"},
	}

	msg := NewMsgFillOrders(creator, marketId, orderType, orders)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, marketId, msg.MarketId)
	require.Equal(t, orderType, msg.OrderType)
	require.Equal(t, orders, msg.Orders)
}

func TestMsgFillOrders_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validMarketId := "BTC-USD"
	validOrderType := OrderTypeBuy
	validOrders := []*FillOrderItem{
		{Price: "1.5", Amount: "1000"},
	}

	tests := []struct {
		name string
		msg  MsgFillOrders
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgFillOrders{
				Creator:   "invalid_address",
				MarketId:  validMarketId,
				OrderType: validOrderType,
				Orders:    validOrders,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgFillOrders{
				Creator:   "",
				MarketId:  validMarketId,
				OrderType: validOrderType,
				Orders:    validOrders,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty market id",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  "",
				OrderType: validOrderType,
				Orders:    validOrders,
			},
			err: ErrInvalidOrderMarketId,
		},
		{
			name: "invalid order type - empty",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: "",
				Orders:    validOrders,
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "invalid order type - wrong value",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: "invalid",
				Orders:    validOrders,
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "invalid order type - uppercase",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: "BUY",
				Orders:    validOrders,
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "invalid order type - mixed case",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: "Buy",
				Orders:    validOrders,
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "empty orders slice",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: validOrderType,
				Orders:    []*FillOrderItem{},
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "nil orders slice",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: validOrderType,
				Orders:    nil,
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid message - buy order with single order",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: OrderTypeBuy,
				Orders:    validOrders,
			},
		},
		{
			name: "valid message - sell order with single order",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: OrderTypeSell,
				Orders:    validOrders,
			},
		},
		{
			name: "valid message - multiple orders",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: validOrderType,
				Orders: []*FillOrderItem{
					{Price: "1.5", Amount: "1000"},
					{Price: "1.6", Amount: "2000"},
					{Price: "1.7", Amount: "3000"},
				},
			},
		},
		{
			name: "valid message - numeric market id",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  "12345",
				OrderType: validOrderType,
				Orders:    validOrders,
			},
		},
		{
			name: "valid message - long market id",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  "VERY-LONG-MARKET-IDENTIFIER",
				OrderType: validOrderType,
				Orders:    validOrders,
			},
		},
		{
			name: "valid message - orders with different values",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: validOrderType,
				Orders: []*FillOrderItem{
					{Price: "0.001", Amount: "1"},
					{Price: "999999.999", Amount: "999999999"},
				},
			},
		},
		{
			name: "invalid price - empty value",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: validOrderType,
				Orders: []*FillOrderItem{
					{Price: "", Amount: ""},
				},
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "duplicate prices in orders",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: validOrderType,
				Orders: []*FillOrderItem{
					{Price: "1.5", Amount: "1000"},
					{Price: "1.5", Amount: "2000"},
				},
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "duplicate prices in orders - three orders with two duplicates",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: validOrderType,
				Orders: []*FillOrderItem{
					{Price: "1.5", Amount: "1000"},
					{Price: "1.6", Amount: "2000"},
					{Price: "1.5", Amount: "3000"},
				},
			},
			err: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "duplicate prices in orders - duplicate empty prices",
			msg: MsgFillOrders{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderType: validOrderType,
				Orders: []*FillOrderItem{
					{Price: "", Amount: "1000"},
					{Price: "", Amount: "2000"},
				},
			},
			err: sdkerrors.ErrInvalidRequest,
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
