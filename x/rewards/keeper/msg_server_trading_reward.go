package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/bze-alphateam/bze/x/rewards/types"
	v2types "github.com/bze-alphateam/bze/x/rewards/v2types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	pendingExpirationPeriodInDays uint32 = 30
)

func (k msgServer) CreateTradingReward(goCtx context.Context, msg *v2types.MsgCreateTradingReward) (*v2types.MsgCreateTradingRewardResponse, error) {
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

	if !msg.PrizeAmount.IsPositive() {
		return nil, errors.Wrapf(types.ErrInvalidAmount, "amount should be greater than 0")
	}

	if msg.PrizeDenom == "" {
		return nil, types.ErrInvalidPrizeDenom
	}

	if msg.MarketId == "" {
		return nil, types.ErrInvalidMarketId
	}

	if msg.Duration == 0 {
		return nil, types.ErrInvalidDuration
	}

	if msg.Slots == 0 {
		return nil, types.ErrInvalidSlots
	}

	tradingReward := types.TradingReward{
		PrizeAmount: msg.PrizeAmount.String(),
		PrizeDenom:  msg.PrizeDenom,
		Duration:    msg.Duration,
		MarketId:    msg.MarketId,
		Slots:       msg.Slots,
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

	toCapture, err := k.getAmountToCapture(tradingReward.PrizeDenom, msg.PrizeAmount.String(), int64(msg.Slots))
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
	tradingReward.RewardId = k.smallZeroFillId(k.ReserveTradingRewardsCounter(ctx))
	tradingReward.ExpireAt = k.getNewTradingRewardExpireAt(ctx, pendingExpirationPeriodInDays)
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

	return &v2types.MsgCreateTradingRewardResponse{RewardId: tradingReward.RewardId}, nil
}
