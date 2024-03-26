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
	// Set all the tradingReward
	for _, elem := range genState.PendingTradingRewardList {
		k.SetPendingTradingReward(ctx, elem)
	}

	for _, elem := range genState.ActiveTradingRewardList {
		k.SetActiveTradingReward(ctx, elem)
	}

	for _, elem := range genState.TradingRewardLeaderboardList {
		k.SetTradingRewardLeaderboard(ctx, elem)
	}

	for _, elem := range genState.TradingRewardCandidateList {
		k.SetTradingRewardCandidate(ctx, elem)
	}

	// Set all the stakingRewardParticipant
	for _, elem := range genState.StakingRewardParticipantList {
		k.SetStakingRewardParticipant(ctx, elem)
	}

	// Set all the stakingRewardParticipant
	for _, elem := range genState.PendingUnlockParticipantList {
		k.SetPendingUnlockParticipant(ctx, elem)
	}

	for _, elem := range genState.PendingTradingRewardExpirationList {
		k.SetPendingTradingRewardExpiration(ctx, elem)
	}

	for _, elem := range genState.ActiveTradingRewardExpirationList {
		k.SetActiveTradingRewardExpiration(ctx, elem)
	}

	for _, elem := range genState.MarketIdTradingRewardIdList {
		k.SetMarketIdRewardId(ctx, elem)
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

	genesis.PendingTradingRewardList = k.GetAllPendingTradingReward(ctx)
	genesis.ActiveTradingRewardList = k.GetAllActiveTradingReward(ctx)
	genesis.StakingRewardParticipantList = k.GetAllStakingRewardParticipant(ctx)
	genesis.PendingUnlockParticipantList = k.GetAllPendingUnlockParticipant(ctx)
	genesis.TradingRewardLeaderboardList = k.GetAllTradingRewardLeaderboard(ctx)
	genesis.TradingRewardCandidateList = k.GetAllTradingRewardCandidate(ctx)

	genesis.MarketIdTradingRewardIdList = k.GetAllMarketIdRewardId(ctx)
	genesis.PendingTradingRewardExpirationList = k.GetAllPendingTradingRewardExpiration(ctx)
	genesis.ActiveTradingRewardExpirationList = k.GetAllActiveTradingRewardExpiration(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
