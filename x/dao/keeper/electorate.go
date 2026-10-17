package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// isLiveElectorate reports whether the DAO has an electorate able to vote (D11).
// A static list is live when it is non-empty. A stake electorate is live when
// it names at least one source and its sources hold some stake; a source that
// no longer exists adds nothing.
func (k Keeper) isLiveElectorate(ctx sdk.Context, dao types.Dao) bool {
	switch dao.Type {
	case types.DaoType_DAO_TYPE_STATIC_LIST:
		return len(dao.Members) > 0
	case types.DaoType_DAO_TYPE_STAKE:
		if dao.Electorate.IsEmpty() {
			return false
		}

		return k.electorateStake(ctx, dao.Electorate).IsPositive()
	default:
		return false
	}
}

// electorateStake sums the staked amount over the electorate's sources.
func (k Keeper) electorateStake(ctx sdk.Context, electorate *types.StakeElectorate) math.Int {
	total := math.ZeroInt()
	for _, id := range electorate.SrIds {
		sr, found := k.rewardsKeeper.GetStakingReward(ctx, id)
		if !found {
			continue
		}

		// staked_amount is a plain string in x/rewards; an unparsable one adds nothing
		if staked, ok := math.NewIntFromString(sr.StakedAmount); ok && staked.IsPositive() {
			total = total.Add(staked)
		}
	}

	if electorate.DrDenom != "" {
		dr, found := k.rewardsKeeper.GetDenomReward(ctx, electorate.DrDenom)
		if found && !dr.StakedAmount.IsNil() && dr.StakedAmount.IsPositive() {
			total = total.Add(dr.StakedAmount)
		}
	}

	return total
}

// validateStakeSources checks a stake electorate against state and params: at
// most max_sr_per_dao staking rewards, every source exists, all sources share
// one staking denom (D29) and every source's lock reaches the minimum (D28).
// The stateless rules (unique ids, no members) are in ValidateBasic.
func (k Keeper) validateStakeSources(ctx sdk.Context, electorate *types.StakeElectorate, params types.Params) error {
	if electorate.IsEmpty() {
		return nil
	}

	if uint64(len(electorate.SrIds)) > params.MaxSrPerDao {
		return errorsmod.Wrapf(types.ErrInvalidElectorate, "a stake dao can have at most %d staking rewards, got %d", params.MaxSrPerDao, len(electorate.SrIds))
	}

	stakingDenom := ""
	sameDenom := func(source, denom string) error {
		if stakingDenom == "" {
			stakingDenom = denom
			return nil
		}

		if denom != stakingDenom {
			return errorsmod.Wrapf(types.ErrInvalidElectorate, "%s stakes %s, the electorate stakes %s: all sources must share one staking denom", source, denom, stakingDenom)
		}

		return nil
	}

	for _, id := range electorate.SrIds {
		sr, found := k.rewardsKeeper.GetStakingReward(ctx, id)
		if !found {
			return errorsmod.Wrapf(types.ErrInvalidElectorate, "staking reward %s not found", id)
		}

		source := "staking reward " + id
		if err := validateSourceLock(source, sr.Lock, params.MinElectorateLockDays); err != nil {
			return err
		}

		if err := sameDenom(source, sr.StakingDenom); err != nil {
			return err
		}
	}

	if electorate.DrDenom != "" {
		dr, found := k.rewardsKeeper.GetDenomReward(ctx, electorate.DrDenom)
		if !found {
			return errorsmod.Wrapf(types.ErrInvalidElectorate, "denom reward %s not found", electorate.DrDenom)
		}

		source := "denom reward " + electorate.DrDenom
		if err := validateSourceLock(source, dr.Lock, params.MinElectorateLockDays); err != nil {
			return err
		}

		if err := sameDenom(source, dr.StakingDenom); err != nil {
			return err
		}
	}

	return nil
}

// validateSourceLock requires the source's unstaking lock (days) to reach
// min_electorate_lock_days, so stake cannot leave and vote again from another
// wallet within a voting period (D2, D28). The check runs only when a source
// joins an electorate: raising the param later leaves existing DAOs as they are.
//
// Denom Rewards owned by a DAO may one day get a changeable lock; if that
// happens, re-check this rule.
func validateSourceLock(source string, lock, minLock uint32) error {
	if lock < minLock {
		return errorsmod.Wrapf(types.ErrInvalidElectorate, "%s has a %d day lock, at least %d days are required", source, lock, minLock)
	}

	return nil
}

// electorateSources lists the source keys of a stake electorate.
func electorateSources(electorate *types.StakeElectorate) [][]byte {
	if electorate.IsEmpty() {
		return nil
	}

	sources := make([][]byte, 0, len(electorate.SrIds)+1)
	for _, id := range electorate.SrIds {
		sources = append(sources, types.StakingRewardSource(id))
	}

	if electorate.DrDenom != "" {
		sources = append(sources, types.DenomRewardSource(electorate.DrDenom))
	}

	return sources
}
