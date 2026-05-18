package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// UpdateDepositConfig implements MsgUpdateDepositConfig.
//
// Order of operations:
//  1. ValidateBasic (signer + dao_id + stateless caps on the deposit cfg).
//  2. assertAdmin — only the DAO admin may update.
//  3. Validate against Params (deposit_period upper bound).
//  4. Replace dao.deposit, persist.
//
// Existing proposals retain their own deposit_snapshot; this change
// applies to NEW proposals only.
func (k msgServer) UpdateDepositConfig(goCtx context.Context, msg *types.MsgUpdateDepositConfig) (*types.MsgUpdateDepositConfigResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, err := k.assertAdmin(ctx, msg.DaoId, msg.Authority)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	if err := types.ValidateDepositConfigAgainstParams(msg.Deposit, params.MaxDepositPeriod); err != nil {
		return nil, err
	}

	dao.Deposit = msg.Deposit
	k.SetDao(ctx, dao)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateDepositConfig,
		sdk.NewAttribute(types.AttributeKeyDaoID, fmt.Sprintf("%d", dao.Id)),
		sdk.NewAttribute(types.AttributeKeyAdmin, msg.Authority),
		sdk.NewAttribute(types.AttributeKeyMinDeposit, msg.Deposit.MinDeposit.String()),
		sdk.NewAttribute(types.AttributeKeyDepositPeriod, msg.Deposit.DepositPeriod.String()),
		sdk.NewAttribute(types.AttributeKeyForfeitDest, msg.Deposit.ForfeitDestination.String()),
		sdk.NewAttribute(types.AttributeKeyRefundPolicy, msg.Deposit.VotingRefundPolicy.String()),
	))

	return &types.MsgUpdateDepositConfigResponse{}, nil
}
