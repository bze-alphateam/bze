package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// MsgRouter is the subset of baseapp.MessageRouter that daodao needs to
// dispatch a proposal's msgs[] bundle at execution time. *baseapp.MsgServiceRouter
// satisfies this interface structurally — `baseapp.MsgServiceHandler` is a
// type alias for `func(ctx sdk.Context, req sdk.Msg) (*sdk.Result, error)`,
// so a method returning either type is interface-compatible.
//
// The interface lives in expected_keepers so the keeper depends on a small
// surface (one method) rather than the full baseapp router, and so tests
// can supply a fake without pulling baseapp into their fixtures.
type MsgRouter interface {
	Handler(msg sdk.Msg) baseapp.MsgServiceHandler
}

// AccountKeeper defines the subset of x/auth's keeper that daodao needs to
// register and look up DAO BaseAccounts.
type AccountKeeper interface {
	HasAccount(ctx context.Context, addr sdk.AccAddress) bool
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the subset of x/bank's keeper that daodao needs.
//
// SpendableCoins backs balance checks (vesting/locked balances are
// excluded so users get a nice domain error instead of a lower-level
// bank error from the send call).
//
// SendCoinsFromAccountToModule routes the DAO creation fee into the
// burner module account (Epic 1), and Epic 4 reuses it to forfeit
// proposal deposits to the burner.
//
// SendCoins (Epic 4) is used by the deposit-period machinery:
//   - escrow → depositor for refunds;
//   - escrow → dao.account_address for treasury forfeit.
//
// Both addresses are BaseAccounts so SendCoins is the right primitive.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}

// DistrKeeper defines the subset of x/distribution's keeper that daodao
// needs. Used to route the DAO creation fee to the chain community pool when
// Params.dao_creation_fee_destination == "community_pool".
type DistrKeeper interface {
	FundCommunityPool(ctx context.Context, amount sdk.Coins, sender sdk.AccAddress) error
}

// RewardsKeeper defines the subset of x/rewards's keeper that daodao needs
// to support REWARD_STAKED voting backends.
//
// Methods are sdk.Context-typed to match the rewards module's own keeper
// signatures; daodao callers unwrap their context.Context before calling.
//
// Methods:
//   - GetStakingReward verifies a reward_id exists, lets us read the
//     program's StakedAmount (= TotalPower), `Lock` (flash-vote check), and
//     `Creator` (must equal the DAO's account_address — enforced when
//     MsgUpdateVotingBackend lands in Epic 5).
//   - GetStakingRewardParticipant returns an individual address's current
//     stake amount (= per-address voting power).
//   - IterateStakingRewardParticipantsByReward walks every participant of a
//     given reward program and calls `cb` for each. Returning true from
//     `cb` stops iteration. Added in Epic 3 to back the REWARD_STAKED
//     SnapshotAll path.
type RewardsKeeper interface {
	GetStakingReward(ctx sdk.Context, rewardID string) (rewardstypes.StakingReward, bool)
	GetStakingRewardParticipant(ctx sdk.Context, address, rewardID string) (rewardstypes.StakingRewardParticipant, bool)
	IterateStakingRewardParticipantsByReward(ctx sdk.Context, rewardID string, cb func(rewardstypes.StakingRewardParticipant) (stop bool))
}
