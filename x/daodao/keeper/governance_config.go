package keeper

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// GetGovernanceConfig returns the DAO's current governance configuration.
// Convenience wrapper around GetDao; included so callers don't keep
// re-reading the full DAO record just for governance.
func (k Keeper) GetGovernanceConfig(ctx context.Context, daoID uint64) (types.GovernanceConfig, bool) {
	dao, ok := k.GetDao(ctx, daoID)
	if !ok {
		return types.GovernanceConfig{}, false
	}
	return dao.Governance, true
}

// validateGovernanceAgainstChainState applies the Param- and reward-keeper-
// dependent checks that can't run in stateless ValidateBasic:
//
//   - voting_period <= Params.max_voting_period (chain-tunable ceiling).
//   - For REWARD_STAKED DAOs, StakingReward.lock >= voting_period
//     (flash-vote rule, D16 in README).
//
// Called by MsgCreateDao (after computing the DAO's intended backend) and
// by MsgUpdateGovernanceConfig.
func (k Keeper) validateGovernanceAgainstChainState(ctx context.Context, dao types.Dao, g types.GovernanceConfig) error {
	params := k.GetParams(ctx)
	if err := types.ValidateGovernanceConfigAgainstParams(g, params.MaxVotingPeriod); err != nil {
		return err
	}
	if dao.VotingBackend == types.VotingBackendType_VOTING_BACKEND_REWARD_STAKED {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		program, found := k.rewardsKeeper.GetStakingReward(sdkCtx, dao.RewardId)
		if !found {
			return errorsmod.Wrapf(types.ErrDaoNotFound,
				"REWARD_STAKED DAO %d references missing reward_id %q", dao.Id, dao.RewardId)
		}
		// StakingReward.Lock is uint32 days (x/rewards stores days; see the
		// `lockedUntil += int64(sr.Lock) * 24` arithmetic in rewards'
		// beginUnlock). Convert to a Duration for the comparison.
		lock := time.Duration(program.Lock) * 24 * time.Hour
		if lock < g.VotingPeriod {
			return errorsmod.Wrapf(types.ErrFlashVoteLockTooShort,
				"reward %q lock %s < DAO voting_period %s", dao.RewardId, lock, g.VotingPeriod)
		}
	}
	return nil
}
