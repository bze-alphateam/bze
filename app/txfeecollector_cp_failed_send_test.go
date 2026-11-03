package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// A send to a token's community pool that fails (here a blocked address, which MsgSetDenomConfig refuses but the store
// doesn't) leaves the token's coins in txfeecollector_cp: they are never swapped or burned in its place. Before, a
// failed routing step fell back to the conversion, which burned a token without a pool.
func TestTxfeecollector_FailedCommunityPoolSendLeavesCoinsInPlace(t *testing.T) {
	bApp, ctx := newZeroOutputApp(t)
	admin := sdk.AccAddress("token_admin_________")
	token := "factory/" + admin.String() + "/blocked"
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomAuthority(ctx, token, tokenfactorytypes.DenomAuthority{Admin: admin.String(), Mintable: true}))
	blocked := authtypes.NewModuleAddress(txfeecollectortypes.ModuleName)
	require.True(t, bApp.BankKeeper.BlockedAddr(blocked), "test setup: the community pool address is blocked")
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomConfig(ctx, token, tokenfactorytypes.DenomConfig{
		CommunityPool: blocked.String(),
		TransferFee:   math.LegacyZeroDec(),
		GasSurcharge:  math.LegacyZeroDec(),
	}))

	coin := sdk.NewInt64Coin(token, 1_000)
	mintToModule(t, bApp, ctx, txfeecollectortypes.CpFeeCollector, sdk.NewCoins(coin))

	txfeecollectorEndBlock(t, bApp, ctx)

	cpCollector := authtypes.NewModuleAddress(txfeecollectortypes.CpFeeCollector)
	require.Equal(t, sdk.NewCoins(coin), bApp.BankKeeper.GetAllBalances(ctx, cpCollector), "the coin stays in the collector")
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(burnertypes.ModuleName)).IsZero(), "nothing burned")
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, blocked).IsZero(), "nothing reached the blocked address")
}
