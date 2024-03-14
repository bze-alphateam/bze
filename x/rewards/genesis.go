package rewards

import (
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the stakingReward
	for _, elem := range genState.StakingRewardList {
		k.SetStakingReward(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
	k.SetTradingRewardsCounter(ctx, genState.TradingRewardsCounter)
	k.SetStakingRewardsCounter(ctx, genState.StakingRewardsCounter)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.StakingRewardList = k.GetAllStakingReward(ctx)
	genesis.StakingRewardsCounter = k.GetStakingRewardsCounter(ctx)
	genesis.TradingRewardsCounter = k.GetTradingRewardsCounter(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
