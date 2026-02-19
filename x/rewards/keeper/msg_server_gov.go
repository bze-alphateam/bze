package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) ActivateTradingReward(goCtx context.Context, msg *types.MsgActivateTradingReward) (*types.MsgActivateTradingRewardResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	if k.GetAuthority() != msg.GetCreator() {
		return nil, errors.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.GetCreator())
	}

	//check if already active
	_, found := k.GetActiveTradingReward(ctx, msg.RewardId)
	if found {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "trading reward already active")
	}

	//check if the reward exists and is pending
	r, found := k.GetPendingTradingReward(ctx, msg.RewardId)
	if !found {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "trading reward not found")
	}

	//remove the pending reward expiration because we're moving this one to active trading reward
	k.RemovePendingTradingRewardExpiration(ctx, r.ExpireAt, r.RewardId)
	k.RemovePendingTradingReward(ctx, r.RewardId)

	//move the trading reward to active
	r.ExpireAt = k.getNewTradingRewardExpireAt(ctx, r.Duration)
	k.SetActiveTradingReward(ctx, r)

	//save expiration
	exp := types.TradingRewardExpiration{
		RewardId: r.RewardId,
		ExpireAt: r.ExpireAt,
	}
	k.SetActiveTradingRewardExpiration(ctx, exp)

	//save the market id, so we can find the reward by market id
	k.SetMarketIdRewardId(ctx, types.MarketIdTradingRewardId{
		RewardId: r.RewardId,
		MarketId: r.MarketId,
	})

	err := ctx.EventManager().EmitTypedEvent(
		&types.TradingRewardActivationEvent{
			RewardId: exp.RewardId,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	k.Logger().Info(fmt.Sprintf("trading reward with id [%s] has been activated", r.RewardId))

	return &types.MsgActivateTradingRewardResponse{}, nil
}
