package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	daokeeper "github.com/bze-alphateam/bze/x/dao/keeper"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
	rewardskeeper "github.com/bze-alphateam/bze/x/rewards/keeper"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// x/dao is registered as the rewards hook subscriber: a finished, emptied
// staking reward used by a DAO electorate cannot be deleted, an unused one can.
func TestRewardsHooks_DaoVetoesPinnedStakingRewardRemoval(t *testing.T) {
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{Time: time.Unix(1_700_000_000, 0)})

	params := daotypes.DefaultParams()
	params.DaoCreationFee = sdk.NewInt64Coin("ubze", 0)
	require.NoError(t, bApp.DaoKeeper.SetParams(ctx, params))

	for _, id := range []string{"1", "2"} {
		bApp.RewardsKeeper.SetStakingReward(ctx, rewardstypes.StakingReward{
			RewardId:     id,
			StakingDenom: "ubze",
			Duration:     1,
			Payouts:      1,
			Lock:         5,
			StakedAmount: math.ZeroInt().String(),
		})
	}

	creator := sdk.AccAddress("dao_creator_________").String()
	_, err = daokeeper.NewMsgServerImpl(bApp.DaoKeeper).CreateDao(ctx, &daotypes.MsgCreateDao{
		Creator: creator,
		Admin:   creator,
		Type:    daotypes.DaoType_DAO_TYPE_STAKE,
		Name:    "stakers",
		Config: daotypes.DaoConfig{
			MinQuorum:         math.LegacyMustNewDecFromStr("0.4"),
			Threshold:         math.LegacyMustNewDecFromStr("0.5"),
			VotingPeriod:      72 * time.Hour,
			ExpeditedPeriod:   24 * time.Hour,
			ExpediteQuorum:    math.LegacyMustNewDecFromStr("0.4"),
			ExpediteThreshold: math.LegacyMustNewDecFromStr("0.5"),
		},
		Electorate: &daotypes.StakeElectorate{SrIds: []string{"1"}},
	})
	require.NoError(t, err)

	rewardsServer := rewardskeeper.NewMsgServerImpl(bApp.RewardsKeeper)

	_, err = rewardsServer.DeleteStakingReward(ctx, &rewardstypes.MsgDeleteStakingReward{Creator: creator, RewardId: "1"})
	require.ErrorIs(t, err, daotypes.ErrSourcePinned)
	_, found := bApp.RewardsKeeper.GetStakingReward(ctx, "1")
	require.True(t, found)

	_, err = rewardsServer.DeleteStakingReward(ctx, &rewardstypes.MsgDeleteStakingReward{Creator: creator, RewardId: "2"})
	require.NoError(t, err)
	_, found = bApp.RewardsKeeper.GetStakingReward(ctx, "2")
	require.False(t, found)
}
