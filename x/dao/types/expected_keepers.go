package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// AccountKeeper defines the expected account keeper. DAO accounts are derived
// from the module address and created through it.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
}

// BankKeeper defines the expected bank keeper. DAO treasuries and proposal
// deposits move through it.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
}

// RewardsKeeper defines the expected rewards keeper. Stake DAOs vote with the
// stakers of its staking rewards and Denom Rewards.
type RewardsKeeper interface {
	GetStakingReward(ctx sdk.Context, rewardId string) (rewardstypes.StakingReward, bool)
	GetDenomReward(ctx sdk.Context, stakingDenom string) (rewardstypes.DenomReward, bool)
	GetStakingRewardParticipant(ctx sdk.Context, address, rewardId string) (rewardstypes.StakingRewardParticipant, bool)
	GetDenomRewardParticipant(ctx sdk.Context, stakingDenom, address string) (rewardstypes.DenomRewardParticipant, bool)
}
