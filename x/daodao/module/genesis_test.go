package daodao_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/keeper"
	daodao "github.com/bze-alphateam/bze/x/daodao/module"
	"github.com/bze-alphateam/bze/x/daodao/testutil"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// GenesisTestSuite isolates module-level Init/Export tests from the keeper
// package. Mocks are reset per-test via SetupTest.
type GenesisTestSuite struct {
	suite.Suite

	k       keeper.Keeper
	ctx     sdk.Context
	acc     *testutil.MockAccountKeeper
	bank    *testutil.MockBankKeeper
	distr   *testutil.MockDistrKeeper
	rewards *testutil.MockRewardsKeeper
}

func (suite *GenesisTestSuite) SetupTest() {
	t := suite.T()
	// No `defer ctrl.Finish()` — gomock v0.4+ registers Finish via t.Cleanup,
	// which fires AFTER the test method completes. A manual defer would
	// finalize the controller before expectations are set in the test body.
	ctrl := gomock.NewController(t)

	suite.acc = testutil.NewMockAccountKeeper(ctrl)
	suite.bank = testutil.NewMockBankKeeper(ctrl)
	suite.distr = testutil.NewMockDistrKeeper(ctrl)
	suite.rewards = testutil.NewMockRewardsKeeper(ctrl)

	k, ctx := keepertest.DaodaoKeeper(t, suite.acc, suite.bank, suite.distr, suite.rewards)
	suite.k = k
	suite.ctx = ctx
}

func TestGenesisSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

// TestGenesis_DefaultsRoundTrip: defaults survive Init/Export with no DAOs.
func (suite *GenesisTestSuite) TestGenesis_DefaultsRoundTrip() {
	genesisState := types.GenesisState{
		Params:       types.DefaultParams(),
		DaoIdCounter: 1,
	}

	daodao.InitGenesis(suite.ctx, suite.k, genesisState)
	got := daodao.ExportGenesis(suite.ctx, suite.k)
	suite.Require().NotNil(got)
	suite.Require().Equal(genesisState.Params, got.Params)
	suite.Require().Equal(genesisState.DaoIdCounter, got.DaoIdCounter)
	suite.Require().Empty(got.Daos)
}

// TestGenesis_RoundTripWithDaos seeds a parent + child and verifies they
// survive export → import → export.
func (suite *GenesisTestSuite) TestGenesis_RoundTripWithDaos() {
	parentID := uint64(1)
	childID := uint64(2)
	creator := sample.AccAddress()

	gov := types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
		ThresholdBps: 5_000,
		QuorumBps:    0,
		VotingPeriod: 24 * time.Hour,
		AllowRevote:  true,
	}
	dep := types.DepositConfig{
		MinDeposit:         sdk.NewInt64Coin("ubze", 1),
		DepositPeriod:      7 * 24 * time.Hour,
		ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
		VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_ON_PASS,
	}

	parent := types.Dao{
		Id:             parentID,
		Metadata:       types.DaoMetadata{Name: "parent"},
		Creator:        creator,
		AccountAddress: types.DaoAccountAddress(parentID).String(),
		Admin:          creator,
		CreatedAtBlock: 5,
		VotingBackend:  types.VotingBackendType_VOTING_BACKEND_STATIC,
		Governance:     gov,
		Deposit:        dep,
	}
	child := types.Dao{
		Id:             childID,
		Metadata:       types.DaoMetadata{Name: "child"},
		Creator:        creator,
		AccountAddress: types.DaoAccountAddress(childID).String(),
		Admin:          parent.AccountAddress,
		ParentDaoId:    parentID,
		CreatedAtBlock: 10,
		VotingBackend:  types.VotingBackendType_VOTING_BACKEND_STATIC,
		Governance:     gov,
		Deposit:        dep,
	}

	// InitGenesis registers a BaseAccount for each DAO. Order matches the
	// daos slice; HasAccount returns false (so we create new accounts).
	for _, d := range []types.Dao{parent, child} {
		addr := types.DaoAccountAddress(d.Id)
		suite.acc.EXPECT().HasAccount(gomock.Any(), addr).Return(false).Times(1)
		suite.acc.EXPECT().NewAccountWithAddress(gomock.Any(), addr).
			Return(authtypes.NewBaseAccountWithAddress(addr)).Times(1)
		suite.acc.EXPECT().SetAccount(gomock.Any(), gomock.Any()).Times(1)
	}

	genesisState := types.GenesisState{
		Params:       types.DefaultParams(),
		DaoIdCounter: 3,
		Daos:         []types.Dao{parent, child},
		StaticMembers: []types.StaticMemberEntry{
			{DaoId: parentID, Address: creator, Weight: 1},
			{DaoId: childID, Address: creator, Weight: 1},
		},
	}
	require.NoError(suite.T(), genesisState.Validate())

	daodao.InitGenesis(suite.ctx, suite.k, genesisState)
	got := daodao.ExportGenesis(suite.ctx, suite.k)
	suite.Require().NotNil(got)

	suite.Require().Equal(genesisState.Params, got.Params)
	suite.Require().Equal(genesisState.DaoIdCounter, got.DaoIdCounter)
	suite.Require().Len(got.Daos, 2)
	suite.Require().Equal(parent, got.Daos[0])
	suite.Require().Equal(child, got.Daos[1])
}
