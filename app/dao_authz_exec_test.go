package app_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
)

var daoExecProposer = sdk.AccAddress("dao_exec_proposer___")

// daoExecChain creates the self-owned DAO 2 with daoExecProposer as its only
// member, then DAOs 3..last, each administered by the DAO before it. The last
// DAO gets 10ubze.
func daoExecChain(t *testing.T, bApp *app.App, ctx sdk.Context, last uint64) {
	t.Helper()
	for id := uint64(2); id <= last; id++ {
		msg := &daotypes.MsgCreateDao{
			Creator: daoExecProposer.String(),
			Type:    daotypes.DaoType_DAO_TYPE_STATIC_LIST,
			Name:    "chain",
			Config:  newAdminDaoConfig(),
			Members: []string{daoExecProposer.String()},
		}
		if id > 2 {
			msg.Admin = daotypes.DaoAddress(id - 1).String()
		}
		_, err := runAsTx(bApp, ctx, msg)
		require.NoError(t, err)
		require.Equal(t, id, bApp.DaoKeeper.GetDaoCounter(ctx))
	}

	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 10))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, daotypes.DaoAddress(last), coins))
}

// daoExecPassed submits msgs to DAO 2 and votes yes; it returns the proposal id.
func daoExecPassed(t *testing.T, bApp *app.App, ctx sdk.Context, msgs ...sdk.Msg) uint64 {
	t.Helper()
	submit, err := daotypes.NewMsgSubmitProposal(daoExecProposer.String(), 2, "run", "", msgs, false)
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, submit)
	require.NoError(t, err)
	id := bApp.DaoKeeper.GetProposalCounter(ctx)
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgVote(daoExecProposer.String(), id, daotypes.VoteOption_YES))
	require.NoError(t, err)

	return id
}

// daoExecSettle settles the proposal at its voting end and returns the stored
// status and the reason of its EventProposalExecuted.
func daoExecSettle(t *testing.T, bApp *app.App, ctx sdk.Context, id uint64) (daotypes.ProposalStatus, string) {
	t.Helper()
	proposal, found := bApp.DaoKeeper.GetProposal(ctx, id)
	require.True(t, found)
	ctx = ctx.WithBlockTime(time.Unix(proposal.VotingEnd, 0))
	// the router returns the handler's events in the result, as baseapp gets them
	settle := daotypes.NewMsgSettleProposal(sdk.AccAddress("dao_exec_settler____").String(), id)
	cacheCtx, write := ctx.CacheContext()
	res, err := bApp.MsgServiceRouter().Handler(settle)(cacheCtx, settle)
	require.NoError(t, err)
	write()

	reason := "<no execution event>"
	for _, e := range res.Events {
		msg, err := sdk.ParseTypedEvent(e)
		if err != nil {
			continue
		}
		if executed, ok := msg.(*daotypes.EventProposalExecuted); ok {
			reason = executed.Error
		}
	}
	proposal, _ = bApp.DaoKeeper.GetProposal(ctx, id)

	return proposal.Status, reason
}

// daoExecNested returns DAO 2's authz exec (granted to itself) of the
// MsgAdminExecute chain from DAO 2 down to DAO last, which sends 1ubze to to.
func daoExecNested(t *testing.T, last uint64, to sdk.AccAddress) sdk.Msg {
	t.Helper()
	var msg sdk.Msg = banktypes.NewMsgSend(daotypes.DaoAddress(last), to, sdk.NewCoins(sdk.NewInt64Coin("ubze", 1)))
	for id := last; id > 2; id-- {
		inner, err := daotypes.NewMsgAdminExecute(daotypes.DaoAddress(id-1).String(), id, []sdk.Msg{msg})
		require.NoError(t, err)
		msg = inner
	}
	exec := authz.NewMsgExec(daotypes.DaoAddress(2), []sdk.Msg{msg})

	return &exec
}

// The execution depth is carried through the real authz keeper: a proposal
// whose sub-DAO executions are wrapped in an authz exec runs up to
// MaxDaoExecutionDepth levels and fails one more.
func TestDaoAuthzExec_DepthCapHolds(t *testing.T) {
	require.Equal(t, 5, daotypes.MaxDaoExecutionDepth)
	recipient := sdk.AccAddress("dao_exec_recipient__")

	for _, tc := range []struct {
		depth      uint64
		wantStatus daotypes.ProposalStatus
		wantReason string
	}{
		{depth: daotypes.MaxDaoExecutionDepth, wantStatus: daotypes.ProposalStatus_EXECUTED},
		{depth: daotypes.MaxDaoExecutionDepth + 1, wantStatus: daotypes.ProposalStatus_EXECUTION_FAILED, wantReason: "execution nested too deep"},
	} {
		bApp, ctx, _, _ := newAdminDaoApp(t)
		// DAO 2's proposal is the first level, each DAO below it one more
		last := tc.depth + 1
		daoExecChain(t, bApp, ctx, last)
		id := daoExecPassed(t, bApp, ctx, daoExecNested(t, last, recipient))

		status, reason := daoExecSettle(t, bApp, ctx, id)

		require.Equal(t, tc.wantStatus, status, "depth %d", tc.depth)
		require.Equal(t, tc.wantReason, reason, "depth %d", tc.depth)
		want := int64(1)
		if tc.wantReason != "" {
			want = 0
		}
		require.Equal(t, want, bApp.BankKeeper.GetBalance(ctx, recipient, "ubze").Amount.Int64(), "depth %d", tc.depth)
	}
}

// A settle wrapped in a real authz exec inside a proposal is refused: the
// proposal fails with a fixed reason and the target stays in voting.
func TestDaoAuthzExec_NestedSettleRefused(t *testing.T) {
	bApp, ctx, _, _ := newAdminDaoApp(t)
	daoExecChain(t, bApp, ctx, 2)
	dao := daotypes.DaoAddress(2)
	target := daoExecPassed(t, bApp, ctx)
	exec := authz.NewMsgExec(dao, []sdk.Msg{daotypes.NewMsgSettleProposal(dao.String(), target)})
	outer := daoExecPassed(t, bApp, ctx, &exec)

	status, reason := daoExecSettle(t, bApp, ctx, outer)

	require.Equal(t, daotypes.ProposalStatus_EXECUTION_FAILED, status)
	require.Equal(t, "message not allowed", reason)
	proposal, found := bApp.DaoKeeper.GetProposal(ctx, target)
	require.True(t, found)
	require.Equal(t, daotypes.ProposalStatus_VOTING, proposal.Status)
}
