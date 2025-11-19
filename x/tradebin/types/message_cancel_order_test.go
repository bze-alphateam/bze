package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgCancelOrder(t *testing.T) {
	creator := sample.AccAddress()
	marketId := "BTC-USD"
	orderId := "order123"
	orderType := OrderTypeBuy

	msg := NewMsgCancelOrder(creator, marketId, orderId, orderType)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, marketId, msg.MarketId)
	require.Equal(t, orderId, msg.OrderId)
	require.Equal(t, orderType, msg.OrderType)
}

func TestMsgCancelOrder_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validMarketId := "BTC-USD"
	validOrderId := "order123"
	validOrderType := OrderTypeBuy

	tests := []struct {
		name string
		msg  MsgCancelOrder
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgCancelOrder{
				Creator:   "invalid_address",
				MarketId:  validMarketId,
				OrderId:   validOrderId,
				OrderType: validOrderType,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgCancelOrder{
				Creator:   "",
				MarketId:  validMarketId,
				OrderId:   validOrderId,
				OrderType: validOrderType,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty market id",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  "",
				OrderId:   validOrderId,
				OrderType: validOrderType,
			},
			err: ErrInvalidOrderMarketId,
		},
		{
			name: "empty order id",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderId:   "",
				OrderType: validOrderType,
			},
			err: ErrInvalidOrderId,
		},
		{
			name: "invalid order type - empty",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderId:   validOrderId,
				OrderType: "",
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "invalid order type - wrong value",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderId:   validOrderId,
				OrderType: "invalid",
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "invalid order type - uppercase",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderId:   validOrderId,
				OrderType: "BUY",
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "invalid order type - mixed case",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderId:   validOrderId,
				OrderType: "Buy",
			},
			err: ErrInvalidOrderType,
		},
		{
			name: "valid message - buy order",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderId:   validOrderId,
				OrderType: OrderTypeBuy,
			},
		},
		{
			name: "valid message - sell order",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderId:   validOrderId,
				OrderType: OrderTypeSell,
			},
		},
		{
			name: "valid message - numeric market id",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  "12345",
				OrderId:   validOrderId,
				OrderType: validOrderType,
			},
		},
		{
			name: "valid message - numeric order id",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderId:   "67890",
				OrderType: validOrderType,
			},
		},
		{
			name: "valid message - long market id",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  "VERY-LONG-MARKET-IDENTIFIER",
				OrderId:   validOrderId,
				OrderType: validOrderType,
			},
		},
		{
			name: "valid message - long order id",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  validMarketId,
				OrderId:   "very-long-order-identifier-123",
				OrderType: validOrderType,
			},
		},
		{
			name: "valid message - single character ids",
			msg: MsgCancelOrder{
				Creator:   validCreator,
				MarketId:  "A",
				OrderId:   "B",
				OrderType: validOrderType,
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
