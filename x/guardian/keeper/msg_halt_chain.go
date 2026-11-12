package keeper

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// SetUpgradeKeeper wires the x/upgrade keeper. The app sets it after the
// depinject build, like the rate-limiting keeper.
func (k *Keeper) SetUpgradeKeeper(upgradeKeeper types.UpgradeKeeper) {
	k.upgradeKeeper = upgradeKeeper
}

// HaltChain schedules an upgrade plan at the next block height, the earliest
// one x/upgrade accepts in a transaction. Every node stops at that height with
// the usual "UPGRADE NEEDED" message until it runs a binary with an upgrade
// handler for the plan name, or until the validators skip the height with
// --unsafe-skip-upgrades. x/upgrade keeps one plan, so this replaces any plan
// already scheduled. There is no message to undo it: a cancel would need a
// governance proposal to pass within the same block.
func (k msgServer) HaltChain(goCtx context.Context, msg *types.MsgHaltChain) (*types.MsgHaltChainResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.authorize(ctx, msg.Signer); err != nil {
		return nil, err
	}

	if k.upgradeKeeper == nil {
		return nil, types.ErrUpgradeKeeperNotSet
	}

	height := ctx.BlockHeight() + 1
	plan := upgradetypes.Plan{
		Name:   types.HaltPlanName(height),
		Height: height,
		Info:   msg.Reason,
	}
	if err := k.upgradeKeeper.ScheduleUpgrade(ctx, plan); err != nil {
		return nil, err
	}

	if err := ctx.EventManager().EmitTypedEvent(&types.ChainHaltScheduledEvent{
		Signer: msg.Signer,
		Name:   plan.Name,
		Height: plan.Height,
		Reason: msg.Reason,
	}); err != nil {
		return nil, err
	}

	k.Logger().Info("chain halt scheduled", "signer", msg.Signer, "height", plan.Height, "name", plan.Name, "reason", msg.Reason)

	return &types.MsgHaltChainResponse{Name: plan.Name, Height: plan.Height}, nil
}
