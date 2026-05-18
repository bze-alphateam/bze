package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/daodao/keeper"
	"github.com/bze-alphateam/bze/x/daodao/testutil"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// IntegrationTestSuite is the shared fixture for all daodao keeper-level
// integration tests. Mocks are reset per-test via SetupTest. See
// scripts/mockgen.sh for how the mocks under x/daodao/testutil are
// generated from types/expected_keepers.go.
type IntegrationTestSuite struct {
	suite.Suite

	ctx       sdk.Context
	k         *keeper.Keeper
	msgServer types.MsgServer

	acc     *testutil.MockAccountKeeper
	bank    *testutil.MockBankKeeper
	distr   *testutil.MockDistrKeeper
	rewards *testutil.MockRewardsKeeper
}

// SetupTest wires fresh mocks and a fresh keeper for each test.
//
// We do NOT call `defer mockCtrl.Finish()` here — gomock v0.4+ already
// registers Finish via t.Cleanup, which runs AFTER the test method
// returns. A manual defer at the end of SetupTest would finalize the
// controller before the test sets its expectations, silently letting
// missing-call regressions pass.
func (suite *IntegrationTestSuite) SetupTest() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)

	mockAcc := testutil.NewMockAccountKeeper(mockCtrl)
	require.NotNil(t, mockAcc)
	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	require.NotNil(t, mockBank)
	mockDistr := testutil.NewMockDistrKeeper(mockCtrl)
	require.NotNil(t, mockDistr)
	mockRewards := testutil.NewMockRewardsKeeper(mockCtrl)
	require.NotNil(t, mockRewards)

	k, ctx := keepertest.DaodaoKeeper(t, mockAcc, mockBank, mockDistr, mockRewards)
	suite.ctx = ctx
	suite.k = &k
	suite.acc = mockAcc
	suite.bank = mockBank
	suite.distr = mockDistr
	suite.rewards = mockRewards
	// Pass the SAME pointer to the msg server so Epic 5's SetMsgRouter
	// mutations (set via suite.k or suite.installRouter) are visible to
	// the handlers it dispatches.
	suite.msgServer = keeper.NewMsgServerImpl(suite.k)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
