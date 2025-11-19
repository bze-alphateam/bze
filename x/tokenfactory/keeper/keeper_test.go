package keeper_test

import (
	keeper2 "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/tokenfactory/keeper"
	"github.com/bze-alphateam/bze/x/tokenfactory/testutil"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"testing"
)

type IntegrationTestSuite struct {
	suite.Suite

	ctx       sdk.Context
	k         *keeper.Keeper
	msgServer types.MsgServer
	bank      *testutil.MockBankKeeper
	distr     *testutil.MockDistrKeeper
	acc       *testutil.MockAccountKeeper
}

func (suite *IntegrationTestSuite) SetupTest() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	require.NotNil(t, mockBank)
	mockDistr := testutil.NewMockDistrKeeper(mockCtrl)
	require.NotNil(t, mockDistr)
	mockAcc := testutil.NewMockAccountKeeper(mockCtrl)
	require.NotNil(t, mockAcc)

	k, ctx := keeper2.TokenfactoryKeeper(t, mockBank, mockDistr, mockAcc)
	suite.ctx = ctx
	suite.k = &k
	suite.bank = mockBank
	suite.distr = mockDistr
	suite.acc = mockAcc
	suite.msgServer = keeper.NewMsgServerImpl(k)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
