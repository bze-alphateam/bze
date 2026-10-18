package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// The app registers x/dao on the x/rewards Denom Reward hooks: joining a
// Denom Reward a DAO votes with writes the join record; exiting deletes it
// and takes the stake out of the staker's open ballot.
func TestDaoDenomRewardHooks_Wired(t *testing.T) {
	bApp, ctx, admin, _ := newAdminDaoApp(t)
	staker := sdk.AccAddress("dao_staker__________")

	bApp.RewardsKeeper.SetDenomReward(ctx, rewardstypes.DenomReward{StakingDenom: "ubze", Lock: 5, StakedAmount: math.ZeroInt()})
	_, err := runAsTx(bApp, ctx, &daotypes.MsgCreateDao{
		Creator:    admin.String(),
		Admin:      admin.String(),
		Type:       daotypes.DaoType_DAO_TYPE_STAKE,
		Name:       "stakers",
		Config:     newAdminDaoConfig(),
		Electorate: &daotypes.StakeElectorate{DrDenom: "ubze"},
	})
	require.NoError(t, err)
	// the lock is checked only when the source joins the DAO; dropping it here
	// lets the exit pay out at once, with no epoch module state needed
	bApp.RewardsKeeper.SetDenomReward(ctx, rewardstypes.DenomReward{StakingDenom: "ubze", Lock: 0, StakedAmount: math.ZeroInt()})

	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, staker, coins))

	ctx = ctx.WithBlockHeight(3)
	_, err = runAsTx(bApp, ctx, rewardstypes.NewMsgJoinDenomReward(staker.String(), "ubze", math.NewInt(400)))
	require.NoError(t, err)

	record, found := bApp.DaoKeeper.GetJoinRecord(ctx, daotypes.DenomRewardSource("ubze"), staker)
	require.True(t, found)
	require.Equal(t, int64(3), record.Height)

	// DAO 1 is the helper's static-list DAO; the stake DAO is 2
	ctx = ctx.WithBlockHeight(4)
	submitMsg, err := daotypes.NewMsgSubmitProposal(staker.String(), 2, "signal", "", nil, false)
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, submitMsg)
	require.NoError(t, err)
	proposalId := bApp.DaoKeeper.GetProposalCounter(ctx)
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgVote(staker.String(), proposalId, daotypes.VoteOption_YES))
	require.NoError(t, err)
	proposal, _ := bApp.DaoKeeper.GetProposal(ctx, proposalId)
	require.Equal(t, math.NewInt(400), proposal.Yes)

	_, err = runAsTx(bApp, ctx, &rewardstypes.MsgExitDenomReward{Creator: staker.String(), Denom: "ubze"})
	require.NoError(t, err)
	_, found = bApp.DaoKeeper.GetJoinRecord(ctx, daotypes.DenomRewardSource("ubze"), staker)
	require.False(t, found)

	proposal, _ = bApp.DaoKeeper.GetProposal(ctx, proposalId)
	require.True(t, proposal.Yes.IsZero())
	ballot, _ := bApp.DaoKeeper.GetBallot(ctx, proposalId, staker)
	require.True(t, ballot.Weight.IsZero())
	require.False(t, bApp.DaoKeeper.HasVoteIndex(ctx, daotypes.DenomRewardSource("ubze"), staker, proposalId))
}

// newStakeDaoApp builds the real app with staking reward 1 and the ubze
// Denom Reward, and creates stake DAO 2 over both. The locks are dropped after
// the creation checked them, so exits pay out at once. staker holds 2,000ubze.
func newStakeDaoApp(t *testing.T, staker sdk.AccAddress) (*app.App, sdk.Context) {
	t.Helper()

	bApp, ctx, admin, _ := newAdminDaoApp(t)
	reward := rewardstypes.StakingReward{RewardId: "1", PrizeAmount: "1", PrizeDenom: "ubze", StakingDenom: "ubze", Duration: 10, Payouts: 0, Lock: 5, StakedAmount: "0", DistributedStake: "0"}
	bApp.RewardsKeeper.SetStakingReward(ctx, reward)
	bApp.RewardsKeeper.SetDenomReward(ctx, rewardstypes.DenomReward{StakingDenom: "ubze", Lock: 5, StakedAmount: math.ZeroInt()})
	_, err := runAsTx(bApp, ctx, &daotypes.MsgCreateDao{
		Creator:    admin.String(),
		Admin:      admin.String(),
		Type:       daotypes.DaoType_DAO_TYPE_STAKE,
		Name:       "stakers",
		Config:     newAdminDaoConfig(),
		Electorate: &daotypes.StakeElectorate{SrIds: []string{"1"}, DrDenom: "ubze"},
	})
	require.NoError(t, err)
	reward.Lock = 0
	bApp.RewardsKeeper.SetStakingReward(ctx, reward)
	bApp.RewardsKeeper.SetDenomReward(ctx, rewardstypes.DenomReward{StakingDenom: "ubze", Lock: 0, StakedAmount: math.ZeroInt()})

	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 2_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, staker, coins))

	return bApp, ctx
}

// submitAndVote submits a signal proposal to stake DAO 2 as proposer and
// votes yes; it returns the proposal id.
func submitAndVote(t *testing.T, bApp *app.App, ctx sdk.Context, proposer sdk.AccAddress) uint64 {
	t.Helper()

	submitMsg, err := daotypes.NewMsgSubmitProposal(proposer.String(), 2, "signal", "", nil, false)
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, submitMsg)
	require.NoError(t, err)
	proposalId := bApp.DaoKeeper.GetProposalCounter(ctx)
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgVote(proposer.String(), proposalId, daotypes.VoteOption_YES))
	require.NoError(t, err)

	return proposalId
}

// The staking reward hooks through the real x/rewards messages: a join writes
// the join record, a second join (a top-up) moves it to its block so the
// added stake cannot weigh on the open proposal, a later proposal counts the
// whole stake, and the exit takes it out of both ballots.
func TestDaoStakingRewardHooks_JoinTopUpExit(t *testing.T) {
	staker := sdk.AccAddress("dao_staker__________")
	bApp, ctx := newStakeDaoApp(t, staker)
	source := daotypes.StakingRewardSource("1")

	ctx = ctx.WithBlockHeight(3)
	_, err := runAsTx(bApp, ctx, rewardstypes.NewMsgJoinStaking(staker.String(), "1", "400"))
	require.NoError(t, err)
	record, found := bApp.DaoKeeper.GetJoinRecord(ctx, source, staker)
	require.True(t, found)
	require.Equal(t, int64(3), record.Height)

	ctx = ctx.WithBlockHeight(4)
	open := submitAndVote(t, bApp, ctx, staker)
	proposal, _ := bApp.DaoKeeper.GetProposal(ctx, open)
	require.Equal(t, math.NewInt(400), proposal.Yes)

	ctx = ctx.WithBlockHeight(5)
	_, err = runAsTx(bApp, ctx, rewardstypes.NewMsgJoinStaking(staker.String(), "1", "600"))
	require.NoError(t, err)
	record, _ = bApp.DaoKeeper.GetJoinRecord(ctx, source, staker)
	require.Equal(t, int64(5), record.Height)
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgVote(staker.String(), open, daotypes.VoteOption_NO))
	require.ErrorIs(t, err, daotypes.ErrNoVotingPower)
	proposal, _ = bApp.DaoKeeper.GetProposal(ctx, open)
	require.Equal(t, math.NewInt(400), proposal.Yes)

	ctx = ctx.WithBlockHeight(6)
	later := submitAndVote(t, bApp, ctx, staker)
	proposal, _ = bApp.DaoKeeper.GetProposal(ctx, later)
	require.Equal(t, math.NewInt(1_000), proposal.Yes)

	_, err = runAsTx(bApp, ctx, rewardstypes.NewMsgExitStaking(staker.String(), "1"))
	require.NoError(t, err)
	_, found = bApp.DaoKeeper.GetJoinRecord(ctx, source, staker)
	require.False(t, found)
	for _, id := range []uint64{open, later} {
		proposal, _ = bApp.DaoKeeper.GetProposal(ctx, id)
		require.True(t, proposal.Yes.IsZero(), "proposal %d", id)
		require.False(t, bApp.DaoKeeper.HasVoteIndex(ctx, source, staker, id))
	}
}

// The same top-up rule for a Denom Reward, through MsgJoinDenomReward.
func TestDaoDenomRewardHooks_TopUp(t *testing.T) {
	staker := sdk.AccAddress("dao_staker__________")
	bApp, ctx := newStakeDaoApp(t, staker)
	source := daotypes.DenomRewardSource("ubze")

	ctx = ctx.WithBlockHeight(3)
	_, err := runAsTx(bApp, ctx, rewardstypes.NewMsgJoinDenomReward(staker.String(), "ubze", math.NewInt(400)))
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(4)
	open := submitAndVote(t, bApp, ctx, staker)

	ctx = ctx.WithBlockHeight(5)
	_, err = runAsTx(bApp, ctx, rewardstypes.NewMsgJoinDenomReward(staker.String(), "ubze", math.NewInt(600)))
	require.NoError(t, err)
	record, found := bApp.DaoKeeper.GetJoinRecord(ctx, source, staker)
	require.True(t, found)
	require.Equal(t, int64(5), record.Height)
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgVote(staker.String(), open, daotypes.VoteOption_NO))
	require.ErrorIs(t, err, daotypes.ErrNoVotingPower)
	proposal, _ := bApp.DaoKeeper.GetProposal(ctx, open)
	require.Equal(t, math.NewInt(400), proposal.Yes)
}
