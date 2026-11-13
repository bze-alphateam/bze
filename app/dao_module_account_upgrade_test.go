package app_test

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
)

// newDaoUpgradeApp is the DAO test app with the IBC client params the upgrade's rate-limit step reads.
func newDaoUpgradeApp(t *testing.T) (*app.App, sdk.Context, sdk.AccAddress) {
	t.Helper()
	bApp, ctx, admin, _ := newAdminDaoApp(t)
	// no InitChain here: the client params must be set by hand
	bApp.IBCKeeper.ClientKeeper.SetParams(ctx, clienttypes.DefaultParams())

	return bApp, ctx, admin
}

// Before v8.3.0 the dao module address was not blocked, so a plain send could leave a BaseAccount there. The
// upgrade turns it into the dao module account with the real auth keeper: same address, account number and
// sequence, the app's permissions, balance untouched. A proposal escrowing a deposit then succeeds.
func TestUpgrade_ConvertsPlainAccountAtDaoAddress(t *testing.T) {
	bApp, ctx, admin := newDaoUpgradeApp(t)
	daoAddr := authtypes.NewModuleAddress(daotypes.ModuleName)
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, daoAddr), "no account at the dao address yet")

	// pre-upgrade state: a plain send to the address creates a BaseAccount (the keeper-level send skips the
	// blocked-recipient check that v8.3.0 applies to messages)
	planted := sdk.NewCoins(sdk.NewInt64Coin("ubze", 7))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, planted))
	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, authtypes.NewModuleAddress(minttypes.ModuleName), daoAddr, planted))
	plain, ok := bApp.AccountKeeper.GetAccount(ctx, daoAddr).(*authtypes.BaseAccount)
	require.True(t, ok, "a plain account sits at the dao address")
	require.NoError(t, plain.SetSequence(3))
	bApp.AccountKeeper.SetAccount(ctx, plain)

	_, err := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.AccountKeeper, bApp.GuardianKeeper, bApp.RatelimitKeeper, bApp.BankKeeper, bApp.TradebinKeeper, bApp.BankKeeper, bApp.RewardsKeeper, runtime.NewKVStoreService(bApp.GetKey(ibctransfertypes.StoreKey)), bApp.TransferKeeper, ibcStoreService(bApp))(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, bApp.ModuleManager.GetVersionMap(),
	)
	require.NoError(t, err)

	macc, ok := bApp.AccountKeeper.GetAccount(ctx, daoAddr).(sdk.ModuleAccountI)
	require.True(t, ok, "the dao address holds a module account")
	require.Equal(t, daotypes.ModuleName, macc.GetName())
	require.ElementsMatch(t, app.GetMaccPerms()[daotypes.ModuleName], macc.GetPermissions())
	require.Equal(t, plain.GetAccountNumber(), macc.GetAccountNumber())
	require.Equal(t, uint64(3), macc.GetSequence())
	require.Equal(t, planted.String(), bApp.BankKeeper.GetAllBalances(ctx, daoAddr).String())
	require.Equal(t, macc.GetAddress(), bApp.AccountKeeper.GetModuleAccount(ctx, daotypes.ModuleName).GetAddress())

	// a proposal with a positive deposit escrows it in the dao module account
	proposer := sdk.AccAddress("dao_proposer________")
	config := newAdminDaoConfig()
	deposit := sdk.NewInt64Coin("ubze", 40)
	config.ProposalDeposit = &deposit
	_, err = runAsTx(bApp, ctx, &daotypes.MsgCreateDao{
		Creator: admin.String(),
		Admin:   admin.String(),
		Type:    daotypes.DaoType_DAO_TYPE_STATIC_LIST,
		Name:    "depositors",
		Config:  config,
		Members: []string{proposer.String()},
	})
	require.NoError(t, err)
	funds := sdk.NewCoins(sdk.NewInt64Coin("ubze", 100))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, funds))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, proposer, funds))

	// DAO 1 is the helper's DAO; this one is 2
	submitMsg, err := daotypes.NewMsgSubmitProposal(proposer.String(), 2, "signal", "", nil, false)
	require.NoError(t, err)
	require.NotPanics(t, func() {
		_, err = runAsTx(bApp, ctx, submitMsg)
	})
	require.NoError(t, err)
	require.Equal(t, int64(47), bApp.BankKeeper.GetBalance(ctx, daoAddr, "ubze").Amount.Int64())
	require.Equal(t, int64(60), bApp.BankKeeper.GetBalance(ctx, proposer, "ubze").Amount.Int64())
}

// With nothing at the dao address the upgrade creates the module account with the app's permissions.
func TestUpgrade_CreatesDaoModuleAccount(t *testing.T) {
	bApp, ctx, _ := newDaoUpgradeApp(t)
	daoAddr := authtypes.NewModuleAddress(daotypes.ModuleName)
	require.Nil(t, bApp.AccountKeeper.GetAccount(ctx, daoAddr))

	_, err := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.AccountKeeper, bApp.GuardianKeeper, bApp.RatelimitKeeper, bApp.BankKeeper, bApp.TradebinKeeper, bApp.BankKeeper, bApp.RewardsKeeper, runtime.NewKVStoreService(bApp.GetKey(ibctransfertypes.StoreKey)), bApp.TransferKeeper, ibcStoreService(bApp))(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, bApp.ModuleManager.GetVersionMap(),
	)
	require.NoError(t, err)

	macc, ok := bApp.AccountKeeper.GetAccount(ctx, daoAddr).(sdk.ModuleAccountI)
	require.True(t, ok)
	require.Equal(t, daotypes.ModuleName, macc.GetName())
	require.ElementsMatch(t, app.GetMaccPerms()[daotypes.ModuleName], macc.GetPermissions())
	require.True(t, bApp.BankKeeper.BlockedAddr(daoAddr), "the dao module address is blocked")
}
