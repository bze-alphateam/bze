package keeper_test

import (
	keeper2 "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/testutil"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
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
}

func (suite *IntegrationTestSuite) SetupTest() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	require.NotNil(t, mockBank)
	mockDistr := testutil.NewMockDistrKeeper(mockCtrl)
	require.NotNil(t, mockDistr)

	k, ctx := keeper2.CointrunkKeeper(t, mockBank, mockDistr)
	suite.ctx = ctx
	suite.k = &k
	suite.bank = mockBank
	suite.distr = mockDistr
	suite.msgServer = keeper.NewMsgServerImpl(k)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
