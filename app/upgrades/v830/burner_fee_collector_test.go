package v830_test

import (
	"errors"
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	v830testutil "github.com/bze-alphateam/bze/app/upgrades/v830/testutil"
)

var oldBurnerCollector = authtypes.NewModuleAddress("txfeecollector_burner")

func runSweep(t *testing.T, bank v830.BankKeeper) error {
	t.Helper()
	cfg, mm, ctx := guardianSetup(t)
	_, err := v830.CreateUpgradeHandler(cfg, mm, &fakeGuardian{}, acceptingRateLimits(t), bank, noPools{}, nil, &fakeStakingRewards{}, emptyIBCStore())(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{fakeModuleName: 2},
	)
	return err
}

// Every coin left at the old collector address, whatever its kind, goes to the burner module in one send.
func TestHandler_SweepsBurnerFeeCollectorToBurner(t *testing.T) {
	left := sdk.NewCoins(
		sdk.NewInt64Coin("ubze", 1_000),
		sdk.NewInt64Coin("factory/bze1creator/token", 20),
		sdk.NewInt64Coin("ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4", 3),
		sdk.NewInt64Coin("ulp_factory/bze1creator/token_ubze", 7),
	)
	bank := v830testutil.NewMockBankKeeper(gomock.NewController(t))
	gomock.InOrder(
		bank.EXPECT().GetAllBalances(gomock.Any(), oldBurnerCollector).Return(left).Times(1),
		bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), oldBurnerCollector, "burner", left).Return(nil).Times(1),
	)

	require.NoError(t, runSweep(t, bank))
}

func TestHandler_EmptyBurnerFeeCollectorSendsNothing(t *testing.T) {
	bank := v830testutil.NewMockBankKeeper(gomock.NewController(t))
	bank.EXPECT().GetAllBalances(gomock.Any(), oldBurnerCollector).Return(sdk.NewCoins()).Times(1)
	bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	require.NoError(t, runSweep(t, bank))
}

func TestHandler_SweepErrorFailsTheUpgrade(t *testing.T) {
	boom := errors.New("send failed")
	left := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1))
	bank := v830testutil.NewMockBankKeeper(gomock.NewController(t))
	bank.EXPECT().GetAllBalances(gomock.Any(), oldBurnerCollector).Return(left).Times(1)
	bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), oldBurnerCollector, "burner", left).Return(boom).Times(1)

	require.ErrorIs(t, runSweep(t, bank), boom)
}

// The sweep runs last: a failed step before it leaves the collector untouched.
func TestHandler_RateLimitErrorSkipsTheSweep(t *testing.T) {
	cfg, mm, ctx := guardianSetup(t)
	rl := v830testutil.NewMockRateLimitKeeper(gomock.NewController(t))
	boom := errors.New("store failure")
	rl.EXPECT().AddRateLimit(gomock.Any(), gomock.Any()).Return(boom).Times(1)
	bank := v830testutil.NewMockBankKeeper(gomock.NewController(t))
	bank.EXPECT().GetAllBalances(gomock.Any(), gomock.Any()).Times(0)

	_, err := v830.CreateUpgradeHandler(cfg, mm, &fakeGuardian{}, rl, bank, noPools{}, nil, &fakeStakingRewards{}, emptyIBCStore())(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{fakeModuleName: 2},
	)
	require.ErrorIs(t, err, boom)
}
