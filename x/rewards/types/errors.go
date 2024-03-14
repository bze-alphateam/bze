package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/rewards module sentinel errors
var (
	ErrInvalidPrizeAmount  = sdkerrors.Register(ModuleName, 5000, "invalid prize amount")
	ErrInvalidPrizeDenom   = sdkerrors.Register(ModuleName, 5002, "invalid prize denom")
	ErrInvalidStakingDenom = sdkerrors.Register(ModuleName, 5003, "invalid staking denom")
	ErrInvalidMinStake     = sdkerrors.Register(ModuleName, 5004, "invalid min stake")
	ErrInvalidDuration     = sdkerrors.Register(ModuleName, 5005, "invalid duration")
	ErrInvalidLockingTime  = sdkerrors.Register(ModuleName, 5006, "invalid staking reward lock")
)
