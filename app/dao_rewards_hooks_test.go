package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

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
