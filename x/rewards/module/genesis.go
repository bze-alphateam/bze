package rewards

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

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

	// Set all the stakingRewardParticipant. The participant reverse index is
	// not exported (derivable state) — rebuild it from the imported list.
	for _, elem := range genState.StakingRewardParticipantList {
		k.SetStakingRewardParticipant(ctx, elem)
		k.SetStakingRewardParticipantIndexEntry(ctx, elem.RewardId, elem.Address)
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

	// Set all the boost records and their participant entries
	for _, elem := range genState.BoostList {
		k.SetBoost(ctx, elem)
	}

	for _, elem := range genState.BoostParticipantList {
		k.SetBoostParticipant(ctx, elem)
	}

	// Restore queue states
	if genState.UnlockParticipantsQueue != nil {
		k.SetUnlockParticipantsQueue(ctx, *genState.UnlockParticipantsQueue)
	}

	if genState.StakingRewardsDistributionQueue != nil {
		k.SetStakingRewardsDistributionQueue(ctx, *genState.StakingRewardsDistributionQueue)
	}

	if genState.TradingRewardExpirationQueue != nil {
		k.SetTradingRewardExpirationQueue(ctx, *genState.TradingRewardExpirationQueue)
	}

	// this line is used by starport scaffolding # genesis/module/init
	k.SetTradingRewardsCounter(ctx, genState.TradingRewardsCounter)
	k.SetStakingRewardsCounter(ctx, genState.StakingRewardsCounter)
	k.SetBoostsCounter(ctx, genState.BoostCounter)
}

// ExportGenesis returns the module's exported genesis.
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

	genesis.BoostList = k.GetAllBoosts(ctx)
	genesis.BoostParticipantList = k.GetAllLiveBoostParticipant(ctx)
	genesis.BoostCounter = k.GetBoostsCounter(ctx)
	genesis.PendingTradingRewardExpirationList = k.GetAllPendingTradingRewardExpiration(ctx)
	genesis.ActiveTradingRewardExpirationList = k.GetAllActiveTradingRewardExpiration(ctx)

	// Export queue states
	if unlockQueue, found := k.GetUnlockParticipantsQueue(ctx); found {
		genesis.UnlockParticipantsQueue = &unlockQueue
	}

	if stakingDistQueue, found := k.GetStakingRewardsDistributionQueue(ctx); found {
		genesis.StakingRewardsDistributionQueue = &stakingDistQueue
	}

	if tradingExpQueue, found := k.GetTradingRewardExpirationQueue(ctx); found {
		genesis.TradingRewardExpirationQueue = &tradingExpQueue
	}

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
