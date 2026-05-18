package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// UpdateGovernanceConfig implements MsgUpdateGovernanceConfig.
//
// Order of operations:
//  1. ValidateBasic (signer + dao_id + stateless caps on the config).
//  2. assertAdmin — only the DAO's current admin may update.
//  3. Validate the new config against chain state (Params.max_voting_period
//     and, for REWARD_STAKED DAOs, the flash-vote lock rule).
//  4. Replace dao.governance, persist.
//
// Existing proposals retain their own governance_snapshot; this change
// applies to NEW proposals only.
func (k msgServer) UpdateGovernanceConfig(goCtx context.Context, msg *types.MsgUpdateGovernanceConfig) (*types.MsgUpdateGovernanceConfigResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, err := k.assertAdmin(ctx, msg.DaoId, msg.Authority)
	if err != nil {
		return nil, err
	}

	if err := k.validateGovernanceAgainstChainState(ctx, dao, msg.Governance); err != nil {
		return nil, err
	}

	dao.Governance = msg.Governance
	k.SetDao(ctx, dao)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateGovernanceConfig,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", dao.Id)),
		sdk.NewAttribute(types.AttributeKeyAdmin, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyApprovalRule, msg.Governance.ApprovalRule.String()),
		sdk.NewAttribute(types.AttributeKeyThresholdBps, fmt.Sprintf("%d", msg.Governance.ThresholdBps)),
		sdk.NewAttribute(types.AttributeKeyQuorumBps, fmt.Sprintf("%d", msg.Governance.QuorumBps)),
		sdk.NewAttribute(types.AttributeKeyVotingPeriod, msg.Governance.VotingPeriod.String()),
		sdk.NewAttribute(types.AttributeKeyAllowRevote, fmt.Sprintf("%t", msg.Governance.AllowRevote)),
	))

	return &types.MsgUpdateGovernanceConfigResponse{}, nil
}
