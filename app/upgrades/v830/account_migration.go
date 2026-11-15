package v830

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"

	"cosmossdk.io/core/address"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/authz"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// The account migration approved via the v8.3.0 upgrade proposal: everything AccountMigrationSource holds in
// staking and in its balance at the upgrade height moves to AccountMigrationRecovery. Nothing is hard-coded: the
// handler reads the source's state at the upgrade height.
const (
	AccountMigrationSource   = "bze1z3mkcr2jz424w6m49frgjmy9uhlrx69phqwg3l"
	AccountMigrationRecovery = "bze1ucx0c63jut4x9ekstszed64hmcumaa62k84xex"
)

// MigrationStakingKeeper is the part of the staking keeper the account migration uses.
type MigrationStakingKeeper interface {
	ValidatorAddressCodec() address.Codec
	BondDenom(ctx context.Context) (string, error)
	PowerReduction(ctx context.Context) sdkmath.Int
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	IterateDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress, cb func(stakingtypes.Delegation) (stop bool)) error
	Unbond(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares sdkmath.LegacyDec) (sdkmath.Int, error)
	Delegate(ctx context.Context, delAddr sdk.AccAddress, bondAmt sdkmath.Int, tokenSrc stakingtypes.BondStatus,
		validator stakingtypes.Validator, subtractAccount bool) (sdkmath.LegacyDec, error)
	RemoveDelegation(ctx context.Context, delegation stakingtypes.Delegation) error
	GetUnbondingDelegations(ctx context.Context, delegator sdk.AccAddress, maxRetrieve uint16) ([]stakingtypes.UnbondingDelegation, error)
	RemoveUnbondingDelegation(ctx context.Context, ubd stakingtypes.UnbondingDelegation) error
	DeleteUnbondingIndex(ctx context.Context, id uint64) error
}

// MigrationDistributionKeeper is the part of the distribution keeper the account migration uses.
type MigrationDistributionKeeper interface {
	SetDelegatorWithdrawAddr(ctx context.Context, delAddr, withdrawAddr sdk.AccAddress) error
	WithdrawValidatorCommission(ctx context.Context, valAddr sdk.ValAddress) (sdk.Coins, error)
}

// MigrationAuthzKeeper is the part of the authz keeper the account migration uses.
type MigrationAuthzKeeper interface {
	GranterGrants(ctx context.Context, req *authz.QueryGranterGrantsRequest) (*authz.QueryGranterGrantsResponse, error)
	DeleteGrant(ctx context.Context, grantee, granter sdk.AccAddress, msgType string) error
}

// MigrationBankKeeper is the part of the bank keeper the account migration uses.
type MigrationBankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	UndelegateCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// MigrationAccountKeeper is the part of the auth keeper the account migration uses.
type MigrationAccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
}

// AccountMigrationKeepers holds what the account migration step needs. A zero value (any keeper missing) skips
// the step.
type AccountMigrationKeepers struct {
	Account      MigrationAccountKeeper
	Bank         MigrationBankKeeper
	Staking      MigrationStakingKeeper
	Distribution MigrationDistributionKeeper
	Authz        MigrationAuthzKeeper
}

func (k AccountMigrationKeepers) complete() bool {
	return k.Account != nil && k.Bank != nil && k.Staking != nil && k.Distribution != nil && k.Authz != nil
}

// migrationStep counts the units of one step.
type migrationStep struct {
	name    string
	ok      int
	failed  int
	aborted bool
}

type migrationReport struct {
	steps     []*migrationStep
	delegated sdkmath.Int // bond denom delegated to recovery
	paidOut   sdkmath.Int // bond denom paid from a staking pool to recovery
	swept     sdk.Coins   // coins sent from the source's balance
}

func (s *migrationStep) count(err error) {
	if err != nil {
		s.failed++
		return
	}
	s.ok++
}

// runStep runs one step; a panic outside the step's units stops that step only, the next steps still run.
func (r *migrationReport) runStep(ctx sdk.Context, name string, fn func(s *migrationStep)) {
	s := &migrationStep{name: name}
	r.steps = append(r.steps, s)
	defer func() {
		if p := recover(); p != nil {
			s.aborted = true
			ctx.Logger().Error("account migration step stopped by a panic",
				"upgrade", UpgradeName, "step", name, "panic", fmt.Sprintf("%v", p))
		}
	}()
	fn(s)
}

// migrateAccount runs the account migration approved via the v8.3.0 upgrade proposal. It never returns an error
// and never panics: each step recovers on its own, and every unit of work runs in its own cached context, written
// only when the unit succeeds. A failed unit is logged and leaves its state as it was; the other units go on.
//
// Order matters:
//  1. the source's withdraw address becomes the recovery address, so the rewards every later Unbond pays out land
//     there;
//  2. each delegation is unbonded from the source and delegated by the recovery address to the same validator, both
//     legs in one unit; if the validator is gone after the unbond (or cannot take a delegation), the amount is paid
//     to the recovery address from the pool holding it;
//  3. each unbonding delegation is delegated back by the recovery address (as a cancelled unbonding would be), or
//     paid out when the validator is gone; its entries and unbonding ids are removed;
//  4. the commission of a validator the source operates is withdrawn (to the recovery address);
//  5. authz grants given by the source are revoked;
//  6. last, the spendable balance is sent to the recovery address: all at once, or one denom per unit if that fails.
//
// Redelegation records of the source are left in place: once the source has no delegation, completing one moves no
// coins and slashing one finds no delegation to unbond (see the tests).
func migrateAccount(ctx sdk.Context, k AccountMigrationKeepers) {
	logger := ctx.Logger().With("upgrade", UpgradeName, "step", "account migration")
	defer func() {
		if r := recover(); r != nil {
			logger.Error("account migration stopped by a panic", "panic", fmt.Sprintf("%v", r))
		}
	}()

	if !k.complete() {
		logger.Error("account migration skipped: keepers not wired")
		return
	}

	source, err := accAddressFromBech32(AccountMigrationSource)
	if err != nil {
		logger.Error("account migration skipped: invalid source address", "error", err.Error())
		return
	}
	recovery, err := accAddressFromBech32(AccountMigrationRecovery)
	if err != nil {
		logger.Error("account migration skipped: invalid recovery address", "error", err.Error())
		return
	}

	if k.Account.GetAccount(ctx, source) == nil {
		logger.Info("account migration: source account does not exist, nothing to do", "source", AccountMigrationSource)
		return
	}

	report := &migrationReport{delegated: sdkmath.ZeroInt(), paidOut: sdkmath.ZeroInt(), swept: sdk.NewCoins()}

	// Payouts from a staking pool need the recipient account to exist.
	report.runStep(ctx, "recovery account", func(s *migrationStep) {
		s.count(runUnit(ctx, "recovery account", func(c sdk.Context) error {
			if k.Account.GetAccount(c, recovery) == nil {
				k.Account.SetAccount(c, k.Account.NewAccountWithAddress(c, recovery))
			}
			return nil
		}))
	})
	report.runStep(ctx, "withdraw address", func(s *migrationStep) {
		// Written straight to the store: the message path checks the withdraw-addr-enabled param.
		s.count(runUnit(ctx, "withdraw address", func(c sdk.Context) error {
			return k.Distribution.SetDelegatorWithdrawAddr(c, source, recovery)
		}))
	})
	report.runStep(ctx, "delegations", func(s *migrationStep) {
		migrateDelegations(ctx, k, source, recovery, report, s)
	})
	report.runStep(ctx, "unbonding delegations", func(s *migrationStep) {
		migrateUnbondingDelegations(ctx, k, source, recovery, report, s)
	})
	report.runStep(ctx, "validator commission", func(s *migrationStep) {
		withdrawOperatorCommission(ctx, k, source, s)
	})
	report.runStep(ctx, "authz grants", func(s *migrationStep) {
		revokeGrants(ctx, k, source, s)
	})
	report.runStep(ctx, "balances", func(s *migrationStep) {
		sweepBalances(ctx, k, source, recovery, report, s)
	})

	logSummary(ctx, k, source, report)
}

// runUnit runs fn in its own cached context and writes it only when fn returns nil without panicking.
func runUnit(ctx sdk.Context, unit string, fn func(sdk.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		if err != nil {
			ctx.Logger().Error("account migration unit failed, its state is unchanged",
				"upgrade", UpgradeName, "unit", unit, "error", err.Error())
		}
	}()

	cacheCtx, write := ctx.CacheContext()
	if err = fn(cacheCtx); err != nil {
		return err
	}
	write()

	return nil
}

func migrateDelegations(ctx sdk.Context, k AccountMigrationKeepers, source, recovery sdk.AccAddress, report *migrationReport, s *migrationStep) {
	var delegations []stakingtypes.Delegation
	if err := runUnit(ctx, "list delegations", func(c sdk.Context) error {
		return k.Staking.IterateDelegatorDelegations(c, source, func(d stakingtypes.Delegation) bool {
			delegations = append(delegations, d)
			return false
		})
	}); err != nil {
		s.failed++
		return
	}

	for _, d := range delegations {
		var delegated, paid sdkmath.Int
		unit := "delegation to " + d.ValidatorAddress
		err := runUnit(ctx, unit, func(c sdk.Context) (err error) {
			delegated, paid, err = moveDelegation(c, k, source, recovery, d, false)
			return err
		})
		if err != nil {
			// The validator could not take the recovery delegation: pay the unbonded amount out instead.
			err = runUnit(ctx, unit+" (payout)", func(c sdk.Context) (err error) {
				delegated, paid, err = moveDelegation(c, k, source, recovery, d, true)
				return err
			})
		}
		if err != nil && d.Shares.IsZero() {
			// A record with no shares holds nothing; Unbond can fail on it (no distribution starting info).
			err = runUnit(ctx, unit+" (remove empty)", func(c sdk.Context) error {
				return k.Staking.RemoveDelegation(c, d)
			})
		}
		s.count(err)
		if err == nil {
			report.delegated = report.delegated.Add(delegated)
			report.paidOut = report.paidOut.Add(paid)
		}
	}
}

// moveDelegation unbonds all of the source's shares in d and gives the amount to the recovery address: a delegation
// to the same validator, from the pool the tokens are in, or a payout from that pool when the validator is gone,
// cannot take a delegation, or payout is set.
func moveDelegation(
	ctx sdk.Context, k AccountMigrationKeepers, source, recovery sdk.AccAddress, d stakingtypes.Delegation, payout bool,
) (delegated, paid sdkmath.Int, err error) {
	delegated, paid = sdkmath.ZeroInt(), sdkmath.ZeroInt()

	valAddr, err := k.Staking.ValidatorAddressCodec().StringToBytes(d.ValidatorAddress)
	if err != nil {
		return delegated, paid, err
	}
	validator, err := k.Staking.GetValidator(ctx, valAddr)
	if err != nil {
		return delegated, paid, err
	}
	if bytes.Equal(valAddr, source) {
		// Moving the operator's self-delegation leaves it below its minimum: Unbond jails the validator.
		ctx.Logger().Info("account migration: moving the self-delegation of the source's validator",
			"upgrade", UpgradeName, "validator", d.ValidatorAddress,
			"power", validator.ConsensusPower(k.Staking.PowerReduction(ctx)), "jailed", validator.IsJailed())
	}
	// Unbond leaves the tokens in the pool of the validator's status; it does not change the status.
	tokenSrc := validator.GetStatus()

	amount, err := k.Staking.Unbond(ctx, source, valAddr, d.Shares)
	if err != nil {
		return delegated, paid, err
	}
	if !amount.IsPositive() {
		// dust shares worth nothing: the delegation record is gone, nothing to give
		return delegated, paid, nil
	}

	// Unbond removes an unbonded validator whose last shares it took.
	validator, err = k.Staking.GetValidator(ctx, valAddr)
	switch {
	case err == nil && !payout && !validator.InvalidExRate():
		if _, err = k.Staking.Delegate(ctx, recovery, amount, tokenSrc, validator, false); err != nil {
			return delegated, paid, err
		}
		return amount, paid, nil
	case err == nil, errors.Is(err, stakingtypes.ErrNoValidatorFound):
		if err = payFromPool(ctx, k, tokenSrc, recovery, amount); err != nil {
			return delegated, paid, err
		}
		return delegated, amount, nil
	default:
		return delegated, paid, err
	}
}

// payFromPool sends amount of the bond denom from the staking pool matching status to the recovery address.
func payFromPool(ctx sdk.Context, k AccountMigrationKeepers, status stakingtypes.BondStatus, recovery sdk.AccAddress, amount sdkmath.Int) error {
	bondDenom, err := k.Staking.BondDenom(ctx)
	if err != nil {
		return err
	}
	pool := stakingtypes.NotBondedPoolName
	if status == stakingtypes.Bonded {
		pool = stakingtypes.BondedPoolName
	}

	return k.Bank.UndelegateCoinsFromModuleToAccount(ctx, pool, recovery, sdk.NewCoins(sdk.NewCoin(bondDenom, amount)))
}

func migrateUnbondingDelegations(ctx sdk.Context, k AccountMigrationKeepers, source, recovery sdk.AccAddress, report *migrationReport, s *migrationStep) {
	var ubds []stakingtypes.UnbondingDelegation
	if err := runUnit(ctx, "list unbonding delegations", func(c sdk.Context) (err error) {
		ubds, err = k.Staking.GetUnbondingDelegations(c, source, math.MaxUint16)
		return err
	}); err != nil {
		s.failed++
		return
	}

	for _, ubd := range ubds {
		var delegated, paid sdkmath.Int
		unit := "unbonding delegation from " + ubd.ValidatorAddress
		err := runUnit(ctx, unit, func(c sdk.Context) (err error) {
			delegated, paid, err = moveUnbondingDelegation(c, k, recovery, ubd, false)
			return err
		})
		if err != nil {
			err = runUnit(ctx, unit+" (payout)", func(c sdk.Context) (err error) {
				delegated, paid, err = moveUnbondingDelegation(c, k, recovery, ubd, true)
				return err
			})
		}
		s.count(err)
		if err == nil {
			report.delegated = report.delegated.Add(delegated)
			report.paidOut = report.paidOut.Add(paid)
		}
	}
}

// moveUnbondingDelegation mirrors cancelling every entry of ubd, with the recovery address as the delegator: the
// entries' balance, held in the not-bonded pool, is delegated to the same validator. When the validator is gone,
// cannot take a delegation, or payout is set, the balance is paid from the not-bonded pool instead. The record and
// its unbonding ids are removed. Like a cancel, the unbonding queue is left as it is: when its time comes, the
// staking EndBlocker finds no record for the pair and skips it.
func moveUnbondingDelegation(
	ctx sdk.Context, k AccountMigrationKeepers, recovery sdk.AccAddress, ubd stakingtypes.UnbondingDelegation, payout bool,
) (delegated, paid sdkmath.Int, err error) {
	delegated, paid = sdkmath.ZeroInt(), sdkmath.ZeroInt()

	valAddr, err := k.Staking.ValidatorAddressCodec().StringToBytes(ubd.ValidatorAddress)
	if err != nil {
		return delegated, paid, err
	}

	total := sdkmath.ZeroInt()
	for _, entry := range ubd.Entries {
		if entry.Balance.IsPositive() {
			total = total.Add(entry.Balance)
		}
		if err = k.Staking.DeleteUnbondingIndex(ctx, entry.UnbondingId); err != nil {
			return delegated, paid, err
		}
	}
	if err = k.Staking.RemoveUnbondingDelegation(ctx, ubd); err != nil {
		return delegated, paid, err
	}

	if !total.IsPositive() {
		return delegated, paid, nil
	}

	validator, err := k.Staking.GetValidator(ctx, valAddr)
	switch {
	case err == nil && !payout && !validator.InvalidExRate():
		if _, err = k.Staking.Delegate(ctx, recovery, total, stakingtypes.Unbonding, validator, false); err != nil {
			return delegated, paid, err
		}
		return total, paid, nil
	case err == nil, errors.Is(err, stakingtypes.ErrNoValidatorFound):
		if err = payFromPool(ctx, k, stakingtypes.Unbonding, recovery, total); err != nil {
			return delegated, paid, err
		}
		return delegated, total, nil
	default:
		return delegated, paid, err
	}
}

// withdrawOperatorCommission withdraws the commission of the validator the source operates, if any, to the
// operator's withdraw address (the recovery address by now). The validator itself stays with its operator key.
func withdrawOperatorCommission(ctx sdk.Context, k AccountMigrationKeepers, source sdk.AccAddress, s *migrationStep) {
	valAddr := sdk.ValAddress(source)
	validator, err := k.Staking.GetValidator(ctx, valAddr)
	if err != nil {
		return
	}
	ctx.Logger().Info("account migration: the source operates a validator",
		"upgrade", UpgradeName, "validator", validator.GetOperator(),
		"power", validator.ConsensusPower(k.Staking.PowerReduction(ctx)), "jailed", validator.IsJailed())
	s.count(runUnit(ctx, "validator commission", func(c sdk.Context) error {
		_, err := k.Distribution.WithdrawValidatorCommission(c, valAddr)
		if errors.Is(err, distrtypes.ErrNoValidatorCommission) {
			return nil
		}
		return err
	}))
}

type grantRef struct {
	grantee sdk.AccAddress
	msgType string
}

// revokeGrants revokes every authz grant the source gave, read with the granter-prefixed query.
func revokeGrants(ctx sdk.Context, k AccountMigrationKeepers, source sdk.AccAddress, s *migrationStep) {
	var grants []grantRef
	if err := runUnit(ctx, "list authz grants", func(c sdk.Context) error {
		var next []byte
		for {
			res, err := k.Authz.GranterGrants(c, &authz.QueryGranterGrantsRequest{
				Granter: source.String(), Pagination: &query.PageRequest{Key: next, Limit: 100},
			})
			if err != nil {
				return err
			}
			for _, g := range res.Grants {
				auth, ok := g.Authorization.GetCachedValue().(authz.Authorization)
				if !ok {
					return fmt.Errorf("grant to %s: unreadable authorization", g.Grantee)
				}
				grantee, err := accAddressFromBech32(g.Grantee)
				if err != nil {
					return err
				}
				grants = append(grants, grantRef{grantee: grantee, msgType: auth.MsgTypeURL()})
			}
			if res.Pagination == nil || len(res.Pagination.NextKey) == 0 {
				return nil
			}
			next = res.Pagination.NextKey
		}
	}); err != nil {
		s.failed++
		return
	}

	for _, g := range grants {
		s.count(runUnit(ctx, fmt.Sprintf("revoke grant to %s %s", g.grantee, g.msgType), func(c sdk.Context) error {
			return k.Authz.DeleteGrant(c, g.grantee, source, g.msgType)
		}))
	}
}

// sweepBalances sends the source's spendable balance to the recovery address in one send, or one denom per unit
// when that send fails, so a denom that cannot move does not hold back the others.
func sweepBalances(ctx sdk.Context, k AccountMigrationKeepers, source, recovery sdk.AccAddress, report *migrationReport, s *migrationStep) {
	var spendable sdk.Coins
	if err := runUnit(ctx, "list balances", func(c sdk.Context) error {
		spendable = k.Bank.SpendableCoins(c, source)
		return nil
	}); err != nil {
		s.failed++
		return
	}
	if spendable.IsZero() {
		return
	}

	if err := runUnit(ctx, "balances", func(c sdk.Context) error {
		return k.Bank.SendCoins(c, source, recovery, spendable)
	}); err == nil {
		s.ok++
		report.swept = report.swept.Add(spendable...)
		return
	}

	for _, coin := range spendable {
		err := runUnit(ctx, "balance "+coin.Denom, func(c sdk.Context) error {
			return k.Bank.SendCoins(c, source, recovery, sdk.NewCoins(coin))
		})
		s.count(err)
		if err == nil {
			report.swept = report.swept.Add(coin)
		}
	}
}

func logSummary(ctx sdk.Context, k AccountMigrationKeepers, source sdk.AccAddress, report *migrationReport) {
	logger := ctx.Logger().With("upgrade", UpgradeName, "step", "account migration")
	for _, s := range report.steps {
		logger.Info("account migration step", "name", s.name, "succeeded", s.ok, "failed", s.failed, "aborted", s.aborted)
	}

	delegationsLeft := 0
	_ = k.Staking.IterateDelegatorDelegations(ctx, source, func(stakingtypes.Delegation) bool {
		delegationsLeft++
		return false
	})
	unbondingsLeft, _ := k.Staking.GetUnbondingDelegations(ctx, source, math.MaxUint16)

	logger.Info("account migration done",
		"source", AccountMigrationSource,
		"recovery", AccountMigrationRecovery,
		"delegated", report.delegated.String(),
		"paid_out", report.paidOut.String(),
		"swept", report.swept.String(),
		"source_delegations_left", delegationsLeft,
		"source_unbondings_left", len(unbondingsLeft),
		"source_balance_left", k.Bank.GetAllBalances(ctx, source).String(),
	)
}

// accAddressFromBech32 decodes a bech32 account address whatever the configured prefix.
func accAddressFromBech32(addr string) (sdk.AccAddress, error) {
	_, bz, err := bech32.DecodeAndConvert(addr)
	if err != nil {
		return nil, err
	}

	return bz, sdk.VerifyAddressFormat(bz)
}
