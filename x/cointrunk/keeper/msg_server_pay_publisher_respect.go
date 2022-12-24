package keeper

import (
	"context"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) PayPublisherRespect(goCtx context.Context, msg *types.MsgPayPublisherRespect) (*types.MsgPayPublisherRespectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	coin, err := sdk.ParseCoinNormalized(msg.Amount)
	if err != nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid amount (%s)", err)
	}
	//TODO: param to decide the denom we accept to be paid as respect
	if !coin.IsPositive() {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid coin amount (amount should be positive)")
	}

	publisher, found := k.GetPublisher(ctx, msg.Address)
	if !found {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "publisher (%s) could not be found", msg.Address)
	}

	publisherAcc, err := sdk.AccAddressFromBech32(publisher.Address)
	if err != nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "could not get publisher account (%s)", err)
	}

	creatorAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid creator account (%s)", err)
	}

	totalAmountInt := sdk.NewInt(coin.Amount.Int64())
	//TODO: param to decide the tax % we take
	taxPercent := sdk.NewDecWithPrec(20, 2) //20%
	taxAmountDec := taxPercent.MulInt(totalAmountInt)
	taxAmountInt := taxAmountDec.TruncateInt()
	if taxAmountInt.IsNegative() {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid tax amount (is negative)")
	}

	publisherAmountInt := totalAmountInt.Sub(taxAmountInt)
	if !publisherAmountInt.IsPositive() {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid publisher amount (is not positive)")
	}

	publisherRewardCoin := sdk.NewCoin(coin.Denom, publisherAmountInt)
	sdkErr := k.bankKeeper.SendCoins(ctx, creatorAcc, publisherAcc, sdk.NewCoins(publisherRewardCoin))
	if sdkErr != nil {
		return nil, sdkErr
	}

	//TODO: check the tax is > 0
	taxPaidCoin := sdk.NewCoin(coin.Denom, taxAmountInt)
	err = k.distrKeeper.FundCommunityPool(ctx, sdk.NewCoins(taxPaidCoin), creatorAcc)
	if err != nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "Could not fund community pool (%s)", err)
	}

	publisher.Respect += coin.Amount.Int64()
	k.SetPublisher(ctx, publisher)
	_ = ctx

	return &types.MsgPayPublisherRespectResponse{
		RespectPaid:        coin.Amount.Uint64(),
		PublisherReward:    publisherRewardCoin.String(),
		CommunityPoolFunds: taxPaidCoin.String(),
	}, nil
}
