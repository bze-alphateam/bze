package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

var (
	_ rewardstypes.StakingRewardHooks = stakingRewardHooks{}
	_ rewardstypes.DenomRewardHooks   = denomRewardHooks{}
)

// stakingRewardHooks receives the x/rewards staking reward notifications.
type stakingRewardHooks struct {
	k Keeper
}

// StakingRewardHooks returns the hooks the app registers on the rewards keeper.
func (k Keeper) StakingRewardHooks() rewardstypes.StakingRewardHooks {
	return stakingRewardHooks{k: k}
}

// AfterStakingRewardJoin records when the address joined a pinned staking reward.
func (h stakingRewardHooks) AfterStakingRewardJoin(ctx sdk.Context, rewardId, address string, _ math.Int, _ string) error {
	h.k.afterSourceJoin(ctx, types.StakingRewardSource(rewardId), address)
	return nil
}

// AfterStakingRewardIncrease moves the join height to the top-up's block, as
// a new join, so stake added during a vote cannot weigh on proposals already
// open.
func (h stakingRewardHooks) AfterStakingRewardIncrease(ctx sdk.Context, rewardId, address string, _, _ math.Int, _ string) error {
	h.k.afterSourceJoin(ctx, types.StakingRewardSource(rewardId), address)
	return nil
}

// AfterStakingRewardExit forgets the address's join record and takes its
// share out of its open ballots.
func (h stakingRewardHooks) AfterStakingRewardExit(ctx sdk.Context, rewardId, address string, _ math.Int, _ string) error {
	h.k.afterSourceExit(ctx, types.StakingRewardSource(rewardId), address)
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

// denomRewardHooks receives the x/rewards Denom Reward notifications.
type denomRewardHooks struct {
	k Keeper
}

// DenomRewardHooks returns the hooks the app registers on the rewards keeper.
func (k Keeper) DenomRewardHooks() rewardstypes.DenomRewardHooks {
	return denomRewardHooks{k: k}
}

// AfterDenomRewardJoin records when the address joined a pinned Denom Reward.
func (h denomRewardHooks) AfterDenomRewardJoin(ctx sdk.Context, stakingDenom, address string, _ math.Int) error {
	h.k.afterSourceJoin(ctx, types.DenomRewardSource(stakingDenom), address)
	return nil
}

// AfterDenomRewardIncrease moves the join height to the top-up's block, as a
// new join, so stake added during a vote cannot weigh on proposals already
// open.
func (h denomRewardHooks) AfterDenomRewardIncrease(ctx sdk.Context, stakingDenom, address string, _, _ math.Int) error {
	h.k.afterSourceJoin(ctx, types.DenomRewardSource(stakingDenom), address)
	return nil
}

// AfterDenomRewardExit forgets the address's join record and takes its share
// out of its open ballots.
func (h denomRewardHooks) AfterDenomRewardExit(ctx sdk.Context, stakingDenom, address string, _ math.Int) error {
	h.k.afterSourceExit(ctx, types.DenomRewardSource(stakingDenom), address)
	return nil
}

// afterSourceJoin writes the join height, after a join or a top-up, when a DAO
// or an open proposal pins the source; stake joining an unused source needs no
// record. A ballot already cast keeps its weight until the voter votes again. The After*
// hooks abort the x/rewards transaction on error, so this never fails: an
// address x/rewards accepted but this module can't parse is skipped.
func (k Keeper) afterSourceJoin(ctx sdk.Context, source []byte, address string) {
	if k.GetSourcePins(ctx, source) == 0 {
		return
	}

	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		k.Logger().Error("dao join hook: invalid address", "address", address, "error", err)
		return
	}

	k.setJoinRecord(ctx, source, addr, ctx.BlockHeight())
}

// afterSourceExit deletes the join record, if any, and removes the exited
// stake from the address's ballots on proposals in voting (D2, D5).
func (k Keeper) afterSourceExit(ctx sdk.Context, source []byte, address string) {
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		k.Logger().Error("dao exit hook: invalid address", "address", address, "error", err)
		return
	}

	k.removeJoinRecord(ctx, source, addr)
	k.reduceBallots(ctx, source, addr)
}
