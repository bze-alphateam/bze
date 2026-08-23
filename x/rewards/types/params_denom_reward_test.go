package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestDefaultParams_DenomRewardDefaults(t *testing.T) {
	p := DefaultParams()

	// fees
	require.Equal(t, sdk.NewInt64Coin("ubze", 25_000_000000), p.CreateDenomRewardFee)
	require.Equal(t, sdk.NewInt64Coin("ubze", 25_000_000000), p.CreateDenomRewardPrizeFee)
	require.Equal(t, sdk.NewInt64Coin("ubze", 25_000_000000), p.AddDenomRewardScheduleFee)

	// numeric params — lock defaults to 7 days, min_stake to 0
	require.Equal(t, uint32(50), p.MaxPrizeDenomsPerDr)
	require.Equal(t, uint64(1_000_000), p.ExtraGasForDenomExit)
	require.Equal(t, uint32(7), p.DenomRewardLock)
	require.Equal(t, uint64(0), p.DenomRewardMinStake)

	// defaults must validate
	require.NoError(t, p.Validate())
}

func TestParams_Validate_DenomRewardFees(t *testing.T) {
	// an invalid (negative) DR fee coin must fail validation
	bad := DefaultParams()
	bad.CreateDenomRewardFee = sdk.Coin{Denom: "ubze", Amount: math.NewInt(-1)}
	require.Error(t, bad.Validate())

	bad = DefaultParams()
	bad.CreateDenomRewardPrizeFee = sdk.Coin{Denom: "ubze", Amount: math.NewInt(-1)}
	require.Error(t, bad.Validate())

	bad = DefaultParams()
	bad.AddDenomRewardScheduleFee = sdk.Coin{Denom: "ubze", Amount: math.NewInt(-1)}
	require.Error(t, bad.Validate())
}
