package keeper

import (
	"context"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// UpdateVotingBackend implements MsgUpdateVotingBackend.
//
// v1 constraints (same-type only):
//   - STATIC → STATIC: rejected with a "use MsgUpdateMembers" pointer.
//     Member-set churn already has a well-named operation; routing it
//     through here would just duplicate that surface.
//   - REWARD_STAKED → REWARD_STAKED: allowed if the new reward exists and
//     its `lock` satisfies the flash-vote rule against the DAO's current
//     voting_period. NOTE: the plan also calls for verifying that the new
//     reward's creator equals dao.account_address (D15). The current
//     x/rewards.StakingReward proto doesn't carry a Creator field, so
//     that check is deferred — until x/rewards gains a Creator field via
//     a cross-module change, the practical safety comes from the natural
//     flow (a DAO swapping backends will have authored the reward via a
//     prior passed proposal, so its dispatched MsgCreateStakingReward
//     used dao.account_address as signer). The flash-vote rule below is
//     the security-critical guarantee and is enforced.
//   - Cross-type (STATIC ↔ REWARD_STAKED): rejected — deferred to a later
//     epic with explicit per-voter migration rules.
//
// Order of operations:
//  1. ValidateBasic (signer + stateless oneof checks).
//  2. assertAdmin.
//  3. Resolve the requested type and dispatch type-specific rules.
//  4. Persist Dao.VotingBackend / Dao.RewardId.
//  5. Emit event.
func (k msgServer) UpdateVotingBackend(goCtx context.Context, msg *types.MsgUpdateVotingBackend) (*types.MsgUpdateVotingBackendResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, err := k.assertAdmin(ctx, msg.DaoId, msg.Authority)
	if err != nil {
		return nil, err
	}

	switch cfg := msg.GetVotingConfig().(type) {
	case *types.MsgUpdateVotingBackend_Static:
		// Cross-type rejection takes precedence.
		if dao.VotingBackend != types.VotingBackendType_VOTING_BACKEND_STATIC {
			return nil, errorsmod.Wrapf(types.ErrBackendTypeMismatch,
				"dao %d is %s; cannot switch to STATIC", dao.Id, dao.VotingBackend)
		}
		// Same-type STATIC → "use MsgUpdateMembers" pointer.
		return nil, errorsmod.Wrap(types.ErrVotingConfigNotAllowed,
			"STATIC → STATIC reconfiguration is not supported here; use MsgUpdateMembers to change the member list")

	case *types.MsgUpdateVotingBackend_RewardStaked:
		if dao.VotingBackend != types.VotingBackendType_VOTING_BACKEND_REWARD_STAKED {
			return nil, errorsmod.Wrapf(types.ErrBackendTypeMismatch,
				"dao %d is %s; cannot switch to REWARD_STAKED", dao.Id, dao.VotingBackend)
		}
		newRewardID := cfg.RewardStaked.RewardId

		// Validate the flash-vote lock against the live rewards keeper.
		// See the comment above for the deferred ownership check.
		program, found := k.rewardsKeeper.GetStakingReward(ctx, newRewardID)
		if !found {
			return nil, errorsmod.Wrapf(types.ErrDaoNotFound,
				"reward_id %q not found", newRewardID)
		}
		// StakingReward.Lock is uint32 days (x/rewards convention; see
		// keeper/governance_config.go's parallel check).
		lock := time.Duration(program.Lock) * 24 * time.Hour
		if lock < dao.Governance.VotingPeriod {
			return nil, errorsmod.Wrapf(types.ErrFlashVoteLockTooShort,
				"reward %q lock %s < DAO voting_period %s",
				newRewardID, lock, dao.Governance.VotingPeriod)
		}

		dao.RewardId = newRewardID
		k.SetDao(ctx, dao)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeUpdateVotingBackend,
			sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", dao.Id)),
			sdk.NewAttribute(types.AttributeKeyAdmin, msg.Authority),
			sdk.NewAttribute(types.AttributeKeyVotingBackend, dao.VotingBackend.String()),
			sdk.NewAttribute(types.AttributeKeyRewardID, newRewardID),
		))
		return &types.MsgUpdateVotingBackendResponse{}, nil

	default:
		// Validate basic guards against this; defensive.
		return nil, errorsmod.Wrap(types.ErrMissingVotingConfig, "unknown voting_config")
	}
}
