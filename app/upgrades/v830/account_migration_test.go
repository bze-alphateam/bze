package v830_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/runtime"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
)

const (
	migBond       = "ubze"
	migIBCDenom   = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	migFactory    = "factory/bze1creator/ctl"
	migUnbondingT = 21 * 24 * time.Hour
	migSelfStake  = 1_000_000
)

var (
	migSource   = sdk.MustAccAddressFromBech32(v830.AccountMigrationSource)
	migRecovery = sdk.MustAccAddressFromBech32(v830.AccountMigrationRecovery)
)

type migFixture struct {
	t   *testing.T
	app *app.App
	ctx sdk.Context
	msg stakingtypes.MsgServer
}

type migValidator struct {
	operator sdk.AccAddress
	val      sdk.ValAddress
	cons     sdk.ConsAddress
}

// newMigFixture is the real app with staking, distribution and slashing params set and the version map stored, so
// the registered v8.3.0 handler can run through the upgrade keeper.
func newMigFixture(t *testing.T) *migFixture {
	t.Helper()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	start := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{Height: 100, Time: start}).WithHeaderInfo(header.Info{Height: 100, Time: start})

	require.NoError(t, bApp.BankKeeper.SetParams(ctx, banktypes.DefaultParams()))
	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = migBond
	stakingParams.UnbondingTime = migUnbondingT
	require.NoError(t, bApp.StakingKeeper.SetParams(ctx, stakingParams))
	require.NoError(t, bApp.SlashingKeeper.SetParams(ctx, slashingtypes.DefaultParams()))
	require.NoError(t, bApp.DistrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool()))
	require.NoError(t, bApp.DistrKeeper.Params.Set(ctx, distrtypes.DefaultParams()))
	require.NoError(t, bApp.UpgradeKeeper.SetModuleVersionMap(ctx, bApp.ModuleManager.GetVersionMap()))
	// the handler's rate limit step reads the IBC client params, as on any live chain
	bApp.IBCKeeper.ClientKeeper.SetParams(ctx, clienttypes.DefaultParams())

	return &migFixture{t: t, app: bApp, ctx: ctx, msg: stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)}
}

func (f *migFixture) fund(addr sdk.AccAddress, coins ...sdk.Coin) {
	f.t.Helper()
	amt := sdk.NewCoins(coins...)
	require.NoError(f.t, f.app.BankKeeper.MintCoins(f.ctx, minttypes.ModuleName, amt))
	require.NoError(f.t, f.app.BankKeeper.SendCoinsFromModuleToAccount(f.ctx, minttypes.ModuleName, addr, amt))
}

func ubze(amount int64) sdk.Coin { return sdk.NewInt64Coin(migBond, amount) }

func (f *migFixture) nextBlock(d time.Duration) {
	f.ctx = f.ctx.WithBlockHeight(f.ctx.BlockHeight() + 1).WithBlockTime(f.ctx.BlockTime().Add(d))
	f.ctx = f.ctx.WithHeaderInfo(header.Info{Height: f.ctx.BlockHeight(), Time: f.ctx.BlockTime()})
}

func (f *migFixture) endBlock() {
	f.t.Helper()
	_, err := f.app.StakingKeeper.EndBlocker(f.ctx)
	require.NoError(f.t, err)
}

// createValidatorFor creates a validator operated by operator with a self-delegation of selfStake.
func (f *migFixture) createValidatorFor(operator sdk.AccAddress, seed byte, selfStake int64) migValidator {
	f.t.Helper()
	f.fund(operator, ubze(selfStake))
	pk := ed25519.GenPrivKeyFromSecret([]byte{seed}).PubKey()
	msg, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(operator).String(), pk, ubze(selfStake), stakingtypes.Description{Moniker: string('a' + seed)},
		stakingtypes.NewCommissionRates(math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(1, 2)),
		math.OneInt(),
	)
	require.NoError(f.t, err)
	_, err = f.msg.CreateValidator(f.ctx, msg)
	require.NoError(f.t, err)

	return migValidator{operator: operator, val: sdk.ValAddress(operator), cons: sdk.ConsAddress(pk.Address())}
}

func (f *migFixture) createValidator(seed byte) migValidator {
	return f.createValidatorFor(sdk.AccAddress(bytes.Repeat([]byte{seed}, 20)), seed, migSelfStake)
}

func (f *migFixture) delegate(del sdk.AccAddress, v migValidator, amount int64) {
	f.t.Helper()
	_, err := f.msg.Delegate(f.ctx, stakingtypes.NewMsgDelegate(del.String(), v.val.String(), ubze(amount)))
	require.NoError(f.t, err)
}

func (f *migFixture) undelegate(del sdk.AccAddress, v migValidator, amount int64) {
	f.t.Helper()
	_, err := f.msg.Undelegate(f.ctx, stakingtypes.NewMsgUndelegate(del.String(), v.val.String(), ubze(amount)))
	require.NoError(f.t, err)
}

func (f *migFixture) validator(v migValidator) stakingtypes.Validator {
	f.t.Helper()
	val, err := f.app.StakingKeeper.GetValidator(f.ctx, v.val)
	require.NoError(f.t, err)
	return val
}

// allocate gives a validator rewards backed by coins in the distribution module.
func (f *migFixture) allocate(v migValidator, amount int64) {
	f.t.Helper()
	coins := sdk.NewCoins(ubze(amount))
	require.NoError(f.t, f.app.BankKeeper.MintCoins(f.ctx, minttypes.ModuleName, coins))
	require.NoError(f.t, f.app.BankKeeper.SendCoinsFromModuleToModule(f.ctx, minttypes.ModuleName, distrtypes.ModuleName, coins))
	require.NoError(f.t, f.app.DistrKeeper.AllocateTokensToValidator(f.ctx, f.validator(v), sdk.NewDecCoinsFromCoins(coins...)))
}

// pendingRewards is what withdrawing del's rewards at v would pay right now.
func (f *migFixture) pendingRewards(del sdk.AccAddress, v migValidator) math.Int {
	f.t.Helper()
	res, err := distrkeeper.NewQuerier(f.app.DistrKeeper).DelegationRewards(f.ctx, &distrtypes.QueryDelegationRewardsRequest{
		DelegatorAddress: del.String(), ValidatorAddress: v.val.String(),
	})
	require.NoError(f.t, err)
	coins, _ := res.Rewards.TruncateDecimal()
	return coins.AmountOf(migBond)
}

// unbondAmount is what Unbond returns for all of del's shares at v.
func (f *migFixture) unbondAmount(del sdk.AccAddress, v migValidator) math.Int {
	f.t.Helper()
	d, err := f.app.StakingKeeper.GetDelegation(f.ctx, del, v.val)
	require.NoError(f.t, err)
	val := f.validator(v)
	if d.Shares.Equal(val.DelegatorShares) {
		return val.Tokens
	}
	return val.TokensFromShares(d.Shares).TruncateInt()
}

// delegatedValue is the token value of del's delegation at v, zero without one.
func (f *migFixture) delegatedValue(del sdk.AccAddress, v migValidator) math.Int {
	f.t.Helper()
	d, err := f.app.StakingKeeper.GetDelegation(f.ctx, del, v.val)
	if errors.Is(err, stakingtypes.ErrNoDelegation) {
		return math.ZeroInt()
	}
	require.NoError(f.t, err)
	return f.validator(v).TokensFromShares(d.Shares).TruncateInt()
}

func (f *migFixture) applyUpgrade() {
	f.t.Helper()
	require.NotPanics(f.t, func() {
		require.NoError(f.t, f.app.UpgradeKeeper.ApplyUpgrade(f.ctx, upgradetypes.Plan{Name: v830.UpgradeName, Height: f.ctx.BlockHeight()}))
	})
}

// runHandler runs the v8.3.0 handler built with the given account migration keepers.
func (f *migFixture) runHandler(keepers v830.AccountMigrationKeepers) {
	f.t.Helper()
	b := f.app
	handler := v830.CreateUpgradeHandler(b.Configurator(), b.ModuleManager, b.AccountKeeper, b.GuardianKeeper,
		b.RatelimitKeeper, b.BankKeeper, b.TradebinKeeper, b.BankKeeper, b.RewardsKeeper,
		runtime.NewKVStoreService(b.GetKey(transfertypes.StoreKey)), b.TransferKeeper,
		runtime.NewKVStoreService(b.GetKey(ibcexported.StoreKey)), keepers)
	require.NotPanics(f.t, func() {
		_, err := handler(f.ctx, upgradetypes.Plan{Name: v830.UpgradeName}, b.ModuleManager.GetVersionMap())
		require.NoError(f.t, err)
	})
}

func (f *migFixture) sourceDelegations() []stakingtypes.Delegation {
	f.t.Helper()
	dels, err := f.app.StakingKeeper.GetDelegatorDelegations(f.ctx, migSource, 1000)
	require.NoError(f.t, err)
	return dels
}

func (f *migFixture) balance(addr sdk.AccAddress) sdk.Coins {
	return f.app.BankKeeper.GetAllBalances(f.ctx, addr)
}

// requireInvariants checks the staking pools against the validator tokens and unbonding entries, and the
// distribution records against the delegations (the checks the SDK's removed invariants made).
func (f *migFixture) requireInvariants() {
	f.t.Helper()
	sk, dk := f.app.StakingKeeper, f.app.DistrKeeper

	bonded, notBonded := math.ZeroInt(), math.ZeroInt()
	vals, err := sk.GetAllValidators(f.ctx)
	require.NoError(f.t, err)
	for _, v := range vals {
		if v.IsBonded() {
			bonded = bonded.Add(v.Tokens)
		} else {
			notBonded = notBonded.Add(v.Tokens)
		}
	}
	require.NoError(f.t, sk.IterateUnbondingDelegations(f.ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) bool {
		for _, e := range ubd.Entries {
			notBonded = notBonded.Add(e.Balance)
		}
		return false
	}))
	require.Equal(f.t, bonded.String(),
		f.app.BankKeeper.GetBalance(f.ctx, authtypes.NewModuleAddress(stakingtypes.BondedPoolName), migBond).Amount.String(), "bonded pool")
	require.Equal(f.t, notBonded.String(),
		f.app.BankKeeper.GetBalance(f.ctx, authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName), migBond).Amount.String(), "not bonded pool")

	// every delegation has a distribution starting info and every starting info a delegation
	dels, err := sk.GetAllDelegations(f.ctx)
	require.NoError(f.t, err)
	infos := 0
	dk.IterateDelegatorStartingInfos(f.ctx, func(val sdk.ValAddress, del sdk.AccAddress, _ distrtypes.DelegatorStartingInfo) bool {
		_, err := sk.GetDelegation(f.ctx, del, val)
		require.NoError(f.t, err, "starting info without a delegation")
		infos++
		return false
	})
	require.Equal(f.t, len(dels), infos, "delegations vs starting infos")

	// reference counts: one per validator (current period), per delegation and per slash event
	slashes := 0
	dk.IterateValidatorSlashEvents(f.ctx, func(sdk.ValAddress, uint64, distrtypes.ValidatorSlashEvent) bool {
		slashes++
		return false
	})
	refs := uint64(0)
	dk.IterateValidatorHistoricalRewards(f.ctx, func(_ sdk.ValAddress, _ uint64, r distrtypes.ValidatorHistoricalRewards) bool {
		refs += uint64(r.ReferenceCount)
		return false
	})
	require.Equal(f.t, uint64(len(vals)+len(dels)+slashes), refs, "distribution reference counts")
}

func (f *migFixture) requireSourceEmpty() {
	f.t.Helper()
	require.Empty(f.t, f.sourceDelegations(), "source delegations")
	ubds, err := f.app.StakingKeeper.GetUnbondingDelegations(f.ctx, migSource, 1000)
	require.NoError(f.t, err)
	require.Empty(f.t, ubds, "source unbonding delegations")
	require.True(f.t, f.balance(migSource).IsZero(), "source balance: %s", f.balance(migSource))
	wa, err := f.app.DistrKeeper.GetDelegatorWithdrawAddr(f.ctx, migSource)
	require.NoError(f.t, err)
	require.Equal(f.t, migRecovery.String(), wa.String(), "source withdraw address")
}

// Everything the source owns moves to the recovery address through the handler registered in the app: delegations
// to bonded, unbonding, unbonded and removed validators, one where the recovery address already delegates,
// unbonding entries (at a removed validator too), rewards, the grants it gave and every balance.
// Later EndBlockers and a slash of the source validator of a redelegation pay nothing to the source.
func TestAccountMigration_MovesEverything(t *testing.T) {
	f := newMigFixture(t)
	a, b, c, d, e := f.createValidator(1), f.createValidator(2), f.createValidator(3), f.createValidator(4), f.createValidator(5)
	f.endBlock()

	f.fund(migSource, ubze(100_000_000), sdk.NewInt64Coin(migIBCDenom, 390_000), sdk.NewInt64Coin(migFactory, 2_050_000))
	f.fund(migRecovery, ubze(3_000_000))
	f.delegate(migSource, a, 10_000_000)
	f.delegate(migSource, b, 8_000_000)
	f.delegate(migSource, c, 6_000_000)
	f.delegate(migSource, d, 7_000_000)
	f.delegate(migSource, e, 5_000_000)
	f.delegate(migRecovery, c, 2_000_000)

	// c jailed, e's operator leaves (jailed below min self delegation): both unbond, then are unbonded
	f.nextBlock(5 * time.Second)
	f.undelegate(e.operator, e, migSelfStake)
	require.NoError(t, f.app.StakingKeeper.Jail(f.ctx, c.cons))
	f.endBlock()
	f.nextBlock(migUnbondingT + time.Hour)
	f.endBlock()
	require.True(t, f.validator(c).IsUnbonded())
	require.True(t, f.validator(e).IsUnbonded())

	f.nextBlock(5 * time.Second)
	f.allocate(a, 1_000_000)
	f.allocate(b, 500_000)
	f.allocate(d, 300_000)

	// b jailed: unbonding
	f.nextBlock(5 * time.Second)
	require.NoError(t, f.app.StakingKeeper.Jail(f.ctx, b.cons))
	f.endBlock()
	require.True(t, f.validator(b).IsUnbonding())

	// unbonding entries: two at a, one at unbonding b, one at unbonded e (removed by the migration)
	f.nextBlock(5 * time.Second)
	f.undelegate(migSource, a, 100_000)
	f.nextBlock(5 * time.Second)
	f.undelegate(migSource, a, 50_000)
	f.undelegate(migSource, b, 30_000)
	f.undelegate(migSource, e, 500_000)
	ubdCompletion := f.ctx.BlockTime().Add(migUnbondingT)
	// another delegator unbonds in the same block: its pair shares the source's queue slice
	bystander := sdk.AccAddress("migration_bystander_")
	f.fund(bystander, ubze(400_000))
	f.delegate(bystander, a, 400_000)
	f.undelegate(bystander, a, 400_000)

	// a redelegation d -> a
	f.nextBlock(5 * time.Second)
	redelegationHeight := f.ctx.BlockHeight()
	_, err := f.msg.BeginRedelegate(f.ctx, stakingtypes.NewMsgBeginRedelegate(migSource.String(), d.val.String(), a.val.String(), ubze(1_000_000)))
	require.NoError(t, err)

	// authz: given by the source, and two the source did not give, which stay
	other, other2 := sdk.AccAddress("migration_other_____"), sdk.AccAddress("migration_other2____")
	stakeAuth, err := stakingtypes.NewStakeAuthorization([]sdk.ValAddress{a.val}, nil, stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE, nil)
	require.NoError(t, err)
	expiry := f.ctx.BlockTime().Add(365 * 24 * time.Hour)
	require.NoError(t, f.app.AuthzKeeper.SaveGrant(f.ctx, other, migSource, stakeAuth, &expiry))
	sendURL := sdk.MsgTypeURL(&banktypes.MsgSend{})
	require.NoError(t, f.app.AuthzKeeper.SaveGrant(f.ctx, migSource, other, authz.NewGenericAuthorization(sendURL), nil))
	require.NoError(t, f.app.AuthzKeeper.SaveGrant(f.ctx, other2, other, authz.NewGenericAuthorization(sendURL), nil))

	f.nextBlock(5 * time.Second)
	f.allocate(a, 400_000)
	f.allocate(d, 200_000)
	f.requireInvariants()

	// expectations at the upgrade height
	ubdIDs := []uint64{}
	ubds, err := f.app.StakingKeeper.GetUnbondingDelegations(f.ctx, migSource, 1000)
	require.NoError(t, err)
	ubdTotal := map[string]math.Int{}
	for _, ubd := range ubds {
		total := math.ZeroInt()
		for _, entry := range ubd.Entries {
			total = total.Add(entry.Balance)
			ubdIDs = append(ubdIDs, entry.UnbondingId)
		}
		ubdTotal[ubd.ValidatorAddress] = total
	}
	require.Len(t, ubdTotal, 3)

	rewards := math.ZeroInt()
	for _, v := range []migValidator{a, b, c, d, e} {
		rewards = rewards.Add(f.pendingRewards(migSource, v))
	}
	require.True(t, rewards.IsPositive())

	wantRecovery := map[string]math.Int{}
	tokensBefore := map[string]math.Int{}
	for _, v := range []migValidator{a, b, c, d} {
		want := f.delegatedValue(migRecovery, v).Add(f.unbondAmount(migSource, v))
		if total, ok := ubdTotal[v.val.String()]; ok {
			want = want.Add(total)
		}
		wantRecovery[v.val.String()] = want
		tokensBefore[v.val.String()] = f.validator(v).Tokens
	}
	paidOut := f.unbondAmount(migSource, e).Add(ubdTotal[e.val.String()])
	sourceLiquid := f.balance(migSource)
	recoveryBefore := f.balance(migRecovery)

	f.applyUpgrade()

	f.requireSourceEmpty()
	f.requireInvariants()

	for _, v := range []migValidator{a, b, c, d} {
		want := wantRecovery[v.val.String()]
		got := f.delegatedValue(migRecovery, v)
		require.True(t, got.LTE(want) && got.GTE(want.SubRaw(1)), "recovery at %s: got %s want %s", v.val, got, want)
		wantTokens := tokensBefore[v.val.String()]
		if total, ok := ubdTotal[v.val.String()]; ok {
			wantTokens = wantTokens.Add(total)
		}
		require.Equal(t, wantTokens.String(), f.validator(v).Tokens.String(), "tokens of %s", v.val)
	}
	require.True(t, f.validator(b).IsUnbonding())
	require.True(t, f.validator(c).IsUnbonded())
	_, err = f.app.StakingKeeper.GetValidator(f.ctx, e.val)
	require.ErrorIs(t, err, stakingtypes.ErrNoValidatorFound, "e lost its last shares and is removed")

	// rewards, the payout of e and every liquid coin are on the recovery address
	gained := f.balance(migRecovery).Sub(recoveryBefore...)
	require.Equal(t, sourceLiquid.AmountOf(migIBCDenom).String(), gained.AmountOf(migIBCDenom).String())
	require.Equal(t, sourceLiquid.AmountOf(migFactory).String(), gained.AmountOf(migFactory).String())
	// rewards: each delegation's withdrawal at the upgrade height pays what the query computed at that height
	wantBond := sourceLiquid.AmountOf(migBond).Add(paidOut).Add(rewards)
	require.Equal(t, wantBond.String(), gained.AmountOf(migBond).String(), "recovery ubze: liquid + payout + rewards")

	// unbonding records and ids are gone; the bystander's unbonding is still queued in the shared slice
	for _, id := range ubdIDs {
		_, err := f.app.StakingKeeper.GetUnbondingDelegationByUnbondingID(f.ctx, id)
		require.Error(t, err, "unbonding id %d", id)
	}
	pairs, err := f.app.StakingKeeper.GetUBDQueueTimeSlice(f.ctx, ubdCompletion)
	require.NoError(t, err)
	require.Contains(t, pairs, stakingtypes.DVPair{DelegatorAddress: bystander.String(), ValidatorAddress: a.val.String()})
	bystanderBefore := f.balance(bystander).AmountOf(migBond)

	// authz: the grant the source gave is revoked, the others stay
	f.app.AuthzKeeper.IterateGrants(f.ctx, func(granter, grantee sdk.AccAddress, _ authz.Grant) bool {
		require.False(t, granter.Equals(migSource), "grant %s -> %s left", granter, grantee)
		return false
	})
	kept, _ := f.app.AuthzKeeper.GetAuthorization(f.ctx, other2, other, sendURL)
	require.NotNil(t, kept, "an unrelated grant stays")
	kept, _ = f.app.AuthzKeeper.GetAuthorization(f.ctx, migSource, other, sendURL)
	require.NotNil(t, kept, "a grant to the source is the granter's and stays")

	// later: a slash of d (source of the source's redelegation, at its creation height) and the end of every
	// unbonding and redelegation period pay nothing to the source and change nothing on the recovery address
	reds, err := f.app.StakingKeeper.GetRedelegations(f.ctx, migSource, 10)
	require.NoError(t, err)
	require.Len(t, reds, 1, "the redelegation record stays")
	recoveryAfter := f.balance(migRecovery)
	recoveryAtA := f.delegatedValue(migRecovery, a)

	f.nextBlock(5 * time.Second)
	dVal := f.validator(d)
	_, err = f.app.StakingKeeper.Slash(f.ctx, d.cons, redelegationHeight, dVal.ConsensusPower(f.app.StakingKeeper.PowerReduction(f.ctx)), math.LegacyNewDecWithPrec(1, 1))
	require.NoError(t, err)
	f.requireInvariants()
	require.Equal(t, recoveryAtA.String(), f.delegatedValue(migRecovery, a).String(), "the stale redelegation record slashes nothing")

	f.nextBlock(migUnbondingT + time.Hour)
	f.nextBlock(migUnbondingT + time.Hour)
	f.endBlock()
	f.nextBlock(5 * time.Second)
	f.endBlock()

	f.requireInvariants()
	require.True(t, f.balance(migSource).IsZero(), "nothing completes to the source")
	require.Equal(t, recoveryAfter.String(), f.balance(migRecovery).String())
	reds, err = f.app.StakingKeeper.GetRedelegations(f.ctx, migSource, 10)
	require.NoError(t, err)
	require.Empty(t, reds, "the redelegation record completed")
	ubds, err = f.app.StakingKeeper.GetUnbondingDelegations(f.ctx, bystander, 10)
	require.NoError(t, err)
	require.Empty(t, ubds, "the bystander's unbonding completed")
	require.Equal(t, bystanderBefore.AddRaw(400_000).String(), f.balance(bystander).AmountOf(migBond).String())
}

// rejectingBank refuses every send from the source, so the balance sweep moves nothing.
type rejectingBank struct {
	v830.MigrationBankKeeper
}

func (b rejectingBank) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	if from.Equals(migSource) {
		return errors.New("sends from the source refused")
	}
	return b.MigrationBankKeeper.SendCoins(ctx, from, to, amt)
}

// The withdraw address is set before the delegations move: the rewards their unbonding pays land on the recovery
// address directly, not on the source (here the sweep, which would move them later, is refused).
func TestAccountMigration_RewardsGoToRecoveryThroughWithdrawAddress(t *testing.T) {
	f := newMigFixture(t)
	a := f.createValidator(1)
	f.endBlock()
	f.fund(migSource, ubze(5_000_000))
	f.fund(migRecovery, ubze(1))
	f.delegate(migSource, a, 4_000_000)
	f.nextBlock(5 * time.Second)
	f.allocate(a, 1_000_000)
	f.nextBlock(5 * time.Second)

	rewards := f.pendingRewards(migSource, a)
	require.True(t, rewards.IsPositive())
	sourceBefore := f.balance(migSource)
	recoveryBefore := f.balance(migRecovery)

	keepers := f.app.V830AccountMigrationKeepers()
	keepers.Bank = rejectingBank{keepers.Bank}
	f.runHandler(keepers)

	require.Equal(t, sourceBefore.String(), f.balance(migSource).String(), "no reward was paid to the source")
	require.Equal(t, recoveryBefore.Add(sdk.NewCoin(migBond, rewards)).String(), f.balance(migRecovery).String())
	require.Empty(t, f.sourceDelegations())
	f.requireInvariants()
}

// failingStaking fails Unbond for one validator and panics for another.
type failingStaking struct {
	v830.MigrationStakingKeeper
	fail, panicky sdk.ValAddress
}

func (s failingStaking) Unbond(ctx context.Context, del sdk.AccAddress, val sdk.ValAddress, shares math.LegacyDec) (math.Int, error) {
	amount, err := s.MigrationStakingKeeper.Unbond(ctx, del, val, shares)
	switch {
	case val.Equals(s.fail):
		return math.ZeroInt(), errors.New("unbond refused")
	case val.Equals(s.panicky):
		panic("unbond panicked")
	}
	return amount, err
}

// failingBank fails the send of one denom and panics on another.
type failingBank struct {
	v830.MigrationBankKeeper
}

func (b failingBank) SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error {
	err := b.MigrationBankKeeper.SendCoins(ctx, from, to, amt)
	switch {
	case amt.AmountOf("ufail").IsPositive():
		return errors.New("send refused")
	case amt.AmountOf("upanic").IsPositive():
		panic("send panicked")
	}
	return err
}

// A unit that fails or panics (after it has written) leaves its state as it was; the handler still succeeds and
// every other unit completes.
func TestAccountMigration_FailedUnitsLeaveStateUntouched(t *testing.T) {
	f := newMigFixture(t)
	a, b, c := f.createValidator(1), f.createValidator(2), f.createValidator(3)
	f.endBlock()
	f.fund(migSource, ubze(30_000_000), sdk.NewInt64Coin("ufail", 7), sdk.NewInt64Coin("upanic", 8), sdk.NewInt64Coin("uok", 9))
	for _, v := range []migValidator{a, b, c} {
		f.delegate(migSource, v, 5_000_000)
	}
	f.nextBlock(5 * time.Second)
	f.allocate(a, 100_000)
	f.allocate(c, 100_000)
	f.nextBlock(5 * time.Second)

	before := map[string]stakingtypes.Delegation{}
	for _, d := range f.sourceDelegations() {
		before[d.ValidatorAddress] = d
	}
	tokensA, tokensB := f.validator(a).Tokens, f.validator(b).Tokens
	rewardsA := f.pendingRewards(migSource, a)

	keepers := f.app.V830AccountMigrationKeepers()
	keepers.Staking = failingStaking{MigrationStakingKeeper: keepers.Staking, fail: a.val, panicky: b.val}
	keepers.Bank = failingBank{keepers.Bank}
	f.runHandler(keepers)

	after := map[string]stakingtypes.Delegation{}
	for _, d := range f.sourceDelegations() {
		after[d.ValidatorAddress] = d
	}
	require.Len(t, after, 2)
	require.Equal(t, before[a.val.String()], after[a.val.String()], "failed unit left untouched")
	require.Equal(t, before[b.val.String()], after[b.val.String()], "panicked unit left untouched")
	require.Equal(t, tokensA.String(), f.validator(a).Tokens.String())
	require.Equal(t, tokensB.String(), f.validator(b).Tokens.String())
	require.Equal(t, rewardsA.String(), f.pendingRewards(migSource, a).String(), "rewards of the failed unit not withdrawn")
	require.True(t, f.delegatedValue(migRecovery, a).IsZero())
	require.True(t, f.delegatedValue(migRecovery, b).IsZero())
	require.False(t, f.delegatedValue(migRecovery, c).IsZero(), "the other delegation moved")

	left := f.balance(migSource)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ufail", 7), sdk.NewInt64Coin("upanic", 8)).String(), left.String())
	require.Equal(t, "9", f.balance(migRecovery).AmountOf("uok").String())
	require.True(t, f.balance(migRecovery).AmountOf(migBond).IsPositive())
	f.requireInvariants()
}

// failingDelegate refuses every delegation by the recovery address.
type failingDelegate struct {
	v830.MigrationStakingKeeper
}

func (s failingDelegate) Delegate(ctx context.Context, del sdk.AccAddress, amt math.Int, src stakingtypes.BondStatus,
	val stakingtypes.Validator, subtract bool,
) (math.LegacyDec, error) {
	if del.Equals(migRecovery) {
		return math.LegacyZeroDec(), errors.New("delegation refused")
	}
	return s.MigrationStakingKeeper.Delegate(ctx, del, amt, src, val, subtract)
}

// When the validator refuses the recovery delegation, the unbonded amount is paid to the recovery address from the
// bonded pool, and an unbonding entry from the not-bonded pool.
func TestAccountMigration_DelegateFailurePaysOut(t *testing.T) {
	f := newMigFixture(t)
	a := f.createValidator(1)
	f.endBlock()
	f.fund(migSource, ubze(10_000_000))
	f.delegate(migSource, a, 6_000_000)
	f.nextBlock(5 * time.Second)
	f.undelegate(migSource, a, 1_000_000)
	f.nextBlock(5 * time.Second)

	amount := f.unbondAmount(migSource, a)
	tokens := f.validator(a).Tokens
	liquid := f.balance(migSource).AmountOf(migBond)

	keepers := f.app.V830AccountMigrationKeepers()
	keepers.Staking = failingDelegate{keepers.Staking}
	f.runHandler(keepers)

	f.requireSourceEmpty()
	f.requireInvariants()
	require.True(t, f.delegatedValue(migRecovery, a).IsZero())
	require.Equal(t, tokens.Sub(amount).String(), f.validator(a).Tokens.String())
	require.Equal(t, liquid.Add(amount).AddRaw(1_000_000).String(), f.balance(migRecovery).AmountOf(migBond).String())
}

// Delegations worth nothing (dust shares) and records with no shares at all are removed, nothing is left behind.
func TestAccountMigration_DustAndEmptyDelegations(t *testing.T) {
	f := newMigFixture(t)
	a, g := f.createValidator(1), f.createValidator(7)
	f.endBlock()

	// one ubze delegated, then a loses half its tokens: the shares are worth half an ubze
	f.fund(migSource, ubze(1))
	f.delegate(migSource, a, 1)
	f.nextBlock(5 * time.Second)
	aVal := f.validator(a)
	_, err := f.app.StakingKeeper.Slash(f.ctx, a.cons, f.ctx.BlockHeight(), aVal.ConsensusPower(f.app.StakingKeeper.PowerReduction(f.ctx)), math.LegacyNewDecWithPrec(5, 1))
	require.NoError(t, err)
	require.True(t, f.delegatedValue(migSource, a).IsZero(), "precondition: dust delegation")

	// a record with no shares and no distribution record
	require.NoError(t, f.app.StakingKeeper.SetDelegation(f.ctx, stakingtypes.NewDelegation(migSource.String(), g.val.String(), math.LegacyZeroDec())))
	require.Len(t, f.sourceDelegations(), 2)

	f.nextBlock(5 * time.Second)
	f.applyUpgrade()

	require.Empty(t, f.sourceDelegations())
	_, err = f.app.StakingKeeper.GetDelegation(f.ctx, migRecovery, a.val)
	require.ErrorIs(t, err, stakingtypes.ErrNoDelegation, "nothing delegated for dust")
	f.requireInvariants()
}

// When the source operates a validator, its commission goes to the recovery address and its self-delegation moves
// (which jails the validator: the operator is left below its minimum self-delegation).
func TestAccountMigration_SourceIsOperator(t *testing.T) {
	f := newMigFixture(t)
	v := f.createValidatorFor(migSource, 9, 3_000_000)
	f.endBlock()
	f.fund(migRecovery, ubze(1))
	f.nextBlock(5 * time.Second)
	f.allocate(v, 1_000_000)
	f.nextBlock(5 * time.Second)

	commission, err := f.app.DistrKeeper.GetValidatorAccumulatedCommission(f.ctx, v.val)
	require.NoError(t, err)
	commissionCoins, _ := commission.Commission.TruncateDecimal()
	require.True(t, commissionCoins.AmountOf(migBond).IsPositive())
	rewards := f.pendingRewards(migSource, v)
	recoveryBefore := f.balance(migRecovery).AmountOf(migBond)

	f.applyUpgrade()

	f.requireSourceEmpty()
	f.requireInvariants()
	require.True(t, f.validator(v).IsJailed())
	require.Equal(t, "3000000", f.delegatedValue(migRecovery, v).String())
	require.Equal(t, recoveryBefore.Add(commissionCoins.AmountOf(migBond)).Add(rewards).String(), f.balance(migRecovery).AmountOf(migBond).String())
}

// Without a source account the step changes nothing; with an empty one it only sets the withdraw address.
func TestAccountMigration_NoSourceIsNoOp(t *testing.T) {
	f := newMigFixture(t)
	f.applyUpgrade()
	require.Nil(t, f.app.AccountKeeper.GetAccount(f.ctx, migRecovery), "no recovery account created")
	f.app.DistrKeeper.IterateDelegatorWithdrawAddrs(f.ctx, func(del, _ sdk.AccAddress) bool {
		require.False(t, del.Equals(migSource), "no withdraw address written")
		return false
	})

	f = newMigFixture(t)
	f.app.AccountKeeper.SetAccount(f.ctx, f.app.AccountKeeper.NewAccountWithAddress(f.ctx, migSource))
	f.applyUpgrade()
	f.requireSourceEmpty()
	require.True(t, f.balance(migRecovery).IsZero())
}

// panickingCommissionCheck panics when the commission step looks up the source's own validator, outside any unit.
type panickingCommissionCheck struct {
	v830.MigrationStakingKeeper
}

func (s panickingCommissionCheck) GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	if addr.Equals(sdk.ValAddress(migSource)) {
		panic("validator lookup panicked")
	}
	return s.MigrationStakingKeeper.GetValidator(ctx, addr)
}

// A panic in one step outside its units stops that step only: the staking steps and the later steps still run.
func TestAccountMigration_PanickingStepDoesNotStopOthers(t *testing.T) {
	f := newMigFixture(t)
	a := f.createValidator(1)
	f.endBlock()
	f.fund(migSource, ubze(5_000_000))
	f.delegate(migSource, a, 3_000_000)
	f.nextBlock(5 * time.Second)
	f.undelegate(migSource, a, 1_000_000)
	f.nextBlock(5 * time.Second)

	keepers := f.app.V830AccountMigrationKeepers()
	keepers.Staking = panickingCommissionCheck{keepers.Staking}
	f.runHandler(keepers)

	f.requireSourceEmpty()
	f.requireInvariants()
	require.Equal(t, "3000000", f.delegatedValue(migRecovery, a).String())
	require.Equal(t, "2000000", f.balance(migRecovery).AmountOf(migBond).String())
}

// With one keeper missing the whole step is skipped: the source keeps its delegation and balance, the recovery
// address gets nothing and no withdraw address is written, even though every other keeper is wired.
func TestAccountMigration_UnwiredKeepersSkipTheStep(t *testing.T) {
	f := newMigFixture(t)
	a := f.createValidator(1)
	f.endBlock()
	f.fund(migSource, ubze(50_000_000))
	f.delegate(migSource, a, 10_000_000)

	delegatedBefore := f.delegatedValue(migSource, a)
	balanceBefore := f.balance(migSource)
	require.True(t, delegatedBefore.IsPositive())
	require.False(t, balanceBefore.IsZero())

	keepers := f.app.V830AccountMigrationKeepers()
	keepers.Authz = nil
	f.runHandler(keepers)

	require.Equal(t, delegatedBefore.String(), f.delegatedValue(migSource, a).String(), "delegation stays with the source")
	require.Equal(t, balanceBefore.String(), f.balance(migSource).String(), "balance stays with the source")
	require.True(t, f.balance(migRecovery).IsZero(), "nothing reaches the recovery address")
	require.Zero(t, f.delegatedValue(migRecovery, a).Int64(), "recovery address delegates nothing")
	f.app.DistrKeeper.IterateDelegatorWithdrawAddrs(f.ctx, func(del, _ sdk.AccAddress) bool {
		require.False(t, del.Equals(migSource), "no withdraw address written")
		return false
	})
}
