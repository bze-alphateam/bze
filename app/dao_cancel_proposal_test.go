package app_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	daotypes "github.com/bze-alphateam/bze/x/dao/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// Cancelling a proposal moves its escrowed deposit out of the dao module
// account with the real bank keeper: back to the proposer under ALWAYS, to
// the burner when the deposit is kept.
func TestDaoCancelProposal_ReleasesDeposit(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		policy   daotypes.DepositRefundPolicy
		proposer int64
		burner   int64
	}{
		{desc: "refunded", policy: daotypes.DepositRefundPolicy_ALWAYS, proposer: 100, burner: 0},
		{desc: "kept and burned", policy: daotypes.DepositRefundPolicy_NEVER, proposer: 60, burner: 40},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			bApp, ctx, admin, _ := newAdminDaoApp(t)
			proposer := sdk.AccAddress("dao_proposer________")

			config := newAdminDaoConfig()
			deposit := sdk.NewInt64Coin("ubze", 40)
			config.ProposalDeposit = &deposit
			config.DepositRefundPolicy = tc.policy
			config.KeptDepositDestination = daotypes.FundsDestination_BURN
			_, err := runAsTx(bApp, ctx, &daotypes.MsgCreateDao{
				Creator: admin.String(),
				Admin:   admin.String(),
				Type:    daotypes.DaoType_DAO_TYPE_STATIC_LIST,
				Name:    "depositors",
				Config:  config,
				Members: []string{proposer.String()},
			})
			require.NoError(t, err)

			coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 100))
			require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
			require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, proposer, coins))

			// DAO 1 is the helper's DAO; this one is 2
			submitMsg, err := daotypes.NewMsgSubmitProposal(proposer.String(), 2, "signal", "", nil, false)
			require.NoError(t, err)
			_, err = runAsTx(bApp, ctx, submitMsg)
			require.NoError(t, err)
			daoModule := authtypes.NewModuleAddress(daotypes.ModuleName)
			require.Equal(t, int64(40), bApp.BankKeeper.GetBalance(ctx, daoModule, "ubze").Amount.Int64())

			proposalId := bApp.DaoKeeper.GetProposalCounter(ctx)
			_, err = runAsTx(bApp, ctx, daotypes.NewMsgCancelProposal(proposer.String(), proposalId))
			require.NoError(t, err)

			proposal, _ := bApp.DaoKeeper.GetProposal(ctx, proposalId)
			require.Equal(t, daotypes.ProposalStatus_CANCELLED, proposal.Status)
			burner := authtypes.NewModuleAddress(txfeecollectortypes.BurnerFeeCollector)
			require.True(t, bApp.BankKeeper.GetBalance(ctx, daoModule, "ubze").IsZero())
			require.Equal(t, tc.proposer, bApp.BankKeeper.GetBalance(ctx, proposer, "ubze").Amount.Int64())
			require.Equal(t, tc.burner, bApp.BankKeeper.GetBalance(ctx, burner, "ubze").Amount.Int64())
		})
	}
}
