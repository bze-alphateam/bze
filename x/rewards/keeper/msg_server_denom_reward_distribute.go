package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// DistributeDenomRewards is the instant distribution (airdrop): the amount is escrowed from the
// sender and applied to the (staking denom, prize denom) accumulator immediately, in the same tx
// (Business Logic rule 18). It is permissionless and free when the prize denom already has an
// accumulator on the DR; introducing a NEW prize denom pays the DR-prize creation fee through
// ensureDenomRewardPrize — the only fee an airdrop can ever pay. A DR with zero stakers rejects
// the airdrop (rule 19 / invariant I6): with T = 0 there is nobody to credit and the escrowed
// funds would be stranded forever.
func (k msgServer) DistributeDenomRewards(goCtx context.Context, msg *types.MsgDistributeDenomRewards) (*types.MsgDistributeDenomRewardsResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	dr, found := k.GetDenomReward(ctx, msg.Denom)
	if !found {
		return nil, types.ErrDenomRewardNotFound
	}

	if !dr.StakedAmount.IsPositive() {
		return nil, types.ErrNoStakersInDenomReward
	}

	toCapture, err := denomAmountToCapture(msg.PrizeDenom, msg.Amount, 1)
	if err != nil {
		return nil, err
	}

	if err = k.checkUserBalances(ctx, toCapture, acc); err != nil {
		return nil, err
	}

	// existing prize denom → returned free; new prize denom → cap check + prize creation fee
	prize, err := k.ensureDenomRewardPrize(ctx, dr.StakingDenom, msg.PrizeDenom, acc)
	if err != nil {
		return nil, err
	}

	// escrow the airdrop, then credit it to the accumulator immediately: every staker's pending
	// grows by amount/T in the same tx
	if err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture); err != nil {
		return nil, err
	}

	if err = k.distributeToDenomPrize(ctx, prize, toCapture.AmountOf(msg.PrizeDenom), dr.StakedAmount); err != nil {
		return nil, err
	}

	err = ctx.EventManager().EmitTypedEvent(
		&types.DenomRewardDistributionEvent{
			Denom:      dr.StakingDenom,
			PrizeDenom: msg.PrizeDenom,
			Amount:     toCapture.AmountOf(msg.PrizeDenom).String(),
		},
	)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgDistributeDenomRewardsResponse{}, nil
}
