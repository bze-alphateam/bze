package helpers_test

import (
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

func TestFeeForGas(t *testing.T) {
	fee, err := helpers.FeeForGas("0.01ubze", 500_000)
	require.NoError(t, err)
	require.Equal(t, "5000ubze", fee.String())

	// ceil, like the CLI: 0.0025 × 333,333 = 833.3325 → 834
	fee, err = helpers.FeeForGas("0.0025ufoo", 333_333)
	require.NoError(t, err)
	require.Equal(t, "834ufoo", fee.String())

	_, err = helpers.FeeForGas("not a price", 1)
	require.Error(t, err)
}

func TestUnsignedTx(t *testing.T) {
	msg := map[string]any{"@type": "/bze.tradebin.MsgFillOrders", "creator": "bze1x"}
	raw, err := helpers.UnsignedTx(sdk.NewCoins(sdk.NewInt64Coin("ubze", 5000)), 500_000, msg, msg)
	require.NoError(t, err)

	var tx struct {
		Body struct {
			Messages []map[string]any `json:"messages"`
		} `json:"body"`
		AuthInfo struct {
			Fee struct {
				Amount   []map[string]string `json:"amount"`
				GasLimit string              `json:"gas_limit"`
			} `json:"fee"`
		} `json:"auth_info"`
		Signatures []string `json:"signatures"`
	}
	require.NoError(t, json.Unmarshal(raw, &tx))
	require.Len(t, tx.Body.Messages, 2)
	require.Equal(t, "/bze.tradebin.MsgFillOrders", tx.Body.Messages[0]["@type"])
	require.Equal(t, []map[string]string{{"denom": "ubze", "amount": "5000"}}, tx.AuthInfo.Fee.Amount)
	require.Equal(t, "500000", tx.AuthInfo.Fee.GasLimit)
	require.Empty(t, tx.Signatures)

	_, err = helpers.UnsignedTx(nil, 1)
	require.Error(t, err)
}

func TestTransferred(t *testing.T) {
	events := []helpers.Event{
		{Type: "transfer", Attributes: map[string]string{"sender": "a", "recipient": "b", "amount": "10ubze,3factory/x/usd"}},
		{Type: "transfer", Attributes: map[string]string{"sender": "a", "recipient": "b", "amount": "5ubze"}},
		{Type: "transfer", Attributes: map[string]string{"sender": "a", "recipient": "c", "amount": "100ubze"}},
		{Type: "coin_received", Attributes: map[string]string{"receiver": "b", "amount": "100ubze"}},
	}

	got, err := helpers.Transferred(events, "a", "b", "ubze")
	require.NoError(t, err)
	require.Equal(t, "15", got.String())

	got, err = helpers.Transferred(events, "a", "b", "factory/x/usd")
	require.NoError(t, err)
	require.Equal(t, "3", got.String())

	got, err = helpers.Transferred(events, "b", "a", "ubze")
	require.NoError(t, err)
	require.True(t, got.IsZero())

	_, err = helpers.Transferred([]helpers.Event{
		{Type: "transfer", Attributes: map[string]string{"sender": "a", "recipient": "b", "amount": "bad!"}},
	}, "a", "b", "ubze")
	require.Error(t, err)
}
