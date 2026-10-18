package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/dao/keeper"
	"github.com/bze-alphateam/bze/x/dao/testutil"
	"github.com/bze-alphateam/bze/x/dao/types"
)

// IntegrationTestSuite is the x/dao keeper suite: a real keeper on an in-memory
// store, with gomock dependency keepers. Every message and query test of the
// module is a method on it.
type IntegrationTestSuite struct {
	suite.Suite

	ctx       sdk.Context
	k         *keeper.Keeper
	msgServer types.MsgServer
	acc       *testutil.MockAccountKeeper
	bank      *testutil.MockBankKeeper
	rewards   *testutil.MockRewardsKeeper
	router    *baseapp.MsgServiceRouter
	ledger    *fakeBank
	// stakes is the x/rewards participation served once stubStakes ran
	stakes *stakes
}

func (suite *IntegrationTestSuite) SetupTest() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)

	suite.acc = testutil.NewMockAccountKeeper(mockCtrl)
	suite.bank = testutil.NewMockBankKeeper(mockCtrl)
	suite.rewards = testutil.NewMockRewardsKeeper(mockCtrl)
	suite.stakes = nil

	cdc := keepertest.DaoCodec()
	suite.router = baseapp.NewMsgServiceRouter()
	suite.router.SetInterfaceRegistry(cdc.InterfaceRegistry())

	k, ctx := keepertest.DaoKeeper(t, cdc, suite.router, suite.acc, suite.bank, suite.rewards)
	suite.ctx = ctx
	suite.k = &k
	suite.msgServer = keeper.NewMsgServerImpl(suite.k)

	suite.ledger = newFakeBank()
	banktypes.RegisterMsgServer(suite.router, suite.ledger)
	types.RegisterMsgServer(suite.router, suite.msgServer)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func TestNewKeeper_InvalidAuthorityPanics(t *testing.T) {
	require.Panics(t, func() {
		keeper.NewKeeper(nil, nil, nil, "not-an-address", nil, nil, nil, nil)
	})
}
