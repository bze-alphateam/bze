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

	return &types.MsgCreateTradingRewardResponse{}, nil
}

func (k msgServer) checkUserBalances(ctx sdk.Context, neededCoins sdk.Coins, address sdk.AccAddress) error {
	spendable := k.bankKeeper.SpendableCoins(ctx, address)
	if !spendable.IsAllGTE(neededCoins) {
		return fmt.Errorf("user balance is too low")
	}

	return nil
}

func (k msgServer) getAmountToCapture(feeParam, denom, amount string, prizeMultiplier int64) (sdk.Coins, error) {
	amtInt, ok := sdk.NewIntFromString(amount)
	if !ok {
		return nil, fmt.Errorf("could not convert povided amount to int: %s", amount)
	}

	toCapture := sdk.NewCoin(denom, amtInt)
	toCapture.Amount = toCapture.Amount.MulRaw(prizeMultiplier)
	if !toCapture.IsPositive() {
		//should never happen
		return nil, fmt.Errorf("calculated amount to capture is not positive")
	}

	result := sdk.NewCoins(toCapture)
	if feeParam == "" {
		return result, nil
	}

	fee, err := sdk.ParseCoinNormalized(feeParam)
	if err != nil {
		return nil, fmt.Errorf("could not parse fee param")
	}

	if !fee.IsPositive() {
		return result, nil
	}

	result = result.Add(fee)
	//just avoid any accidental panic
	if !result.IsValid() {
		return nil, fmt.Errorf("invalid amount to capture")
	}

	return result, nil
}
