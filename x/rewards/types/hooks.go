package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingRewardHooks - event hooks called by the rewards module when a user's
// staking reward participation changes. Hooks run synchronously in the tx that
// changed the participation, after the module's own state writes. A non-nil
// error from the After* hooks aborts the whole transaction.
type StakingRewardHooks interface {
	// AfterStakingRewardJoin - the address became a new participant of the reward.
	AfterStakingRewardJoin(ctx sdk.Context, rewardId, address string, amount math.Int, stakingDenom string) error

	// AfterStakingRewardIncrease - an existing participant staked more.
	AfterStakingRewardIncrease(ctx sdk.Context, rewardId, address string, amountAdded, newTotal math.Int, stakingDenom string) error

	// AfterStakingRewardExit - the participant exited, removing their entire stake.
	AfterStakingRewardExit(ctx sdk.Context, rewardId, address string, unstakedAmount math.Int, stakingDenom string) error

	// BeforeStakingRewardRemoval - called before a staking reward record is
	// deleted; a non-nil error stops the deletion, keeping the record alive.
	BeforeStakingRewardRemoval(ctx sdk.Context, rewardId string) error
}

// MultiStakingRewardHooks - combines multiple StakingRewardHooks. Hooks are
// executed in the order they are provided; the first error stops execution.
type MultiStakingRewardHooks []StakingRewardHooks

var _ StakingRewardHooks = MultiStakingRewardHooks{}

func NewMultiStakingRewardHooks(hooks ...StakingRewardHooks) MultiStakingRewardHooks {
	return hooks
}

func (h MultiStakingRewardHooks) AfterStakingRewardJoin(ctx sdk.Context, rewardId, address string, amount math.Int, stakingDenom string) error {
	for i := range h {
		if err := h[i].AfterStakingRewardJoin(ctx, rewardId, address, amount, stakingDenom); err != nil {
			return err
		}
	}

	return nil
}

func (h MultiStakingRewardHooks) AfterStakingRewardIncrease(ctx sdk.Context, rewardId, address string, amountAdded, newTotal math.Int, stakingDenom string) error {
	for i := range h {
		if err := h[i].AfterStakingRewardIncrease(ctx, rewardId, address, amountAdded, newTotal, stakingDenom); err != nil {
			return err
		}
	}

	return nil
}

func (h MultiStakingRewardHooks) AfterStakingRewardExit(ctx sdk.Context, rewardId, address string, unstakedAmount math.Int, stakingDenom string) error {
	for i := range h {
		if err := h[i].AfterStakingRewardExit(ctx, rewardId, address, unstakedAmount, stakingDenom); err != nil {
			return err
		}
	}

	return nil
}

func (h MultiStakingRewardHooks) BeforeStakingRewardRemoval(ctx sdk.Context, rewardId string) error {
	for i := range h {
		if err := h[i].BeforeStakingRewardRemoval(ctx, rewardId); err != nil {
			return err
		}
	}

	return nil
}

// DenomRewardHooks - event hooks called by the rewards module when a user's
// Denom Reward participation changes. Same contract as StakingRewardHooks:
// they run synchronously after the module's own state writes, and a non-nil
// error aborts the whole transaction. There is no removal hook: Denom Rewards
// are never deleted.
type DenomRewardHooks interface {
	// AfterDenomRewardJoin - the address became a new participant of the Denom Reward.
	AfterDenomRewardJoin(ctx sdk.Context, stakingDenom, address string, amount math.Int) error

	// AfterDenomRewardIncrease - an existing participant staked more.
	AfterDenomRewardIncrease(ctx sdk.Context, stakingDenom, address string, amountAdded, newTotal math.Int) error

	// AfterDenomRewardExit - the participant exited, removing their entire stake.
	AfterDenomRewardExit(ctx sdk.Context, stakingDenom, address string, unstakedAmount math.Int) error
}

// MultiDenomRewardHooks - combines multiple DenomRewardHooks. Hooks are
// executed in the order they are provided; the first error stops execution.
type MultiDenomRewardHooks []DenomRewardHooks

var _ DenomRewardHooks = MultiDenomRewardHooks{}

func NewMultiDenomRewardHooks(hooks ...DenomRewardHooks) MultiDenomRewardHooks {
	return hooks
}

func (h MultiDenomRewardHooks) AfterDenomRewardJoin(ctx sdk.Context, stakingDenom, address string, amount math.Int) error {
	for i := range h {
		if err := h[i].AfterDenomRewardJoin(ctx, stakingDenom, address, amount); err != nil {
			return err
		}
	}

	return nil
}

func (h MultiDenomRewardHooks) AfterDenomRewardIncrease(ctx sdk.Context, stakingDenom, address string, amountAdded, newTotal math.Int) error {
	for i := range h {
		if err := h[i].AfterDenomRewardIncrease(ctx, stakingDenom, address, amountAdded, newTotal); err != nil {
			return err
		}
	}

	return nil
}

func (h MultiDenomRewardHooks) AfterDenomRewardExit(ctx sdk.Context, stakingDenom, address string, unstakedAmount math.Int) error {
	for i := range h {
		if err := h[i].AfterDenomRewardExit(ctx, stakingDenom, address, unstakedAmount); err != nil {
			return err
		}
	}

	return nil
}
