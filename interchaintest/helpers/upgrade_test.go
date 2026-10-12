package helpers_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

func TestSupplyChange(t *testing.T) {
	events := []helpers.Event{
		{Type: "coinbase", Attributes: map[string]string{"minter": "mint", "amount": "100ubze"}},
		{Type: "coinbase", Attributes: map[string]string{"minter": "mint", "amount": "50ubze,7uabc"}},
		{Type: "burn", Attributes: map[string]string{"burner": "burner", "amount": "30ubze"}},
		{Type: "transfer", Attributes: map[string]string{"sender": "a", "recipient": "b", "amount": "999ubze"}},
	}
	minted, burned, err := helpers.SupplyChange(events)
	require.NoError(t, err)
	require.Equal(t, map[string]sdk.Coins{"mint": sdk.NewCoins(sdk.NewInt64Coin("ubze", 150), sdk.NewInt64Coin("uabc", 7))}, minted)
	require.Equal(t, map[string]sdk.Coins{"burner": sdk.NewCoins(sdk.NewInt64Coin("ubze", 30))}, burned)

	minted, burned, err = helpers.SupplyChange(nil)
	require.NoError(t, err)
	require.Empty(t, minted)
	require.Empty(t, burned)

	_, _, err = helpers.SupplyChange([]helpers.Event{{Type: "burn", Attributes: map[string]string{"burner": "x", "amount": "not coins"}}})
	require.Error(t, err)
}
