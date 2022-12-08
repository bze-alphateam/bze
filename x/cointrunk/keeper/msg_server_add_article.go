package keeper

import (
	"context"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
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
	paid := !found || publisher.Active != true
	if paid {
		articleLimit := k.AnonArticleLimit(ctx)
		existingPaidArticlesCount := k.GetMonthlyPaidArticleCounter(ctx)
		if existingPaidArticlesCount >= articleLimit {
			return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "Paid article limit reached for current period")
		}

		articleCost, err := sdk.ParseCoinsNormalized(k.AnonArticleCost(ctx))
		if err != nil {
			panic(err)
		}

		publisherAcc, err := sdk.AccAddressFromBech32(msg.Publisher)
		if err != nil {
			panic(err)
		}

		sdkErr := k.bankKeeper.SendCoinsFromAccountToModule(ctx, publisherAcc, burnermoduletypes.ModuleName, articleCost)
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

	_ = ctx

	return &types.MsgAddArticleResponse{}, nil
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

	if acceptedDomain.Active != true {
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

	if acceptedDomain.Active != true {
		return errors.Newf("Provided picture domain (%s) is NOT active", parsedUrl.Host)
	}

	return nil
}
