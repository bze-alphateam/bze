package cointrunk

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/cointrunk/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	k.SetArticleCounter(ctx, genState.ArticlesCounter)

	for _, publisher := range genState.PublisherList {
		k.SetPublisher(ctx, publisher)
	}

	for _, acceptedDomain := range genState.AcceptedDomainList {
		k.SetAcceptedDomain(ctx, acceptedDomain)
	}

	for _, article := range genState.ArticleList {
		k.SetArticle(ctx, article)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.PublisherList = k.GetAllPublisher(ctx)
	genesis.AcceptedDomainList = k.GetAllAcceptedDomain(ctx)
	genesis.ArticleList = k.GetAllArticles(ctx)
	genesis.ArticlesCounter = k.GetArticleCounter(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
