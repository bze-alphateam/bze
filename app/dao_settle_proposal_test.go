package app_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	daotypes "github.com/bze-alphateam/bze/x/dao/types"
)

// A passed treasury spend settled through the real wiring: anyone settles, the
// deposit goes back to the proposer and the DAO's funds move with the real
// bank keeper.
func TestDaoSettleProposal_SpendsTreasury(t *testing.T) {
	bApp, ctx, admin, _ := newAdminDaoApp(t)
	proposer := sdk.AccAddress("dao_proposer________")
	recipient := sdk.AccAddress("dao_recipient_______")
	settler := sdk.AccAddress("dao_settler_________")

	config := newAdminDaoConfig()
	deposit := sdk.NewInt64Coin("ubze", 40)
	config.ProposalDeposit = &deposit
	config.DepositRefundPolicy = daotypes.DepositRefundPolicy_ONLY_PASSED
	_, err := runAsTx(bApp, ctx, &daotypes.MsgCreateDao{
		Creator: admin.String(),
		Type:    daotypes.DaoType_DAO_TYPE_STATIC_LIST,
		Name:    "spenders",
		Config:  config,
		Members: []string{proposer.String()},
	})
	require.NoError(t, err)
	// DAO 1 is the helper's DAO; this self-owned one is 2
	treasury := daotypes.DaoAddress(2)

	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 100))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins.Add(coins...)))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, proposer, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, treasury, coins))

	submitMsg, err := daotypes.NewMsgSubmitProposal(proposer.String(), 2, "pay", "", []sdk.Msg{
		banktypes.NewMsgSend(treasury, recipient, sdk.NewCoins(sdk.NewInt64Coin("ubze", 70))),
	}, false)
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, submitMsg)
	require.NoError(t, err)
	proposalId := bApp.DaoKeeper.GetProposalCounter(ctx)
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgVote(proposer.String(), proposalId, daotypes.VoteOption_YES))
	require.NoError(t, err)

	proposal, _ := bApp.DaoKeeper.GetProposal(ctx, proposalId)
	ctx = ctx.WithBlockTime(time.Unix(proposal.VotingEnd, 0))
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgSettleProposal(settler.String(), proposalId))
	require.NoError(t, err)

	proposal, _ = bApp.DaoKeeper.GetProposal(ctx, proposalId)
	require.Equal(t, daotypes.ProposalStatus_EXECUTED, proposal.Status)
	require.Equal(t, int64(30), bApp.BankKeeper.GetBalance(ctx, treasury, "ubze").Amount.Int64())
	require.Equal(t, int64(70), bApp.BankKeeper.GetBalance(ctx, recipient, "ubze").Amount.Int64())
	require.Equal(t, int64(100), bApp.BankKeeper.GetBalance(ctx, proposer, "ubze").Amount.Int64())
	require.True(t, bApp.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(daotypes.ModuleName), "ubze").IsZero())
}
