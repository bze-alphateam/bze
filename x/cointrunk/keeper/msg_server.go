package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"

	"cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) AddArticle(goCtx context.Context, msg *types.MsgAddArticle) (*types.MsgAddArticleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	err := k.validateMessageDomains(ctx, msg)
	if err != nil {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid domain (%s)", err)
	}
	publisher, found := k.GetPublisher(ctx, msg.Publisher)
	paid := !found || !publisher.Active
	if paid {
		articleLimit := k.AnonArticleLimit(ctx)
		existingPaidArticlesCount := k.GetMonthlyPaidArticleCounter(ctx)
		if existingPaidArticlesCount >= articleLimit {
			return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "Paid article limit reached for current period")
		}

		articleCost := sdk.NewCoins(k.AnonArticleCost(ctx))
		publisherAcc := sdk.MustAccAddressFromBech32(msg.Publisher)
		sdkErr := k.distrKeeper.FundCommunityPool(ctx, articleCost, publisherAcc)
		if sdkErr != nil {
			return nil, sdkErr
		}
	}

	var article = types.Article{
		Title:     msg.Title,
		Url:       msg.Url,
		Picture:   msg.Picture,
		Publisher: msg.Publisher,
		Paid:      paid,
		Id:        0,
		CreatedAt: ctx.BlockHeader().Time.Unix(),
	}

	k.SetArticle(ctx, article)

	if found {
		publisher.ArticlesCount += 1
		k.SetPublisher(ctx, publisher)
	}

	err = k.emitArticleAddedEvent(ctx, article)
	if err != nil {
		return nil, err
	}

	return &types.MsgAddArticleResponse{}, nil
}

func (k msgServer) emitArticleAddedEvent(ctx sdk.Context, article types.Article) error {
	return ctx.EventManager().EmitTypedEvent(
		&types.ArticleAddedEvent{
			ArticleId: article.Id,
			Publisher: article.Publisher,
			Paid:      article.Paid,
		},
	)
}

func (k msgServer) validateMessageDomains(ctx sdk.Context, msg *types.MsgAddArticle) error {
	parsedUrl, err := msg.ParseUrl(msg.Url)
	if err != nil {
		return fmt.Errorf("invalid article url(%s)", err)
	}

	acceptedDomain, found := k.GetAcceptedDomain(ctx, parsedUrl.Host)
	if !found {
		return fmt.Errorf("provided url domain (%s) is not an accepted domain", parsedUrl.Host)
	}

	if !acceptedDomain.Active {
		return fmt.Errorf("provided url domain (%s) is NOT active", parsedUrl.Host)
	}

	//msg.Picture is optional so do not validate it unless needed
	if msg.Picture == "" {
		return nil
	}

	parsedUrl, err = msg.ParseUrl(msg.Picture)
	if err != nil {
		return fmt.Errorf("invalid article picture url(%s)", err)
	}

	acceptedDomain, found = k.GetAcceptedDomain(ctx, parsedUrl.Host)
	if !found {
		return fmt.Errorf("provided picture domain (%s) is not an accepted domain", parsedUrl.Host)
	}

	if !acceptedDomain.Active {
		return fmt.Errorf("provided picture domain (%s) is NOT active", parsedUrl.Host)
	}

	return nil
}

func (k msgServer) PayPublisherRespect(goCtx context.Context, msg *types.MsgPayPublisherRespect) (*types.MsgPayPublisherRespectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	coin, err := sdk.ParseCoinNormalized(msg.Amount)
	if err != nil {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid amount (%s)", err)
	}

	publisherRespectParams := k.PublisherRespectParams(ctx)
	if coin.Denom != publisherRespectParams.Denom {
		return nil, errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"invalid coin denom. Accepted (%s) got (%s)",
			publisherRespectParams.Denom,
			coin.Denom,
		)
	}

	if !coin.IsPositive() {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "invalid coin amount (amount should be positive)")
	}

	publisher, found := k.GetPublisher(ctx, msg.Address)
	if !found {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "publisher (%s) could not be found", msg.Address)
	}

	publisherAcc, err := sdk.AccAddressFromBech32(publisher.Address)
	if err != nil {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "could not get publisher account (%s)", err)
	}

	creatorAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid creator account (%s)", err)
	}

	totalAmountInt := coin.Amount
	taxPercent := publisherRespectParams.Tax
	taxAmountDec := taxPercent.MulInt(totalAmountInt)
	taxAmountInt := taxAmountDec.TruncateInt()
	if taxAmountInt.IsNegative() {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "invalid tax amount (is negative)")
	}

	publisherAmountInt := totalAmountInt.Sub(taxAmountInt)
	if !publisherAmountInt.IsPositive() {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "invalid publisher amount (is not positive)")
	}

	publisherRewardCoin := sdk.NewCoin(coin.Denom, publisherAmountInt)
	sdkErr := k.bankKeeper.SendCoins(ctx, creatorAcc, publisherAcc, sdk.NewCoins(publisherRewardCoin))
	if sdkErr != nil {
		return nil, sdkErr
	}

	taxPaidCoin := sdk.NewCoin(coin.Denom, taxAmountInt)
	if !taxPaidCoin.IsZero() {
		err = k.distrKeeper.FundCommunityPool(ctx, sdk.NewCoins(taxPaidCoin), creatorAcc)
		if err != nil {
			return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "Could not fund community pool (%s)", err)
		}
	}

	respInt, ok := math.NewIntFromString(publisher.Respect)
	if !ok {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "Could not parse publisher respect")
	}

	respInt = respInt.Add(coin.Amount)
	publisher.Respect = respInt.String()
	k.SetPublisher(ctx, publisher)

	err = ctx.EventManager().EmitTypedEvent(&types.PublisherRespectPaidEvent{
		Publisher:          publisher.Address,
		RespectPaid:        coin.Amount.Uint64(),
		CommunityPoolFunds: taxPaidCoin.Amount.Uint64(),
		PublisherReward:    publisherRewardCoin.Amount.Uint64(),
	})
	if err != nil {
		return nil, err
	}

	return &types.MsgPayPublisherRespectResponse{
		RespectPaid:        coin.Amount.Uint64(),
		PublisherReward:    publisherRewardCoin.Amount.Uint64(),
		CommunityPoolFunds: taxPaidCoin.Amount.Uint64(),
	}, nil
}
