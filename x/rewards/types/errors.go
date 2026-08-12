package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/rewards module sentinel errors
var (
	ErrInvalidAmount       = sdkerrors.Register(ModuleName, 5000, "invalid amount")
	ErrInvalidPrizeDenom   = sdkerrors.Register(ModuleName, 5002, "invalid prize denom")
	ErrInvalidStakingDenom = sdkerrors.Register(ModuleName, 5003, "invalid staking denom")
	ErrInvalidMinStake     = sdkerrors.Register(ModuleName, 5004, "invalid min stake")
	ErrInvalidDuration     = sdkerrors.Register(ModuleName, 5005, "invalid duration")
	ErrInvalidLockingTime  = sdkerrors.Register(ModuleName, 5006, "invalid staking reward lock")
	ErrInvalidMarketId     = sdkerrors.Register(ModuleName, 5007, "invalid market_id")
	ErrInvalidSlots        = sdkerrors.Register(ModuleName, 5008, "invalid slots")
	ErrInvalidRewardId     = sdkerrors.Register(ModuleName, 5009, "invalid reward_id")
	ErrRewardAlreadyExists = sdkerrors.Register(ModuleName, 5010, "a reward is already running for this market")
	ErrInvalidSigner       = sdkerrors.Register(ModuleName, 5011, "invalid signer")
	ErrNoRewardsToClaim    = sdkerrors.Register(ModuleName, 5012, "no rewards available to claim")

	ErrStakingRewardNotFinished = sdkerrors.Register(ModuleName, 5013, "staking reward is not finished")
	ErrStakingRewardNotEmpty    = sdkerrors.Register(ModuleName, 5014, "staking reward still has staked funds: stakers must exit first")

	// Denom Rewards errors
	ErrDenomRewardNotFound    = sdkerrors.Register(ModuleName, 5015, "denom reward not found")
	ErrDenomRewardExists      = sdkerrors.Register(ModuleName, 5016, "a denom reward already exists for this denom")
	ErrPrizeDenomCapReached   = sdkerrors.Register(ModuleName, 5017, "prize denom cap reached for this denom reward")
	ErrNoStakersInDenomReward = sdkerrors.Register(ModuleName, 5018, "denom reward has no stakers")
	ErrInvalidScheduleId      = sdkerrors.Register(ModuleName, 5019, "invalid schedule_id")
)
