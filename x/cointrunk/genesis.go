package cointrunk

import (
	"github.com/bze-alphateam/bze/x/cointrunk/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
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

	// this line is used by starport scaffolding # genesis/module/init
}

// ExportGenesis returns the capability module's exported genesis.
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
