package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// The app wires the tokenfactory keeper as txfeecollector's denom community pool provider: a factory token's fee coins
// held for the community pool reach the address set in its DenomConfig, the other coins stay for the conversion.
func TestTxfeecollector_RoutesFeesToDenomCommunityPool(t *testing.T) {
	bApp, ctx, admin, treasury := newAdminDaoApp(t)
	require.NoError(t, bApp.TxfeecollectorKeeper.SetParams(ctx, txfeecollectortypes.DefaultParams()))

	withCp := "factory/" + admin.String() + "/withcp"
	withoutCp := "factory/" + admin.String() + "/nocp"
	for _, denom := range []string{withCp, withoutCp} {
		require.NoError(t, bApp.TokenfactoryKeeper.SetDenomAuthority(ctx, denom, tokenfactorytypes.DenomAuthority{Admin: admin.String(), Mintable: true}))
	}
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomConfig(ctx, withCp, tokenfactorytypes.DenomConfig{
		CommunityPool: treasury.String(),
		TransferFee:   math.LegacyZeroDec(),
		GasSurcharge:  math.LegacyZeroDec(),
	}))

	fees := sdk.NewCoins(sdk.NewInt64Coin(withCp, 1_000), sdk.NewInt64Coin(withoutCp, 2_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, fees))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToModule(ctx, tokenfactorytypes.ModuleName, txfeecollectortypes.CpFeeCollector, fees))

	bApp.TxfeecollectorKeeper.RouteCommunityPoolFeesToDenomCommunityPools(ctx)

	require.Equal(t, math.NewInt(1_000), bApp.BankKeeper.GetBalance(ctx, treasury, withCp).Amount)
	cpCollector := authtypes.NewModuleAddress(txfeecollectortypes.CpFeeCollector)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(withoutCp, 2_000)), bApp.BankKeeper.GetAllBalances(ctx, cpCollector))
}
