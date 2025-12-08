package keeper

import (
	"context"

	"cosmossdk.io/errors"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	expirationPeriodInHours uint32 = 30 * 24
)

func (k msgServer) CreateTradingReward(goCtx context.Context, msg *types.MsgCreateTradingReward) (*types.MsgCreateTradingRewardResponse, error) {
	if k.tradeKeeper == nil {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "trade keeper is not available")
	}

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

	if !k.tradeKeeper.MarketExists(ctx, tradingReward.MarketId) {
		return nil, types.ErrInvalidMarketId
	}

	//if there is already an active reward for this market id do not allow adding another one
	_, found := k.GetMarketIdRewardId(ctx, tradingReward.MarketId)
	if found {
		return nil, types.ErrRewardAlreadyExists
	}

	toCapture, err := k.getAmountToCapture(tradingReward.PrizeDenom, tradingReward.PrizeAmount, int64(tradingReward.Slots))
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate amount needed to create the reward")
	}

	fee := k.getRewardCreationFee(ctx, k.GetParams(ctx).CreateTradingRewardFee)
	neededBalance := toCapture
	if fee != nil {
		neededBalance = neededBalance.Add(fee...)
	}

	err = k.checkUserBalances(ctx, neededBalance, acc)
	if err != nil {
		return nil, err
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture)
	if err != nil {
		return nil, err
	}

	if fee != nil {
		capturedFee, err := k.tradeKeeper.CaptureAndSwapUserFee(ctx, acc, fee, types.ModuleName)
		if err != nil {
			return nil, err
		}

		err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, capturedFee)
		if err != nil {
			return nil, err
		}
	}

	//add ID
	tradingReward.RewardId = k.smallZeroFillId(k.GetTradingRewardsCounter(ctx))
	tradingReward.ExpireAt = k.getNewTradingRewardExpireAt(ctx)
	k.SetPendingTradingReward(
		ctx,
		tradingReward,
	)

	//save expiration
	exp := types.TradingRewardExpiration{
		RewardId: tradingReward.RewardId,
		ExpireAt: tradingReward.ExpireAt,
	}
	k.SetPendingTradingRewardExpiration(ctx, exp)

	err = ctx.EventManager().EmitTypedEvent(
		&types.TradingRewardCreateEvent{
			RewardId:    tradingReward.RewardId,
			PrizeAmount: tradingReward.PrizeAmount,
			PrizeDenom:  tradingReward.PrizeDenom,
			Duration:    tradingReward.Duration,
			MarketId:    tradingReward.MarketId,
			Slots:       tradingReward.Slots,
			Creator:     msg.Creator,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgCreateTradingRewardResponse{RewardId: tradingReward.RewardId}, nil
}
