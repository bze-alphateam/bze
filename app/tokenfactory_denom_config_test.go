package app_test

import (
	"testing"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"

	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// A DAO treasury is a plain account, so it can be a token's community pool, while the module
// accounts the app blocks are refused by the real bank keeper.
func TestSetDenomConfig_CommunityPoolAgainstRealBlockedAddrs(t *testing.T) {
	bApp, ctx, admin, treasury := newAdminDaoApp(t)
	denom := "factory/" + admin.String() + "/cp"
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomAuthority(ctx, denom, tokenfactorytypes.DenomAuthority{Admin: admin.String(), Mintable: true}))

	setCP := func(cp string) error {
		msg := tokenfactorytypes.NewMsgSetDenomConfig(admin.String(), denom, cp, math.LegacyMustNewDecFromStr("0.001"), math.LegacyZeroDec())
		_, err := runAsTx(bApp, ctx, msg)

		return err
	}

	require.NoError(t, setCP(treasury.String()))
	cp, ok := bApp.TokenfactoryKeeper.GetDenomCommunityPool(ctx, denom)
	require.True(t, ok)
	require.Equal(t, treasury, cp)

	for _, module := range []string{"dao", authtypes.FeeCollectorName, txfeecollectortypes.CpFeeCollector, distrtypes.ModuleName} {
		err := setCP(authtypes.NewModuleAddress(module).String())
		require.ErrorIs(t, err, tokenfactorytypes.ErrInvalidDenomConfig, module)
	}

	cp, ok = bApp.TokenfactoryKeeper.GetDenomCommunityPool(ctx, denom)
	require.True(t, ok)
	require.Equal(t, treasury, cp, "a refused tx leaves the stored config as it was")
}
