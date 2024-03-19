package keeper

import (
	"context"
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) CreateTradingReward(goCtx context.Context, msg *types.MsgCreateTradingReward) (*types.MsgCreateTradingRewardResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	tradingReward, err := msg.ToTradingReward()
	if err != nil {
		return nil, err
	}

	//check denom
	ok := k.bankKeeper.HasSupply(ctx, tradingReward.PrizeDenom)
	if !ok {
		return nil, types.ErrInvalidPrizeDenom
	}

	if !k.tradingKeeper.MarketExists(ctx, tradingReward.MarketId) {
		return nil, types.ErrInvalidMarketId
	}

	feeParam := k.GetParams(ctx).CreateTradingRewardFee
	toCapture, err := k.getAmountToCapture(feeParam, tradingReward.PrizeDenom, tradingReward.PrizeAmount, int64(tradingReward.Slots))
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "could not calculate amount needed to create the reward")
	}

	err = k.checkUserBalances(ctx, toCapture, acc)
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture)
	if err != nil {
		return nil, err
	}

	//add ID
	tradingReward.RewardId = k.smallZeroFillId(k.GetTradingRewardsCounter(ctx))
	k.SetTradingReward(
		ctx,
		tradingReward,
	)

	return &types.MsgCreateTradingRewardResponse{RewardId: tradingReward.RewardId}, nil
}

func (k msgServer) checkUserBalances(ctx sdk.Context, neededCoins sdk.Coins, address sdk.AccAddress) error {
	spendable := k.bankKeeper.SpendableCoins(ctx, address)
	if !spendable.IsAllGTE(neededCoins) {
		return fmt.Errorf("user balance is too low")
	}

	return nil
}
