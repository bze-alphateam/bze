package v830_test

import (
	"context"
	"testing"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"

	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
)

// fakeAccounts is an in-memory account keeper: accounts by address, numbers handed out in order.
type fakeAccounts struct {
	accounts map[string]sdk.AccountI
	next     uint64
	sets     int
}

func newFakeAccounts() *fakeAccounts {
	return &fakeAccounts{accounts: map[string]sdk.AccountI{}, next: 100}
}

func (f *fakeAccounts) GetAccount(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	return f.accounts[addr.String()]
}

func (f *fakeAccounts) NewAccount(_ context.Context, acc sdk.AccountI) sdk.AccountI {
	if err := acc.SetAccountNumber(f.next); err != nil {
		panic(err)
	}
	f.next++

	return acc
}

func (f *fakeAccounts) SetAccount(_ context.Context, acc sdk.AccountI) {
	f.sets++
	f.accounts[acc.GetAddress().String()] = acc
}

var daoAddr = authtypes.NewModuleAddress(daotypes.ModuleName)

func runWithAccounts(t *testing.T, accounts *fakeAccounts) error {
	t.Helper()
	cfg, mm, ctx := guardianSetup(t)

	_, err := v830.CreateUpgradeHandler(cfg, mm, accounts, &fakeGuardian{}, acceptingRateLimits(t), emptyBank(t), noPools{}, nil, &fakeStakingRewards{}, noTraces{}, nil, emptyIBCStore())(ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{fakeModuleName: 2})

	return err
}

// Nothing at the dao address: the handler creates the dao module account (no permissions) with a new number.
func TestHandler_CreatesDaoModuleAccount(t *testing.T) {
	accounts := newFakeAccounts()

	require.NoError(t, runWithAccounts(t, accounts))

	macc, ok := accounts.accounts[daoAddr.String()].(*authtypes.ModuleAccount)
	require.True(t, ok)
	require.Equal(t, daotypes.ModuleName, macc.GetName())
	require.Empty(t, macc.GetPermissions())
	require.Equal(t, uint64(100), macc.GetAccountNumber())
	require.NoError(t, macc.Validate())
}

// A plain account at the dao address becomes the module account with the same number and sequence.
func TestHandler_ConvertsPlainAccountAtDaoAddress(t *testing.T) {
	accounts := newFakeAccounts()
	accounts.accounts[daoAddr.String()] = authtypes.NewBaseAccount(daoAddr, nil, 42, 5)

	require.NoError(t, runWithAccounts(t, accounts))

	macc, ok := accounts.accounts[daoAddr.String()].(*authtypes.ModuleAccount)
	require.True(t, ok)
	require.Equal(t, daotypes.ModuleName, macc.GetName())
	require.Empty(t, macc.GetPermissions())
	require.Equal(t, uint64(42), macc.GetAccountNumber())
	require.Equal(t, uint64(5), macc.GetSequence())
	require.Equal(t, daoAddr, macc.GetAddress())
	require.Equal(t, uint64(100), accounts.next, "no new account number is used")
}

// The dao module account already exists: nothing is written.
func TestHandler_KeepsExistingDaoModuleAccount(t *testing.T) {
	accounts := newFakeAccounts()
	existing := authtypes.NewModuleAccount(authtypes.NewBaseAccount(daoAddr, nil, 7, 0), daotypes.ModuleName)
	accounts.accounts[daoAddr.String()] = existing

	require.NoError(t, runWithAccounts(t, accounts))

	require.Zero(t, accounts.sets)
	require.Same(t, existing, accounts.accounts[daoAddr.String()])
}

// A module account at the dao address with other permissions is not the expected one: the upgrade fails.
func TestHandler_RefusesMismatchedDaoModuleAccount(t *testing.T) {
	accounts := newFakeAccounts()
	accounts.accounts[daoAddr.String()] = authtypes.NewModuleAccount(authtypes.NewBaseAccount(daoAddr, nil, 7, 0), daotypes.ModuleName, authtypes.Minter)

	require.ErrorContains(t, runWithAccounts(t, accounts), "expected \"dao\"")
	require.Zero(t, accounts.sets)
}

// A vesting account at the dao address, which has no public key, becomes the module account with the same number
// and sequence; the vesting schedule is dropped.
func TestHandler_ConvertsVestingAccountAtDaoAddress(t *testing.T) {
	accounts := newFakeAccounts()
	vesting, err := vestingtypes.NewContinuousVestingAccount(
		authtypes.NewBaseAccount(daoAddr, nil, 9, 4), sdk.NewCoins(sdk.NewInt64Coin("ubze", 10)), 1, 2,
	)
	require.NoError(t, err)
	accounts.accounts[daoAddr.String()] = vesting

	require.NoError(t, runWithAccounts(t, accounts))

	macc, ok := accounts.accounts[daoAddr.String()].(*authtypes.ModuleAccount)
	require.True(t, ok)
	require.Equal(t, daotypes.ModuleName, macc.GetName())
	require.Empty(t, macc.GetPermissions())
	require.Equal(t, uint64(9), macc.GetAccountNumber())
	require.Equal(t, uint64(4), macc.GetSequence())
	require.NoError(t, macc.Validate())
}

// An account with a public key at the dao address is left in place: the upgrade goes on, without panicking.
func TestHandler_LeavesAccountWithPubKeyAtDaoAddress(t *testing.T) {
	accounts := newFakeAccounts()
	withKey := authtypes.NewBaseAccount(daoAddr, secp256k1.GenPrivKeyFromSecret([]byte("dao")).PubKey(), 11, 2)
	accounts.accounts[daoAddr.String()] = withKey

	require.NotPanics(t, func() {
		require.NoError(t, runWithAccounts(t, accounts))
	})
	require.Zero(t, accounts.sets)
	require.Same(t, withKey, accounts.accounts[daoAddr.String()])
}

// The account step runs before the module migrations: a failed migration still finds it done.
func TestHandler_EnsuresModuleAccountsBeforeMigrations(t *testing.T) {
	accounts := newFakeAccounts()
	migrated := 0
	mm := module.NewManager(fakeModule{migrationErr: context.Canceled, migrated: &migrated})
	cfg := module.NewConfigurator(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), baseapp.NewMsgServiceRouter(), baseapp.NewGRPCQueryRouter())
	require.NoError(t, mm.RegisterServices(cfg))
	ctx := testutil.DefaultContext(storetypes.NewKVStoreKey("v830"), storetypes.NewTransientStoreKey("v830_t"))

	_, err := v830.CreateUpgradeHandler(cfg, mm, accounts, &fakeGuardian{}, acceptingRateLimits(t), emptyBank(t), noPools{}, nil, &fakeStakingRewards{}, noTraces{}, nil, emptyIBCStore())(ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{fakeModuleName: 1})
	require.Error(t, err)
	require.Equal(t, 1, migrated)
	require.Contains(t, accounts.accounts, daoAddr.String())
}
