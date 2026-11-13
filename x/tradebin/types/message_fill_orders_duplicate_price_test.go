package types_test

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

// Two spellings of the same price in one MsgFillOrders are refused by the duplicate-price branch,
// not by any other check: every spelling on its own is a valid price.
func TestMsgFillOrders_DuplicatePriceSpellings(t *testing.T) {
	pairs := [][2]string{
		{"1.5", "1.50"},
		{"1.5", "01.5"},
		{"1", "+1"},
		{"1", "1.000000000000000000"},
		{"0.25", "+0.250"},
	}

	for _, pair := range pairs {
		t.Run(pair[0]+"_"+pair[1], func(t *testing.T) {
			for _, price := range pair {
				single := types.NewMsgFillOrders(sample.AccAddress(), "stake/ubze", types.OrderTypeBuy,
					[]*types.FillOrderItem{{Price: price, Amount: "100"}})
				require.NoError(t, single.ValidateBasic(), "price %q alone must be valid", price)
			}

			msg := types.NewMsgFillOrders(sample.AccAddress(), "stake/ubze", types.OrderTypeBuy, []*types.FillOrderItem{
				{Price: pair[0], Amount: "100"},
				{Price: pair[1], Amount: "200"},
			})
			err := msg.ValidateBasic()
			require.ErrorIs(t, err, sdkerrors.ErrInvalidRequest)
			require.ErrorContains(t, err, "duplicate price "+pair[1]+" found in items")
		})
	}
}

// Distinct prices that only share a prefix are not duplicates.
func TestMsgFillOrders_DistinctPricesAccepted(t *testing.T) {
	msg := types.NewMsgFillOrders(sample.AccAddress(), "stake/ubze", types.OrderTypeBuy, []*types.FillOrderItem{
		{Price: "1.5", Amount: "100"},
		{Price: "1.05", Amount: "100"},
		{Price: "15", Amount: "100"},
	})
	require.NoError(t, msg.ValidateBasic())
}
