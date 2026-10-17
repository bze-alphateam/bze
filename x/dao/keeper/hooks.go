package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

var _ rewardstypes.StakingRewardHooks = stakingRewardHooks{}

// stakingRewardHooks receives the x/rewards staking reward notifications.
type stakingRewardHooks struct {
	k Keeper
}

// StakingRewardHooks returns the hooks the app registers on the rewards keeper.
func (k Keeper) StakingRewardHooks() rewardstypes.StakingRewardHooks {
	return stakingRewardHooks{k: k}
}

// AfterStakingRewardJoin is a no-op for now.
func (h stakingRewardHooks) AfterStakingRewardJoin(_ sdk.Context, _, _ string, _ math.Int, _ string) error {
	return nil
}

// AfterStakingRewardIncrease is a no-op for now.
func (h stakingRewardHooks) AfterStakingRewardIncrease(_ sdk.Context, _, _ string, _, _ math.Int, _ string) error {
	return nil
}

// AfterStakingRewardExit is a no-op for now.
func (h stakingRewardHooks) AfterStakingRewardExit(_ sdk.Context, _, _ string, _ math.Int, _ string) error {
	return nil
}

// BeforeStakingRewardRemoval vetoes the removal of a staking reward that a DAO
// electorate (or an open proposal) still uses.
func (h stakingRewardHooks) BeforeStakingRewardRemoval(ctx sdk.Context, rewardId string) error {
	if pins := h.k.GetSourcePins(ctx, types.StakingRewardSource(rewardId)); pins > 0 {
		return errorsmod.Wrapf(types.ErrSourcePinned, "staking reward %s is used %d time(s)", rewardId, pins)
	}

	return nil
}
