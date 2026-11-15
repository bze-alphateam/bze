package v830

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"cosmossdk.io/core/address"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/x/authz"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// The account migration approved via the v8.3.0 upgrade proposal: everything AccountMigrationSource owns at the
// upgrade height moves to AccountMigrationRecovery. Nothing is hard-coded: the handler reads the source's state at
// the upgrade height.
const (
	AccountMigrationSource   = "bze1z3mkcr2jz424w6m49frgjmy9uhlrx69phqwg3l"
	AccountMigrationRecovery = "bze1ucx0c63jut4x9ekstszed64hmcumaa62k84xex"
)

// MigrationStakingKeeper is the part of the staking keeper the account migration uses.
type MigrationStakingKeeper interface {
	ValidatorAddressCodec() address.Codec
	BondDenom(ctx context.Context) (string, error)
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	IterateDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress, cb func(stakingtypes.Delegation) (stop bool)) error
	Unbond(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares sdkmath.LegacyDec) (sdkmath.Int, error)
	Delegate(ctx context.Context, delAddr sdk.AccAddress, bondAmt sdkmath.Int, tokenSrc stakingtypes.BondStatus,
		validator stakingtypes.Validator, subtractAccount bool) (sdkmath.LegacyDec, error)
	RemoveDelegation(ctx context.Context, delegation stakingtypes.Delegation) error
	GetUnbondingDelegations(ctx context.Context, delegator sdk.AccAddress, maxRetrieve uint16) ([]stakingtypes.UnbondingDelegation, error)
	RemoveUnbondingDelegation(ctx context.Context, ubd stakingtypes.UnbondingDelegation) error
	DeleteUnbondingIndex(ctx context.Context, id uint64) error
	GetUBDQueueTimeSlice(ctx context.Context, timestamp time.Time) ([]stakingtypes.DVPair, error)
	SetUBDQueueTimeSlice(ctx context.Context, timestamp time.Time, keys []stakingtypes.DVPair) error
}

// MigrationDistributionKeeper is the part of the distribution keeper the account migration uses.
type MigrationDistributionKeeper interface {
	SetDelegatorWithdrawAddr(ctx context.Context, delAddr, withdrawAddr sdk.AccAddress) error
	WithdrawValidatorCommission(ctx context.Context, valAddr sdk.ValAddress) (sdk.Coins, error)
}

// MigrationAuthzKeeper is the part of the authz keeper the account migration uses.
type MigrationAuthzKeeper interface {
	IterateGrants(ctx context.Context, handler func(granterAddr, granteeAddr sdk.AccAddress, grant authz.Grant) bool)
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
	// Rewards moves the source's x/rewards staking participations and pending unlocks. Optional.
	Rewards MigrationRewardsKeeper
}

// MigrationRewardsKeeper is the part of the rewards keeper the account migration uses.
type MigrationRewardsKeeper interface {
	GetAllStakingRewardParticipant(ctx sdk.Context) []rewardstypes.StakingRewardParticipant
	GetStakingRewardParticipant(ctx sdk.Context, address, rewardId string) (rewardstypes.StakingRewardParticipant, bool)
	SetStakingRewardParticipant(ctx sdk.Context, participant rewardstypes.StakingRewardParticipant)
	RemoveStakingRewardParticipant(ctx sdk.Context, address, rewardId string)
	GetAllPendingUnlockParticipant(ctx sdk.Context) []rewardstypes.PendingUnlockParticipant
	SetPendingUnlockParticipant(ctx sdk.Context, p rewardstypes.PendingUnlockParticipant)
}

func (k AccountMigrationKeepers) complete() bool {
	return k.Account != nil && k.Bank != nil && k.Staking != nil && k.Distribution != nil && k.Authz != nil
}

// migrationStep counts the units of one step.
type migrationStep struct {
	name   string
	ok     int
	failed int
}

type migrationReport struct {
	steps     []*migrationStep
	delegated sdkmath.Int // bond denom delegated to recovery
	paidOut   sdkmath.Int // bond denom paid from a staking pool to recovery
	swept     sdk.Coins   // coins sent from the source's balance
}

func (r *migrationReport) step(name string) *migrationStep {
	s := &migrationStep{name: name}
	r.steps = append(r.steps, s)
	return s
}

// migrateAccount runs the account migration approved via the v8.3.0 upgrade proposal. It never returns an error
// and never panics: every unit of work runs in its own cached context, written only when the unit succeeds. A failed
// unit is logged and leaves its state as it was; the other units go on.
//
// Order matters:
//  1. the source's withdraw address becomes the recovery address, so the rewards every later Unbond pays out land
//     there;
//  2. authz grants given by (and to) the source are revoked;
//  3. each delegation is unbonded from the source and delegated by the recovery address to the same validator, both
//     legs in one unit; if the validator is gone after the unbond (or cannot take a delegation), the amount is paid
//     to the recovery address from the pool holding it;
//  4. each unbonding delegation is delegated back by the recovery address (as a cancelled unbonding would be), or
//     paid out when the validator is gone; its entries, unbonding ids and queue references are removed;
//  5. positions in the chain's own modules;
//  6. last, every spendable denom is sent to the recovery address, one unit per denom.
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
	report.step("recovery account").count(runUnit(ctx, "recovery account", func(c sdk.Context) error {
		if k.Account.GetAccount(c, recovery) == nil {
			k.Account.SetAccount(c, k.Account.NewAccountWithAddress(c, recovery))
		}
		return nil
	}))
	setWithdrawAddress(ctx, k, source, recovery, report.step("withdraw address"))
	withdrawOperatorCommission(ctx, k, source, report.step("validator commission"))
	revokeGrants(ctx, k, source, report.step("authz grants"))
	migrateDelegations(ctx, k, source, recovery, report)
	migrateUnbondingDelegations(ctx, k, source, recovery, report)
	if k.Rewards != nil {
		migrateRewardsPositions(ctx, k.Rewards, source.String(), recovery.String(), report)
	}
	sweepBalances(ctx, k, source, recovery, report)

	logSummary(ctx, k, source, report)
}

func (s *migrationStep) count(err error) {
	if err != nil {
		s.failed++
		return
	}
	s.ok++
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

func setWithdrawAddress(ctx sdk.Context, k AccountMigrationKeepers, source, recovery sdk.AccAddress, s *migrationStep) {
	// Written straight to the store: the message path checks the withdraw-addr-enabled param.
	s.count(runUnit(ctx, "withdraw address", func(c sdk.Context) error {
		return k.Distribution.SetDelegatorWithdrawAddr(c, source, recovery)
	}))
}

// withdrawOperatorCommission withdraws the commission of the validator the source operates, if any, to the
// operator's withdraw address (the recovery address by now). The validator itself stays with its operator key.
func withdrawOperatorCommission(ctx sdk.Context, k AccountMigrationKeepers, source sdk.AccAddress, s *migrationStep) {
	valAddr := sdk.ValAddress(source)
	if _, err := k.Staking.GetValidator(ctx, valAddr); err != nil {
		return
	}
	s.count(runUnit(ctx, "validator commission", func(c sdk.Context) error {
		_, err := k.Distribution.WithdrawValidatorCommission(c, valAddr)
		if errors.Is(err, distrtypes.ErrNoValidatorCommission) {
			return nil
		}
		return err
	}))
}

type grantRef struct {
	granter, grantee sdk.AccAddress
	msgType          string
}

func revokeGrants(ctx sdk.Context, k AccountMigrationKeepers, source sdk.AccAddress, s *migrationStep) {
	var grants []grantRef
	err := runUnit(ctx, "list authz grants", func(c sdk.Context) error {
		var listErr error
		k.Authz.IterateGrants(c, func(granter, grantee sdk.AccAddress, grant authz.Grant) bool {
			if !granter.Equals(source) && !grantee.Equals(source) {
				return false
			}
			auth, err := grant.GetAuthorization()
			if err != nil {
				listErr = errors.Join(listErr, fmt.Errorf("grant %s -> %s: %w", granter, grantee, err))
				return false
			}
			grants = append(grants, grantRef{granter: granter, grantee: grantee, msgType: auth.MsgTypeURL()})
			return false
		})
		return listErr
	})
	if err != nil {
		s.failed++
	}

	for _, g := range grants {
		s.count(runUnit(ctx, fmt.Sprintf("revoke grant %s -> %s %s", g.granter, g.grantee, g.msgType), func(c sdk.Context) error {
			return k.Authz.DeleteGrant(c, g.grantee, g.granter, g.msgType)
		}))
	}
}

func migrateDelegations(ctx sdk.Context, k AccountMigrationKeepers, source, recovery sdk.AccAddress, report *migrationReport) {
	s := report.step("delegations")

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

func migrateUnbondingDelegations(ctx sdk.Context, k AccountMigrationKeepers, source, recovery sdk.AccAddress, report *migrationReport) {
	s := report.step("unbonding delegations")

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
			delegated, paid, err = moveUnbondingDelegation(c, k, source, recovery, ubd, false)
			return err
		})
		if err != nil {
			err = runUnit(ctx, unit+" (payout)", func(c sdk.Context) (err error) {
				delegated, paid, err = moveUnbondingDelegation(c, k, source, recovery, ubd, true)
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
// cannot take a delegation, or payout is set, the balance is paid from the not-bonded pool instead. The record, its
// unbonding ids and its queue references are removed, so nothing completes to the source later.
func moveUnbondingDelegation(
	ctx sdk.Context, k AccountMigrationKeepers, source, recovery sdk.AccAddress, ubd stakingtypes.UnbondingDelegation, payout bool,
) (delegated, paid sdkmath.Int, err error) {
	delegated, paid = sdkmath.ZeroInt(), sdkmath.ZeroInt()

	valAddr, err := k.Staking.ValidatorAddressCodec().StringToBytes(ubd.ValidatorAddress)
	if err != nil {
		return delegated, paid, err
	}

	total := sdkmath.ZeroInt()
	var completions []time.Time
	for _, entry := range ubd.Entries {
		if entry.Balance.IsPositive() {
			total = total.Add(entry.Balance)
		}
		if err = k.Staking.DeleteUnbondingIndex(ctx, entry.UnbondingId); err != nil {
			return delegated, paid, err
		}
		if !slices.ContainsFunc(completions, entry.CompletionTime.Equal) {
			completions = append(completions, entry.CompletionTime)
		}
	}
	if err = k.Staking.RemoveUnbondingDelegation(ctx, ubd); err != nil {
		return delegated, paid, err
	}
	for _, completion := range completions {
		if err = removeFromUBDQueue(ctx, k, completion, ubd.DelegatorAddress, ubd.ValidatorAddress); err != nil {
			return delegated, paid, err
		}
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

// removeFromUBDQueue drops the delegator's pair for validator from the unbonding queue slice at completion.
func removeFromUBDQueue(ctx sdk.Context, k AccountMigrationKeepers, completion time.Time, delegator, validator string) error {
	pairs, err := k.Staking.GetUBDQueueTimeSlice(ctx, completion)
	if err != nil {
		return err
	}
	kept := make([]stakingtypes.DVPair, 0, len(pairs))
	removed := false
	for _, p := range pairs {
		if p.ValidatorAddress == validator && p.DelegatorAddress == delegator {
			removed = true
			continue
		}
		kept = append(kept, p)
	}
	if !removed {
		return nil
	}

	return k.Staking.SetUBDQueueTimeSlice(ctx, completion, kept)
}

// migrateRewardsPositions moves the source's x/rewards positions to the recovery address:
//   - a pending unlock pays its Address field when its epoch comes, so only that field changes (the key, holding
//     the epoch, stays);
//   - a staking participation is re-keyed with the same stake and starting accumulator, so the reward's totals and
//     the accrued rewards are unchanged. When the recovery address already takes part in the same reward the two
//     cannot be merged without paying out pending rewards (rewards module logic), so it is left and logged.
//
// No DAO can exist at the upgrade height (x/dao is new in v8.3.0), so the DAO participation hooks have nothing to
// track for these moves.
func migrateRewardsPositions(ctx sdk.Context, k MigrationRewardsKeeper, source, recovery string, report *migrationReport) {
	unlocks := report.step("rewards pending unlocks")
	var pending []rewardstypes.PendingUnlockParticipant
	if err := runUnit(ctx, "list pending unlocks", func(c sdk.Context) error {
		for _, p := range k.GetAllPendingUnlockParticipant(c) {
			if p.Address == source {
				pending = append(pending, p)
			}
		}
		return nil
	}); err != nil {
		unlocks.failed++
	}
	for _, p := range pending {
		unlocks.count(runUnit(ctx, "pending unlock "+p.Index, func(c sdk.Context) error {
			p.Address = recovery
			k.SetPendingUnlockParticipant(c, p)
			return nil
		}))
	}

	stakes := report.step("rewards staking participations")
	var participations []rewardstypes.StakingRewardParticipant
	if err := runUnit(ctx, "list staking participations", func(c sdk.Context) error {
		for _, p := range k.GetAllStakingRewardParticipant(c) {
			if p.Address == source {
				participations = append(participations, p)
			}
		}
		return nil
	}); err != nil {
		stakes.failed++
	}
	for _, p := range participations {
		stakes.count(runUnit(ctx, "staking participation "+p.RewardId, func(c sdk.Context) error {
			if _, found := k.GetStakingRewardParticipant(c, recovery, p.RewardId); found {
				return fmt.Errorf("recovery address already takes part in staking reward %s", p.RewardId)
			}
			k.RemoveStakingRewardParticipant(c, source, p.RewardId)
			p.Address = recovery
			k.SetStakingRewardParticipant(c, p)
			return nil
		}))
	}
}

func sweepBalances(ctx sdk.Context, k AccountMigrationKeepers, source, recovery sdk.AccAddress, report *migrationReport) {
	s := report.step("balances")

	var spendable sdk.Coins
	if err := runUnit(ctx, "list balances", func(c sdk.Context) error {
		spendable = k.Bank.SpendableCoins(c, source)
		return nil
	}); err != nil {
		s.failed++
		return
	}

	for _, coin := range spendable {
		if !coin.IsPositive() {
			continue
		}
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
		logger.Info("account migration step", "name", s.name, "succeeded", s.ok, "failed", s.failed)
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
