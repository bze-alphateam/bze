package rewards_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func seedCandidates(ctx sdk.Context, k *keeper.Keeper, rewardId string, count int) {
	for i := 0; i < count; i++ {
		k.SetTradingRewardCandidate(ctx, types.TradingRewardCandidate{
			RewardId: rewardId,
			Amount:   "1",
			Address:  fmt.Sprintf("bze1trader%07d", i),
		})
	}
}

// The expiry hook only queues the candidates; AppModule.EndBlock is what removes them.
func TestEndBlock_RemovesExpiredTradingRewardCandidatesInBatches(t *testing.T) {
	f := newWiringFixture(t)
	require.NoError(t, f.k.SetParams(f.ctx, types.DefaultParams()))
	_, _, _, am := f.manager(t)

	const rewardId = "000000000001"
	limit := types.MaxTradingRewardCandidateRemovalsPerBlock
	f.k.SetActiveTradingReward(f.ctx, types.TradingReward{RewardId: rewardId, ExpireAt: 100})
	f.k.SetActiveTradingRewardExpiration(f.ctx, types.TradingRewardExpiration{RewardId: rewardId, ExpireAt: 268})
	seedCandidates(f.ctx, f.k, rewardId, limit+5)

	require.NoError(t, f.k.GetTradingRewardsDistributionHook().AfterEpochEnd(f.ctx, "hour", 268))
	require.Len(t, f.k.GetTradingRewardCandidateByRewardId(f.ctx, rewardId), limit+5)

	require.NoError(t, am.EndBlock(f.ctx))
	require.Len(t, f.k.GetTradingRewardCandidateByRewardId(f.ctx, rewardId), 5)

	require.NoError(t, am.EndBlock(f.ctx))
	require.Empty(t, f.k.GetTradingRewardCandidateByRewardId(f.ctx, rewardId))
	require.Empty(t, f.k.GetAllTradingRewardCandidatePurge(f.ctx))
}

// Rows waiting for removal belong to rewards that no longer exist, so an export leaves them out
// and the imported chain has nothing to finish.
func TestGenesis_ExportDuringCandidatePurgeOmitsLeftoverRows(t *testing.T) {
	newKeeper := func() (*keeper.Keeper, sdk.Context) {
		ctrl := gomock.NewController(t)
		k, ctx := keepertest.RewardsKeeper(t, nil, nil, nil, testutil.NewMockAccountKeeper(ctrl))
		return &k, ctx
	}

	const expiredId, activeId = "000000000001", "000000000002"
	source, sourceCtx := newKeeper()
	seedCandidates(sourceCtx, source, expiredId, 4)
	seedCandidates(sourceCtx, source, activeId, 3)
	source.EnqueueTradingRewardCandidatePurge(sourceCtx, expiredId)
	source.SetTradingRewardsCounter(sourceCtx, 2)

	exported := rewards.ExportGenesis(sourceCtx, *source)
	require.NoError(t, exported.Validate())
	require.Len(t, exported.TradingRewardCandidateList, 3)
	for _, candidate := range exported.TradingRewardCandidateList {
		require.Equal(t, activeId, candidate.RewardId)
	}

	target, targetCtx := newKeeper()
	rewards.InitGenesis(targetCtx, *target, *exported)

	require.Empty(t, target.GetTradingRewardCandidateByRewardId(targetCtx, expiredId), "nothing left to purge")
	require.Len(t, target.GetTradingRewardCandidateByRewardId(targetCtx, activeId), 3)
	require.Empty(t, target.GetAllTradingRewardCandidatePurge(targetCtx))
	require.Equal(t, uint64(2), target.GetTradingRewardsCounter(targetCtx))

	reexported := rewards.ExportGenesis(targetCtx, *target)
	require.Equal(t, exported.TradingRewardCandidateList, reexported.TradingRewardCandidateList)
}
