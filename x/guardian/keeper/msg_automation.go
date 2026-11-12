package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// PauseAutomation stops the protocol's own automatic fund movements: the
// txfeecollector EndBlock conversions and the burner's periodic burn. The coins
// stay where they are until governance resumes the automation.
func (k msgServer) PauseAutomation(goCtx context.Context, msg *types.MsgPauseAutomation) (*types.MsgPauseAutomationResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.authorize(ctx, msg.Signer); err != nil {
		return nil, err
	}

	if k.IsAutomationPaused(ctx) {
		return nil, types.ErrAlreadyPaused
	}

	k.SetAutomationPaused(ctx, true)

	if err := ctx.EventManager().EmitTypedEvent(&types.AutomationPausedEvent{Signer: msg.Signer}); err != nil {
		return nil, err
	}

	return &types.MsgPauseAutomationResponse{}, nil
}

// ResumeAutomation lifts the pause. Only the governance authority can sign it:
// the guardian can stop the automation, never restart it.
func (k msgServer) ResumeAutomation(goCtx context.Context, msg *types.MsgResumeAutomation) (*types.MsgResumeAutomationResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if !k.isAuthority(msg.Authority) {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "only the governance authority can resume, got %s", msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if !k.IsAutomationPaused(ctx) {
		return nil, types.ErrNotPaused
	}

	k.SetAutomationPaused(ctx, false)

	if err := ctx.EventManager().EmitTypedEvent(&types.AutomationResumedEvent{Authority: msg.Authority}); err != nil {
		return nil, err
	}

	return &types.MsgResumeAutomationResponse{}, nil
}
