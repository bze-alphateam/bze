package keeper

import (
	"context"

	"github.com/bze-alphateam/bze/x/cointrunk/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"gopkg.in/errgo.v2/fmt/errors"
)

func (k msgServer) AddArticle(goCtx context.Context, msg *types.MsgAddArticle) (*types.MsgAddArticleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	err := k.validateMessageDomains(ctx, msg)
	if err != nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid domain (%s)", err)
	}
	publisher, found := k.GetPublisher(ctx, msg.Publisher)
	paid := !found || !publisher.Active
	if paid {
		articleLimit := k.AnonArticleLimit(ctx)
		existingPaidArticlesCount := k.GetMonthlyPaidArticleCounter(ctx)
		if existingPaidArticlesCount >= articleLimit {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "Paid article limit reached for current period")
		}

		articleCost := sdk.NewCoins(k.AnonArticleCost(ctx))
		publisherAcc, err := sdk.AccAddressFromBech32(msg.Publisher)
		if err != nil {
			panic(err)
		}

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

	_ = ctx

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
		return errors.Newf("Invalid article url(%s)", err)
	}

	acceptedDomain, found := k.GetAcceptedDomain(ctx, parsedUrl.Host)
	if !found {
		return errors.Newf("Provided url domain (%s) is not an accepted domain", parsedUrl.Host)
	}

	if !acceptedDomain.Active {
		return errors.Newf("Provided url domain (%s) is NOT active", parsedUrl.Host)
	}

	//msg.Picture is optional so do not validate it unless needed
	if msg.Picture == "" {
		return nil
	}

	parsedUrl, err = msg.ParseUrl(msg.Picture)
	if err != nil {
		return errors.Newf("Invalid article picture url(%s)", err)
	}

	acceptedDomain, found = k.GetAcceptedDomain(ctx, parsedUrl.Host)
	if !found {
		return errors.Newf("Provided picture domain (%s) is not an accepted domain", parsedUrl.Host)
	}

	if !acceptedDomain.Active {
		return errors.Newf("Provided picture domain (%s) is NOT active", parsedUrl.Host)
	}

	return nil
}
