package daodao_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/testutil/sample"
	daodao "github.com/bze-alphateam/bze/x/daodao/module"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// minimalGovernance is a Validate-passing default governance config shared
// by the genesis fixtures below.
func minimalGovernance() types.GovernanceConfig {
	return types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
		ThresholdBps: 5_000,
		QuorumBps:    0,
		VotingPeriod: 24 * time.Hour,
		AllowRevote:  true,
	}
}

// minimalDeposit is the DepositConfig sibling of minimalGovernance.
// Min_deposit = 1ubze, period 7d, treasury forfeit, ON_PASS refund.
func minimalDeposit() types.DepositConfig {
	return types.DepositConfig{
		MinDeposit:         sdk.NewInt64Coin("ubze", 1),
		DepositPeriod:      7 * 24 * time.Hour,
		ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
		VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_ON_PASS,
	}
}

// metDepositColl is the deposit_collected value for fixtures where the
// open proposal/poll is in VOTING (or beyond) — must be >= min_deposit
// (1ubze in these fixtures) to satisfy the deposit-gate invariant
// genesis enforces. For DEPOSIT_PERIOD fixtures use sub-min (e.g. 0ubze).
func metDepositColl() sdk.Coin {
	return sdk.NewInt64Coin("ubze", 1)
}

// belowMinDepositColl is the deposit_collected value for fixtures whose
// open proposal/poll sits in DEPOSIT_PERIOD — must be < min_deposit to
// satisfy "DEPOSIT_PERIOD means not-yet-promoted to VOTING".
func belowMinDepositColl() sdk.Coin {
	return sdk.NewInt64Coin("ubze", 0)
}

// TestGenesis_ProposalRoundTrip: a single DAO with one VOTING + one PASSED
// proposal, including votes and the matching snapshot rows, survives
// export → import → export with byte-identical data.
func (suite *GenesisTestSuite) TestGenesis_ProposalRoundTrip() {
	daoID := uint64(1)
	creator := sample.AccAddress()
	addr1 := sample.AccAddress()
	addr2 := sample.AccAddress()

	dao := types.Dao{
		Id:             daoID,
		Metadata:       types.DaoMetadata{Name: "round-trip"},
		Creator:        creator,
		AccountAddress: types.DaoAccountAddress(daoID).String(),
		Admin:          creator,
		CreatedAtBlock: 1,
		VotingBackend:  types.VotingBackendType_VOTING_BACKEND_STATIC,
		Governance:     minimalGovernance(),
		Deposit:        minimalDeposit(),
	}

	// Two proposals on the same DAO, each at snapshot 1 and 2 respectively.
	// Tally totals match what SnapshotTotal will report on import.
	prop1 := types.Proposal{
		DaoId:              daoID,
		ProposalId:         1,
		Proposer:           creator,
		Title:              "first",
		Description:        "",
		SnapshotId:         1,
		CreatedHeight:      10,
		VotingEnd:          suite.ctx.BlockTime().Add(48 * time.Hour),
		Status:             types.ProposalStatus_PROPOSAL_STATUS_VOTING,
		Tally:              types.Tally{YesPower: 0, NoPower: 0, TotalPower: 5},
		GovernanceSnapshot: minimalGovernance(),
		DepositSnapshot:    minimalDeposit(),
		// Open proposal — must have collected >= min_deposit (1ubze) and
		// matching DepositRecord rows so the genesis-validation deposit
		// gate is satisfied.
		DepositCollected: metDepositColl(),
	}
	// prop2 passed because addr1 (power 3) voted YES — 3/5 = 60% >= 50%
	// threshold. Tally must reconcile against the votes below.
	prop2 := types.Proposal{
		DaoId:              daoID,
		ProposalId:         2,
		Proposer:           creator,
		Title:              "second",
		Description:        "",
		SnapshotId:         2,
		CreatedHeight:      20,
		VotingEnd:          suite.ctx.BlockTime().Add(-time.Hour), // already past
		Status:             types.ProposalStatus_PROPOSAL_STATUS_PASSED,
		Tally:              types.Tally{YesPower: 3, TotalPower: 5},
		GovernanceSnapshot: minimalGovernance(),
		DepositSnapshot:    minimalDeposit(),
		// Terminal status — records have been disbursed; deposit_collected
		// is historical metadata and must still satisfy "collected >= min".
		DepositCollected: metDepositColl(),
	}

	vote := types.Vote{
		DaoId:       daoID,
		ProposalId:  2,
		Voter:       addr1,
		Option:      types.VoteOption_VOTE_OPTION_YES,
		Power:       3, // matches snapshot row for (1, 2, addr1)
		VotedHeight: 21,
	}

	snapPowers := []types.SnapshotPowerEntry{
		{DaoId: daoID, SnapshotId: 1, Address: addr1, Power: 3},
		{DaoId: daoID, SnapshotId: 1, Address: addr2, Power: 2},
		{DaoId: daoID, SnapshotId: 2, Address: addr1, Power: 3},
		{DaoId: daoID, SnapshotId: 2, Address: addr2, Power: 2},
	}
	snapTotals := []types.SnapshotTotalEntry{
		{DaoId: daoID, SnapshotId: 1, Total: 5},
		{DaoId: daoID, SnapshotId: 2, Total: 5},
	}

	gs := types.GenesisState{
		Params:             types.DefaultParams(),
		DaoIdCounter:       2,
		Daos:               []types.Dao{dao},
		// Post-review Finding 2: STATIC DAOs require member entries.
		StaticMembers: []types.StaticMemberEntry{{DaoId: daoID, Address: creator, Weight: 1}},
		Proposals:          []types.Proposal{prop1, prop2},
		Votes:              []types.Vote{vote},
		ProposalIdCounters: []types.PerDaoUint64{{DaoId: daoID, Value: 3}}, // > max(prop_id)=2
		SnapshotIdCounters: []types.PerDaoUint64{{DaoId: daoID, Value: 3}}, // > max(snap_id)=2
		SnapshotPowers:     snapPowers,
		SnapshotTotals:     snapTotals,
		// prop1 is open (VOTING) → must have a DepositRecord summing to its
		// deposit_collected. prop2 is terminal → records already disbursed.
		DepositRecords: []types.DepositRecord{
			{DaoId: daoID, ProposalId: 1, Depositor: creator, Amount: metDepositColl()},
		},
	}
	suite.Require().NoError(gs.Validate())

	// Mock account creation: InitGenesis registers a BaseAccount per DAO.
	daoAddr := types.DaoAccountAddress(daoID)
	suite.acc.EXPECT().HasAccount(gomock.Any(), daoAddr).Return(false).Times(1)
	suite.acc.EXPECT().NewAccountWithAddress(gomock.Any(), daoAddr).
		Return(authtypes.NewBaseAccountWithAddress(daoAddr)).Times(1)
	suite.acc.EXPECT().SetAccount(gomock.Any(), gomock.Any()).Times(1)

	daodao.InitGenesis(suite.ctx, suite.k, gs)
	got := daodao.ExportGenesis(suite.ctx, suite.k)
	suite.Require().NotNil(got)

	// Compare per-field rather than the whole struct so a diff failure
	// points at the exact section.
	//
	// Snapshot powers / vote rows are stored under address-bytes-suffixed
	// keys; the iteration order depends on the random sample-address bytes
	// rather than fixture source order, so we use ElementsMatch here
	// (order-independent). Other tables iterate by deterministic numeric
	// keys, so Equal is appropriate.
	suite.Require().Equal(gs.Params, got.Params)
	suite.Require().Equal(gs.DaoIdCounter, got.DaoIdCounter)
	suite.Require().Equal(gs.Daos, got.Daos)
	suite.Require().Equal(gs.Proposals, got.Proposals)
	suite.Require().ElementsMatch(gs.Votes, got.Votes)
	suite.Require().Equal(gs.ProposalIdCounters, got.ProposalIdCounters)
	suite.Require().Equal(gs.SnapshotIdCounters, got.SnapshotIdCounters)
	suite.Require().ElementsMatch(gs.SnapshotPowers, got.SnapshotPowers)
	suite.Require().Equal(gs.SnapshotTotals, got.SnapshotTotals)
	suite.Require().ElementsMatch(gs.DepositRecords, got.DepositRecords)
}

// TestGenesis_RebuildsStatusIndexOnImport: a status-filtered Proposals
// query after import returns proposals correctly — proving the
// ProposalByStatusKey index was rebuilt rather than copied.
func (suite *GenesisTestSuite) TestGenesis_RebuildsStatusIndexOnImport() {
	daoID := uint64(1)
	creator := sample.AccAddress()

	dao := types.Dao{
		Id:             daoID,
		Metadata:       types.DaoMetadata{Name: "status-idx"},
		Creator:        creator,
		AccountAddress: types.DaoAccountAddress(daoID).String(),
		Admin:          creator,
		VotingBackend:  types.VotingBackendType_VOTING_BACKEND_STATIC,
		Governance:     minimalGovernance(),
		Deposit:        minimalDeposit(),
	}
	votingProp := types.Proposal{
		DaoId: daoID, ProposalId: 1, Proposer: creator, Title: "live",
		SnapshotId: 1, VotingEnd: suite.ctx.BlockTime().Add(time.Hour),
		Status:             types.ProposalStatus_PROPOSAL_STATUS_VOTING,
		Tally:              types.Tally{TotalPower: 1},
		GovernanceSnapshot: minimalGovernance(),
		DepositSnapshot:    minimalDeposit(),
		DepositCollected:   metDepositColl(), // open VOTING needs collected >= min
	}
	// Closed proposal with no votes → REJECTED with zero tally counts. Keeps
	// the fixture vote-free while still exercising the status-index rebuild.
	passedProp := types.Proposal{
		DaoId: daoID, ProposalId: 2, Proposer: creator, Title: "done",
		SnapshotId: 1, VotingEnd: suite.ctx.BlockTime().Add(-time.Hour),
		Status:             types.ProposalStatus_PROPOSAL_STATUS_REJECTED,
		Tally:              types.Tally{TotalPower: 1},
		GovernanceSnapshot: minimalGovernance(),
		DepositSnapshot:    minimalDeposit(),
		DepositCollected:   metDepositColl(), // historical record; terminal disbursed records
	}

	gs := types.GenesisState{
		Params:       types.DefaultParams(),
		DaoIdCounter: 2,
		Daos:         []types.Dao{dao},
		// Post-review Finding 2: STATIC DAOs require member entries.
		StaticMembers: []types.StaticMemberEntry{{DaoId: daoID, Address: creator, Weight: 1}},
		Proposals:    []types.Proposal{votingProp, passedProp},
		ProposalIdCounters: []types.PerDaoUint64{{DaoId: daoID, Value: 3}},
		SnapshotIdCounters: []types.PerDaoUint64{{DaoId: daoID, Value: 2}},
		SnapshotTotals: []types.SnapshotTotalEntry{
			{DaoId: daoID, SnapshotId: 1, Total: 1},
		},
		// votingProp is open → needs a record summing to its deposit_collected.
		// passedProp is terminal → records disbursed.
		DepositRecords: []types.DepositRecord{
			{DaoId: daoID, ProposalId: 1, Depositor: creator, Amount: metDepositColl()},
		},
	}
	suite.Require().NoError(gs.Validate())

	daoAddr := types.DaoAccountAddress(daoID)
	suite.acc.EXPECT().HasAccount(gomock.Any(), daoAddr).Return(false).Times(1)
	suite.acc.EXPECT().NewAccountWithAddress(gomock.Any(), daoAddr).
		Return(authtypes.NewBaseAccountWithAddress(daoAddr)).Times(1)
	suite.acc.EXPECT().SetAccount(gomock.Any(), gomock.Any()).Times(1)

	daodao.InitGenesis(suite.ctx, suite.k, gs)

	// Status-filtered query exercises the ProposalByStatusKey index.
	respVoting, err := suite.k.Proposals(suite.ctx, &types.QueryProposalsRequest{
		DaoId:        daoID,
		StatusFilter: types.ProposalStatus_PROPOSAL_STATUS_VOTING,
	})
	suite.Require().NoError(err)
	suite.Require().Len(respVoting.Proposals, 1)
	suite.Require().Equal(uint64(1), respVoting.Proposals[0].ProposalId)

	respRejected, err := suite.k.Proposals(suite.ctx, &types.QueryProposalsRequest{
		DaoId:        daoID,
		StatusFilter: types.ProposalStatus_PROPOSAL_STATUS_REJECTED,
	})
	suite.Require().NoError(err)
	suite.Require().Len(respRejected.Proposals, 1)
	suite.Require().Equal(uint64(2), respRejected.Proposals[0].ProposalId)
}

// TestGenesis_RebuildsExpiringQueueOnImport: after import, the end-blocker
// finalizes the imported VOTING proposal at its voting_end. This proves
// the ExpiringProposalKey queue was rebuilt from the proposal record.
func (suite *GenesisTestSuite) TestGenesis_RebuildsExpiringQueueOnImport() {
	daoID := uint64(1)
	creator := sample.AccAddress()

	dao := types.Dao{
		Id: daoID, Metadata: types.DaoMetadata{Name: "expq"},
		Creator: creator, AccountAddress: types.DaoAccountAddress(daoID).String(),
		Admin: creator,
		VotingBackend: types.VotingBackendType_VOTING_BACKEND_STATIC,
		Governance:    minimalGovernance(),
		Deposit:       minimalDeposit(),
	}
	prop := types.Proposal{
		DaoId: daoID, ProposalId: 1, Proposer: creator, Title: "expq",
		SnapshotId: 1,
		// voting_end is BEFORE the current block time so the next EndBlock fires it.
		VotingEnd:          suite.ctx.BlockTime().Add(-time.Second),
		Status:             types.ProposalStatus_PROPOSAL_STATUS_VOTING,
		Tally:              types.Tally{TotalPower: 1},
		GovernanceSnapshot: minimalGovernance(),
		DepositSnapshot:    minimalDeposit(),
		DepositCollected:   metDepositColl(),
	}
	gs := types.GenesisState{
		Params:       types.DefaultParams(),
		DaoIdCounter: 2,
		Daos:         []types.Dao{dao},
		// Post-review Finding 2: STATIC DAOs require member entries.
		StaticMembers: []types.StaticMemberEntry{{DaoId: daoID, Address: creator, Weight: 1}},
		Proposals:    []types.Proposal{prop},
		ProposalIdCounters: []types.PerDaoUint64{{DaoId: daoID, Value: 2}},
		SnapshotIdCounters: []types.PerDaoUint64{{DaoId: daoID, Value: 2}},
		SnapshotTotals:     []types.SnapshotTotalEntry{{DaoId: daoID, SnapshotId: 1, Total: 1}},
		DepositRecords: []types.DepositRecord{
			{DaoId: daoID, ProposalId: 1, Depositor: creator, Amount: metDepositColl()},
		},
	}
	suite.Require().NoError(gs.Validate())

	daoAddr := types.DaoAccountAddress(daoID)
	suite.acc.EXPECT().HasAccount(gomock.Any(), daoAddr).Return(false).Times(1)
	suite.acc.EXPECT().NewAccountWithAddress(gomock.Any(), daoAddr).
		Return(authtypes.NewBaseAccountWithAddress(daoAddr)).Times(1)
	suite.acc.EXPECT().SetAccount(gomock.Any(), gomock.Any()).Times(1)

	daodao.InitGenesis(suite.ctx, suite.k, gs)

	// REJECTED + ON_PASS refund policy (minimalDeposit default) → forfeit
	// the 1ubze deposit to the DAO's treasury at end-block.
	escrow := types.DepositEscrowAddress(daoID)
	treasury, _ := sdk.AccAddressFromBech32(dao.AccountAddress)
	suite.bank.EXPECT().
		SendCoins(gomock.Any(), escrow, treasury, sdk.NewCoins(metDepositColl())).
		Return(nil).
		Times(1)

	// Advance the block time forward and fire EndBlock — the imported
	// VOTING proposal should transition to REJECTED (no votes, threshold 50%).
	newCtx := suite.ctx.WithBlockTime(suite.ctx.BlockTime().Add(time.Hour))
	suite.Require().NoError(suite.k.EndBlock(newCtx))

	out, ok := suite.k.GetProposal(newCtx, daoID, 1)
	suite.Require().True(ok)
	suite.Require().Equal(types.ProposalStatus_PROPOSAL_STATUS_REJECTED, out.Status,
		"imported proposal should have been re-enqueued and finalized by end-blocker")
}

// TestGenesis_StaticMembersRoundTrip (Finding 2 fix): a STATIC DAO with
// a multi-member set survives export → import. Asserts that on the
// post-import keeper:
//
//   - Every member row is restored (weight intact).
//   - The cached StaticTotalPowerKey is recomputed correctly (sum of
//     weights), even though it isn't serialized — that's the contract
//     of InitStaticMembers / applyStaticMembersInit.
//   - VotingPower queries return the right per-address weights.
//
// Without this round-trip, a STATIC DAO imported from genesis lands
// with `voting_backend = STATIC` but an empty MemberKey range, so
// proposals/polls would see total_power = 0 and the DAO would be
// effectively bricked.
func (suite *GenesisTestSuite) TestGenesis_StaticMembersRoundTrip() {
	daoID := uint64(1)
	creator := sample.AccAddress()
	other := sample.AccAddress()

	dao := types.Dao{
		Id:             daoID,
		Metadata:       types.DaoMetadata{Name: "members"},
		Creator:        creator,
		AccountAddress: types.DaoAccountAddress(daoID).String(),
		Admin:          creator,
		CreatedAtBlock: 7,
		VotingBackend:  types.VotingBackendType_VOTING_BACKEND_STATIC,
		Governance:     minimalGovernance(),
		Deposit:        minimalDeposit(),
	}

	gs := types.GenesisState{
		Params:       types.DefaultParams(),
		DaoIdCounter: 2,
		Daos:         []types.Dao{dao},
		StaticMembers: []types.StaticMemberEntry{
			{DaoId: daoID, Address: creator, Weight: 3},
			{DaoId: daoID, Address: other, Weight: 5},
		},
	}
	suite.Require().NoError(gs.Validate())

	// InitGenesis registers a BaseAccount per DAO.
	daoAddr := types.DaoAccountAddress(daoID)
	suite.acc.EXPECT().HasAccount(gomock.Any(), daoAddr).Return(false).Times(1)
	suite.acc.EXPECT().NewAccountWithAddress(gomock.Any(), daoAddr).
		Return(authtypes.NewBaseAccountWithAddress(daoAddr)).Times(1)
	suite.acc.EXPECT().SetAccount(gomock.Any(), gomock.Any()).Times(1)

	daodao.InitGenesis(suite.ctx, suite.k, gs)

	// Per-address weights survive.
	creatorAddr, _ := sdk.AccAddressFromBech32(creator)
	otherAddr, _ := sdk.AccAddressFromBech32(other)
	gotCreator, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{
		DaoId: daoID, Address: creatorAddr.String(),
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(3), gotCreator.Power)
	gotOther, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{
		DaoId: daoID, Address: otherAddr.String(),
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(5), gotOther.Power)

	// Cached total = 3 + 5 = 8. Recomputed by InitStaticMembers, NOT
	// taken from genesis (we don't serialize the cached total).
	total, err := suite.k.TotalVotingPower(suite.ctx, &types.QueryTotalVotingPowerRequest{DaoId: daoID})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(8), total.Total)

	// Export reproduces the same StaticMembers (order-independent — the
	// store iterates byte-sorted, fixture source order can differ).
	got := daodao.ExportGenesis(suite.ctx, suite.k)
	suite.Require().ElementsMatch(gs.StaticMembers, got.StaticMembers)
}

// TestGenesis_RoundTripIdempotent: export → import → export reproduces the
// same GenesisState. Belt-and-suspenders for byte stability.
func (suite *GenesisTestSuite) TestGenesis_RoundTripIdempotent() {
	daoID := uint64(1)
	creator := sample.AccAddress()
	voter := sample.AccAddress()

	dao := types.Dao{
		Id: daoID, Metadata: types.DaoMetadata{Name: "idem"}, Creator: creator,
		AccountAddress: types.DaoAccountAddress(daoID).String(),
		Admin: creator,
		VotingBackend: types.VotingBackendType_VOTING_BACKEND_STATIC,
		Governance:    minimalGovernance(),
		Deposit:       minimalDeposit(),
	}
	// Single voter (`voter`, snapshot power 1) cast a YES. Tally must
	// reflect that to satisfy the genesis vote-reconciliation pass.
	prop := types.Proposal{
		DaoId: daoID, ProposalId: 1, Proposer: creator, Title: "i",
		SnapshotId: 1, VotingEnd: suite.ctx.BlockTime().Add(time.Hour),
		Status: types.ProposalStatus_PROPOSAL_STATUS_VOTING,
		Tally:  types.Tally{YesPower: 1, TotalPower: 1},
		GovernanceSnapshot: minimalGovernance(),
		DepositSnapshot:    minimalDeposit(),
		DepositCollected:   metDepositColl(),
	}
	gs := types.GenesisState{
		Params:       types.DefaultParams(),
		DaoIdCounter: 2,
		Daos:         []types.Dao{dao},
		// Post-review Finding 2: STATIC DAOs require member entries.
		StaticMembers: []types.StaticMemberEntry{{DaoId: daoID, Address: creator, Weight: 1}},
		Proposals:    []types.Proposal{prop},
		Votes: []types.Vote{{
			DaoId: daoID, ProposalId: 1, Voter: voter,
			Option: types.VoteOption_VOTE_OPTION_YES, Power: 1, VotedHeight: 5,
		}},
		ProposalIdCounters: []types.PerDaoUint64{{DaoId: daoID, Value: 2}},
		SnapshotIdCounters: []types.PerDaoUint64{{DaoId: daoID, Value: 2}},
		SnapshotPowers: []types.SnapshotPowerEntry{
			{DaoId: daoID, SnapshotId: 1, Address: voter, Power: 1},
		},
		SnapshotTotals: []types.SnapshotTotalEntry{
			{DaoId: daoID, SnapshotId: 1, Total: 1},
		},
		DepositRecords: []types.DepositRecord{
			{DaoId: daoID, ProposalId: 1, Depositor: creator, Amount: metDepositColl()},
		},
	}
	suite.Require().NoError(gs.Validate())

	daoAddr := types.DaoAccountAddress(daoID)
	suite.acc.EXPECT().HasAccount(gomock.Any(), daoAddr).Return(false).Times(1)
	suite.acc.EXPECT().NewAccountWithAddress(gomock.Any(), daoAddr).
		Return(authtypes.NewBaseAccountWithAddress(daoAddr)).Times(1)
	suite.acc.EXPECT().SetAccount(gomock.Any(), gomock.Any()).Times(1)

	daodao.InitGenesis(suite.ctx, suite.k, gs)
	first := daodao.ExportGenesis(suite.ctx, suite.k)

	// Re-import on a fresh keeper, then export again — bytes must match.
	suite.SetupTest() // reset
	daoAddr = types.DaoAccountAddress(daoID)
	suite.acc.EXPECT().HasAccount(gomock.Any(), daoAddr).Return(false).Times(1)
	suite.acc.EXPECT().NewAccountWithAddress(gomock.Any(), daoAddr).
		Return(authtypes.NewBaseAccountWithAddress(daoAddr)).Times(1)
	suite.acc.EXPECT().SetAccount(gomock.Any(), gomock.Any()).Times(1)
	daodao.InitGenesis(suite.ctx, suite.k, *first)
	second := daodao.ExportGenesis(suite.ctx, suite.k)

	suite.Require().Equal(first, second)

	// Sanity: the round-trip preserves the voter as a member of snapshot 1.
	voterAddr, err := sdk.AccAddressFromBech32(voter)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1), suite.k.SnapshotPower(suite.ctx, daoID, 1, voterAddr))
}
