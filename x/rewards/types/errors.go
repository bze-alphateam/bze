package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
)
