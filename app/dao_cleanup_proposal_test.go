package app_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	daotypes "github.com/bze-alphateam/bze/x/dao/types"
)

// A finished proposal cleaned up through the real router, one ballot per call:
// the first call keeps the proposal, the second deletes it.
func TestDaoCleanupProposal_DeletesFinishedProposal(t *testing.T) {
	bApp, ctx, admin, _ := newAdminDaoApp(t)
	memberA := sdk.AccAddress("dao_member_a________")
	memberB := sdk.AccAddress("dao_member_b________")
	cleaner := sdk.AccAddress("dao_cleaner_________")

	_, err := runAsTx(bApp, ctx, &daotypes.MsgCreateDao{
		Creator: admin.String(),
		Type:    daotypes.DaoType_DAO_TYPE_STATIC_LIST,
		Name:    "cleaners",
		Config:  newAdminDaoConfig(),
		Members: []string{memberA.String(), memberB.String()},
	})
	require.NoError(t, err)

	// DAO 1 is the helper's DAO; this self-owned one is 2
	submitMsg, err := daotypes.NewMsgSubmitProposal(memberA.String(), 2, "signal", "", nil, false)
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, submitMsg)
	require.NoError(t, err)
	proposalId := bApp.DaoKeeper.GetProposalCounter(ctx)
	for _, member := range []sdk.AccAddress{memberA, memberB} {
		_, err = runAsTx(bApp, ctx, daotypes.NewMsgVote(member.String(), proposalId, daotypes.VoteOption_YES))
		require.NoError(t, err)
	}

	_, err = runAsTx(bApp, ctx, daotypes.NewMsgCleanupProposal(cleaner.String(), proposalId, 0))
	require.ErrorIs(t, err, daotypes.ErrProposalNotFinished)

	proposal, _ := bApp.DaoKeeper.GetProposal(ctx, proposalId)
	ctx = ctx.WithBlockTime(time.Unix(proposal.VotingEnd, 0))
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgSettleProposal(cleaner.String(), proposalId))
	require.NoError(t, err)

	_, err = runAsTx(bApp, ctx, daotypes.NewMsgCleanupProposal(cleaner.String(), proposalId, 1))
	require.NoError(t, err)
	_, found := bApp.DaoKeeper.GetProposal(ctx, proposalId)
	require.True(t, found)

	_, err = runAsTx(bApp, ctx, daotypes.NewMsgCleanupProposal(cleaner.String(), proposalId, 1))
	require.NoError(t, err)
	_, found = bApp.DaoKeeper.GetProposal(ctx, proposalId)
	require.False(t, found)
	require.False(t, bApp.DaoKeeper.HasDaoProposalIndex(ctx, 2, proposalId))
	for _, member := range []sdk.AccAddress{memberA, memberB} {
		_, found = bApp.DaoKeeper.GetBallot(ctx, proposalId, member)
		require.False(t, found)
	}
}
