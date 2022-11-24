package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) AddArticle(goCtx context.Context, msg *types.MsgAddArticle) (*types.MsgAddArticleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	publisher, found := k.GetPublisher(ctx, msg.Publisher)
	paid := !found || publisher.Active != true
	if paid {
		articleLimit := k.AnonArticleLimit(ctx)
		existingArticlesCount := k.GetCounter(ctx)
		if existingArticlesCount >= articleLimit {
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

		sdkErr := k.bankKeeper.SendCoinsFromAccountToModule(ctx, publisherAcc, types.ModuleName, articleCost)
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
	}

	k.SetArticle(ctx, article)

	_ = ctx

	return &types.MsgAddArticleResponse{}, nil
}
