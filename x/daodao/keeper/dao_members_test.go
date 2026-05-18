package keeper_test

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// bulkStaticMembers returns n deterministic StaticMember entries with weight 1.
// We sidestep sample.AccAddress (ed25519 keygen) because at MaxStaticMembers
// scale the keygen cost dominates the test. The addresses don't need to be
// cryptographically derived for the cap test — only unique and bech32-valid.
func bulkStaticMembers(n int) []types.StaticMember {
	members := make([]types.StaticMember, n)
	for i := 0; i < n; i++ {
		b := make([]byte, 20)
		// i+1 so we never produce the zero address (which sdk treats as empty).
		binary.BigEndian.PutUint64(b[12:], uint64(i+1))
		members[i] = types.StaticMember{
			Address: sdk.AccAddress(b).String(),
			Weight:  1,
		}
	}
	return members
}

// TestUpdateMembers_AddAndRemove: standard mixed add/remove flow.
func (suite *IntegrationTestSuite) TestUpdateMembers_AddAndRemove() {
	creator := freshAddr()
	other := freshAddr()
	third := freshAddr()

	suite.expectAccountCreated(1)
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:  creator,
		Metadata: sampleMetadata("members"),
		VotingConfig: staticConfigWithMembers([]types.StaticMember{
			{Address: creator, Weight: 1},
			{Address: other, Weight: 2},
		}),
		Governance: validGovernance(),
		Deposit:    validDeposit(),
	})
	suite.Require().NoError(err)

	// Remove `other`, add `third` with weight 4.
	_, err = suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: creator,
		DaoId:     resp.DaoId,
		Add:       []types.StaticMember{{Address: third, Weight: 4}},
		Remove:    []string{other},
	})
	suite.Require().NoError(err)

	total, err := suite.k.TotalVotingPower(suite.ctx, &types.QueryTotalVotingPowerRequest{DaoId: resp.DaoId})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(5), total.Total) // 1 (creator) + 4 (third)

	// Removed member has 0 power.
	gone, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{DaoId: resp.DaoId, Address: other})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(0), gone.Power)

	// Added member has weight 4.
	got, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{DaoId: resp.DaoId, Address: third})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(4), got.Power)
}

// TestUpdateMembers_Upsert: re-using an existing member's address in `add`
// overwrites the weight rather than failing.
func (suite *IntegrationTestSuite) TestUpdateMembers_Upsert() {
	creator := freshAddr()
	suite.expectAccountCreated(1)
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("upsert"),
		VotingConfig: staticConfigWithMembers([]types.StaticMember{{Address: creator, Weight: 1}}),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	_, err = suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: creator,
		DaoId:     resp.DaoId,
		Add:       []types.StaticMember{{Address: creator, Weight: 7}},
	})
	suite.Require().NoError(err)

	got, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{DaoId: resp.DaoId, Address: creator})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(7), got.Power)
	suite.Require().Equal(uint64(7), got.Total)
}

// TestUpdateMembers_RejectsEmptyMembership: removing the last member leaves
// the DAO empty, which is invalid.
func (suite *IntegrationTestSuite) TestUpdateMembers_RejectsEmptyMembership() {
	creator := freshAddr()
	suite.expectAccountCreated(1)
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("solo"),
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	_, err = suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: creator,
		DaoId:     resp.DaoId,
		Remove:    []string{creator},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "at least one member")
}

// TestUpdateMembers_RemoveNonexistent: removing an address that isn't a
// member is a stateful error.
func (suite *IntegrationTestSuite) TestUpdateMembers_RemoveNonexistent() {
	daoID, admin := suite.createSampleDao("alpha")

	_, err := suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: admin,
		DaoId:     daoID,
		Remove:    []string{freshAddr()},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "not a DAO member")
}

// TestUpdateMembers_AdminGated: non-admin signers are rejected.
func (suite *IntegrationTestSuite) TestUpdateMembers_AdminGated() {
	daoID, _ := suite.createSampleDao("alpha")
	intruder := freshAddr()

	_, err := suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: intruder,
		DaoId:     daoID,
		Add:       []types.StaticMember{{Address: intruder, Weight: 1}},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "unauthorized")
}

// TestUpdateMembers_RejectsEmptyMsg: no add and no remove is rejected at
// ValidateBasic.
func (suite *IntegrationTestSuite) TestUpdateMembers_RejectsEmptyMsg() {
	daoID, admin := suite.createSampleDao("alpha")
	_, err := suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: admin,
		DaoId:     daoID,
	})
	suite.Require().Error(err)
}

// TestUpdateMembers_DisjointAddRemove: an address in both add and remove
// is rejected by ValidateBasic.
func (suite *IntegrationTestSuite) TestUpdateMembers_DisjointAddRemove() {
	daoID, admin := suite.createSampleDao("alpha")
	target := freshAddr()

	_, err := suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: admin,
		DaoId:     daoID,
		Add:       []types.StaticMember{{Address: target, Weight: 1}},
		Remove:    []string{target},
	})
	suite.Require().Error(err)
}

// TestUpdateMembers_RejectsExceedingMaxStaticMembers: ValidateBasic only sizes
// the `add` slice against MaxStaticMembers — it does not consider the existing
// member set. Without the post-update cap check in msgServer.UpdateMembers, an
// admin could grow a near-cap DAO past MaxStaticMembers over successive calls,
// breaking the bound staticVotingBackend.SnapshotAll iteration relies on.
//
// This test fills a DAO to exactly MaxStaticMembers, then attempts to add one
// more member. The expected outcome is an explicit cap rejection.
//
// We do NOT exercise a follow-on "upsert at cap still succeeds" assertion in
// the same test because the integration suite calls msgServer directly on
// the raw sdk.Context — there is no BaseApp CacheContext wrapping each msg,
// so any partial state written by applyMemberUpdates before the cap check
// fires persists into the next call. In production this is irrelevant (the
// real msg path runs inside a CacheContext that is discarded on error).
func (suite *IntegrationTestSuite) TestUpdateMembers_RejectsExceedingMaxStaticMembers() {
	creator := freshAddr()
	// Build MaxStaticMembers members. The creator is one of them so the
	// admin signature matches a real voter; the remaining slots are filled
	// with bulk-generated deterministic addresses.
	members := append(
		[]types.StaticMember{{Address: creator, Weight: 1}},
		bulkStaticMembers(types.MaxStaticMembers-1)...,
	)

	suite.expectAccountCreated(1)
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("at-cap"),
		VotingConfig: staticConfigWithMembers(members),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	// Sanity: the DAO is at exactly MaxStaticMembers.
	total, err := suite.k.TotalVotingPower(suite.ctx, &types.QueryTotalVotingPowerRequest{DaoId: resp.DaoId})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(types.MaxStaticMembers), total.Total)

	// Adding ONE new member must be rejected by the post-update cap check.
	extra := freshAddr()
	_, err = suite.msgServer.UpdateMembers(suite.ctx, &types.MsgUpdateMembers{
		Authority: creator,
		DaoId:     resp.DaoId,
		Add:       []types.StaticMember{{Address: extra, Weight: 1}},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "exceeds max")
}
