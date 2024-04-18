package keeper

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) HandleActivateTradingRewardProposal(ctx sdk.Context, proposal *types.ActivateTradingRewardProposal) error {
	//check if already active
	_, found := k.GetActiveTradingReward(ctx, proposal.RewardId)
	if found {
		return fmt.Errorf("trading reward already active")
	}

	//check if the reward exists and is pending
	r, found := k.GetPendingTradingReward(ctx, proposal.RewardId)
	if !found {
		return fmt.Errorf("trading reward not found")
	}

	//remove the pending reward expiration because we're moving this one to active trading reward
	k.RemovePendingTradingRewardExpiration(ctx, r.ExpireAt, r.RewardId)
	k.RemovePendingTradingReward(ctx, r.RewardId)

	//move the trading reward to active
	r.ExpireAt = k.getNewTradingRewardExpireAt(ctx)
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

	k.Logger(ctx).Info(fmt.Sprintf("trading reward with id [%s] has been activated", r.RewardId))

	return nil
}
