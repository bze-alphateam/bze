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

	// Denom Rewards state
	for _, elem := range genState.DenomRewardList {
		k.SetDenomReward(ctx, elem)
	}

	for _, elem := range genState.DenomRewardPrizeList {
		k.SetDenomRewardPrize(ctx, elem)
	}

	// SetDenomRewardParticipant also writes the address-first drp/a/ marker, so the
	// marker index (derivable state, not exported) is rebuilt here.
	for _, elem := range genState.DenomRewardParticipantList {
		k.SetDenomRewardParticipant(ctx, elem)
	}

	for _, elem := range genState.DenomRewardParticipantIndexList {
		k.SetDenomRewardParticipantIndex(ctx, elem)
	}

	for _, elem := range genState.DenomRewardScheduleList {
		k.SetDenomRewardSchedule(ctx, elem)
	}

	if genState.DenomRewardsDistributionQueue != nil {
		k.SetDenomRewardsDistributionQueue(ctx, *genState.DenomRewardsDistributionQueue)
	}

	k.SetDenomRewardScheduleCounter(ctx, genState.DenomRewardScheduleCounter)

	// this line is used by starport scaffolding # genesis/module/init
	k.SetTradingRewardsCounter(ctx, genState.TradingRewardsCounter)
	k.SetStakingRewardsCounter(ctx, genState.StakingRewardsCounter)
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

	// Denom Rewards state. The drp/a/ marker index is NOT exported: it is derivable
	// state, rebuilt by SetDenomRewardParticipant on import.
	genesis.DenomRewardList = k.GetAllDenomReward(ctx)
	genesis.DenomRewardPrizeList = k.GetAllDenomRewardPrize(ctx)
	genesis.DenomRewardParticipantList = k.GetAllDenomRewardParticipant(ctx)
	genesis.DenomRewardParticipantIndexList = k.GetAllDenomRewardParticipantIndex(ctx)
	genesis.DenomRewardScheduleList = k.GetAllDenomRewardSchedule(ctx)
	genesis.DenomRewardScheduleCounter = k.GetDenomRewardScheduleCounter(ctx)

	if denomDistQueue, found := k.GetDenomRewardsDistributionQueue(ctx); found {
		genesis.DenomRewardsDistributionQueue = &denomDistQueue
	}

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
