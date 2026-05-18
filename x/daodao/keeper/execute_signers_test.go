package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// mustPackAny is a helper for the signer-validation tests. Builds a
// codec.Any wrapping an sdk.Msg with the cached value populated, which
// is the post-unpack state that ValidateBasic + the keeper expect.
func (suite *IntegrationTestSuite) mustPackAny(m sdk.Msg) *cdctypes.Any {
	any, err := cdctypes.NewAnyWithValue(m)
	suite.Require().NoError(err)
	return any
}

// TestCreateProposal_RejectsNonDAOSigner: a msg in the bundle whose signer
// is NOT the DAO's account is rejected at MsgCreateProposal time.
func (suite *IntegrationTestSuite) TestCreateProposal_RejectsNonDAOSigner() {
	daoID, admin := suite.createSampleDao("non-dao-signer")
	other := freshAddr()

	// MsgUpdateMembers with Authority = `other` (not the DAO) — the bundle
	// signer check should reject.
	bad := &types.MsgUpdateMembers{
		Authority: other,
		DaoId:     daoID,
		Add:       []types.StaticMember{{Address: other, Weight: 1}},
	}
	// Note: no expectInitialDepositSend — the signer-check rejection
	// fires BEFORE the keeper collects the initial deposit.
	_, err := suite.msgServer.CreateProposal(suite.ctx, &types.MsgCreateProposal{
		Proposer:       admin,
		DaoId:          daoID,
		Title:          "bad-signer",
		InitialDeposit: sdk.NewInt64Coin("ubze", 1),
		Msgs:           []*cdctypes.Any{suite.mustPackAny(bad)},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid proposal message signers")
}

// TestCreateProposal_RejectsExecuteProposalInBundle: MsgExecuteProposal
// inside a proposal's msgs[] would re-enter the dispatcher and recurse
// (status flips to EXECUTED only AFTER dispatch returns). The
// bundle-msg denylist rejects it at proposal creation.
func (suite *IntegrationTestSuite) TestCreateProposal_RejectsExecuteProposalInBundle() {
	daoID, admin := suite.createSampleDao("exec-in-bundle")
	daoAddr := types.DaoAccountAddress(daoID).String()

	// Even though the inner MsgExecuteProposal is signed by the DAO, it
	// is structurally disallowed inside a bundle.
	innerExec := &types.MsgExecuteProposal{
		Executor:   daoAddr,
		DaoId:      daoID,
		ProposalId: 1,
	}
	_, err := suite.msgServer.CreateProposal(suite.ctx, &types.MsgCreateProposal{
		Proposer:       admin,
		DaoId:          daoID,
		Title:          "self-exec",
		InitialDeposit: sdk.NewInt64Coin("ubze", 1),
		Msgs:           []*cdctypes.Any{suite.mustPackAny(innerExec)},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "not allowed in proposal bundles")
}

// TestCreateProposal_AcceptsDAOSigner: when the bundle's msg.Authority
// equals the DAO's account address, the proposal is accepted. This is
// the canonical self-modify pattern (DAO governance proposes a config
// change, dispatched against itself at execution).
func (suite *IntegrationTestSuite) TestCreateProposal_AcceptsDAOSigner() {
	daoID, admin := suite.createSampleDao("dao-signer-ok")
	daoAddr := types.DaoAccountAddress(daoID).String()

	bundle := &types.MsgUpdateDaoMetadata{
		Authority: daoAddr,
		DaoId:     daoID,
		Metadata:  sampleMetadata("renamed-by-self"),
	}

	suite.expectInitialDepositSend(daoID, admin, sdk.NewInt64Coin("ubze", 1))
	resp, err := suite.msgServer.CreateProposal(suite.ctx, &types.MsgCreateProposal{
		Proposer:       admin,
		DaoId:          daoID,
		Title:          "self-modify",
		InitialDeposit: sdk.NewInt64Coin("ubze", 1),
		Msgs:           []*cdctypes.Any{suite.mustPackAny(bundle)},
	})
	suite.Require().NoError(err)

	// The proposal should be in VOTING (initial_deposit met min).
	p, _ := suite.k.GetProposal(suite.ctx, daoID, resp.ProposalId)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_VOTING, p.Status)
	suite.Require().Len(p.Msgs, 1)
}
