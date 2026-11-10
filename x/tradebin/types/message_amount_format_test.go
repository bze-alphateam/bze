package types_test

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	"github.com/stretchr/testify/require"
)

// nonCanonicalAmounts are amount strings that are not in canonical base-10 form.
var nonCanonicalAmounts = []string{"0100", "00", "+10", "0x10", "0o10", "0b10", "1_0", "-0", "", " ", " 10", "10 "}

var canonicalAmounts = []string{"2", "100", "1000000", "123456789012345678901234567890"}

func createOrderMsg(amount string) *types.MsgCreateOrder {
	return types.NewMsgCreateOrder(sample.AccAddress(), types.OrderTypeBuy, amount, "1", "stake/ubze")
}

func fillOrdersMsg(items ...*types.FillOrderItem) *types.MsgFillOrders {
	return types.NewMsgFillOrders(sample.AccAddress(), "stake/ubze", types.OrderTypeSell, items)
}

func TestAmountFormat_CreateOrderRejectsNonCanonical(t *testing.T) {
	for _, amount := range nonCanonicalAmounts {
		t.Run(amount, func(t *testing.T) {
			require.ErrorIs(t, createOrderMsg(amount).ValidateBasic(), types.ErrInvalidOrderAmount, "amount %q must be refused", amount)
		})
	}
}

func TestAmountFormat_CreateOrderAcceptsCanonical(t *testing.T) {
	for _, amount := range canonicalAmounts {
		t.Run(amount, func(t *testing.T) {
			require.NoError(t, createOrderMsg(amount).ValidateBasic())
		})
	}
}

func TestAmountFormat_FillOrdersRejectsNonCanonical(t *testing.T) {
	for _, amount := range nonCanonicalAmounts {
		t.Run(amount, func(t *testing.T) {
			msg := fillOrdersMsg(&types.FillOrderItem{Price: "2", Amount: "100"}, &types.FillOrderItem{Price: "1", Amount: amount})
			require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidOrderAmount, "amount %q must be refused", amount)
		})
	}
}

func TestAmountFormat_FillOrdersAcceptsCanonical(t *testing.T) {
	for _, amount := range canonicalAmounts {
		t.Run(amount, func(t *testing.T) {
			require.NoError(t, fillOrdersMsg(&types.FillOrderItem{Price: "1", Amount: amount}).ValidateBasic())
		})
	}
}

// Prices keep their accepted decimal forms: clients send short decimals and echo the prices of the book.
func TestAmountFormat_PricesKeepDecimalForms(t *testing.T) {
	for _, price := range []string{"1", "1.0", "0.25", "1.500000000000000000", "0.000001"} {
		t.Run(price, func(t *testing.T) {
			msg := createOrderMsg("100")
			msg.Price = price
			require.NoError(t, msg.ValidateBasic())
			require.NoError(t, fillOrdersMsg(&types.FillOrderItem{Price: price, Amount: "100"}).ValidateBasic())
		})
	}
}

func TestAmountFormat_FillOrdersDuplicatePriceForms(t *testing.T) {
	for _, pair := range [][2]string{{"1", "1.0"}, {"1", "1.000000000000000000"}, {"0.5", "0.50"}, {"2.5", "2.50"}} {
		t.Run(pair[0]+"_"+pair[1], func(t *testing.T) {
			msg := fillOrdersMsg(&types.FillOrderItem{Price: pair[0], Amount: "100"}, &types.FillOrderItem{Price: pair[1], Amount: "100"})
			require.Error(t, msg.ValidateBasic(), "prices %q and %q are the same price", pair[0], pair[1])
		})
	}

	require.NoError(t, fillOrdersMsg(&types.FillOrderItem{Price: "1", Amount: "100"}, &types.FillOrderItem{Price: "1.5", Amount: "100"}).ValidateBasic())
}
