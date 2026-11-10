package app_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	daotypes "github.com/bze-alphateam/bze/x/dao/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
)

// A proposal settled by another account pays its protocol fee in native, with the real keepers: the fee denom of the
// settler's tx says nothing about the token the DAO pays in. This test lives in app because the x/dao keeper suite
// mocks tradebin, which picks the denom.
func TestDaoSettleProposal_ProtocolFeeIgnoresSettlerFeeDenom(t *testing.T) {
	bApp, ctx, token, pool := transferFeeToken(t)
	admin := sdk.AccAddress("dao_admin___________")
	proposer := sdk.AccAddress("dao_proposer________")
	settler := sdk.AccAddress("dao_settler_________")

	_, err := runAsTx(bApp, ctx, &daotypes.MsgCreateDao{
		Creator: admin.String(),
		Type:    daotypes.DaoType_DAO_TYPE_STATIC_LIST,
		Name:    "spenders",
		Config:  newAdminDaoConfig(),
		Members: []string{proposer.String()},
	})
	require.NoError(t, err)
	// DAO 1 is the helper's DAO; this self-owned one is 2
	treasury := daotypes.DaoAddress(2)
	require.NoError(t, bApp.DaoKeeper.SetParams(ctx, daotypes.DefaultParams()))

	// the treasury can pay either way
	fee := daotypes.DefaultDaoCreationFee
	nativeFunds := sdk.NewCoins(fee.AddAmount(fee.Amount))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, nativeFunds))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, treasury, nativeFunds))
	fundTokenPayer(t, bApp, ctx, pool, fee, treasury)
	tokensBefore := bApp.BankKeeper.GetBalance(ctx, treasury, token)
	require.True(t, tokensBefore.IsPositive())

	submitMsg, err := daotypes.NewMsgSubmitProposal(proposer.String(), 2, "create", "", []sdk.Msg{createDaoMsg(treasury)}, false)
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, submitMsg)
	require.NoError(t, err)
	proposalId := bApp.DaoKeeper.GetProposalCounter(ctx)
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgVote(proposer.String(), proposalId, daotypes.VoteOption_YES))
	require.NoError(t, err)
	proposal, _ := bApp.DaoKeeper.GetProposal(ctx, proposalId)
	ctx = ctx.WithBlockTime(time.Unix(proposal.VotingEnd, 0))

	settleCtx := ctx.WithValue(tradebinkeeper.CtxFeeDenomKey, token).WithValue(tradebinkeeper.CtxFeePayerKey, settler)
	_, err = runAsTx(bApp, settleCtx, daotypes.NewMsgSettleProposal(settler.String(), proposalId))
	require.NoError(t, err)

	proposal, _ = bApp.DaoKeeper.GetProposal(ctx, proposalId)
	require.Equal(t, daotypes.ProposalStatus_EXECUTED, proposal.Status)
	require.Equal(t, tokensBefore, bApp.BankKeeper.GetBalance(ctx, treasury, token), "the treasury's token is untouched")
	require.Equal(t, fee, bApp.BankKeeper.GetBalance(ctx, treasury, fee.Denom), "the treasury paid the native fee")
}
