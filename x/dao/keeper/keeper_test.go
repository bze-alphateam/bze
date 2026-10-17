package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
}

func (suite *IntegrationTestSuite) SetupTest() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)

	suite.acc = testutil.NewMockAccountKeeper(mockCtrl)
	suite.bank = testutil.NewMockBankKeeper(mockCtrl)

	k, ctx := keepertest.DaoKeeper(t, suite.acc, suite.bank)
	suite.ctx = ctx
	suite.k = &k
	suite.msgServer = keeper.NewMsgServerImpl(suite.k)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func TestNewKeeper_InvalidAuthorityPanics(t *testing.T) {
	require.Panics(t, func() {
		keeper.NewKeeper(nil, nil, nil, "not-an-address", nil, nil)
	})
}
