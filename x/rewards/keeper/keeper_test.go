package keeper_test

import (
	keeper2 "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"github.com/bze-alphateam/bze/x/rewards/types"
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
	epoch     *testutil.MockEpochKeeper
	trade     *testutil.MockTradingKeeper
}

func (suite *IntegrationTestSuite) SetupTest() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	require.NotNil(t, mockBank)
	mockDistr := testutil.NewMockDistrKeeper(mockCtrl)
	require.NotNil(t, mockDistr)
	mockEpoch := testutil.NewMockEpochKeeper(mockCtrl)
	require.NotNil(t, mockEpoch)
	trade := testutil.NewMockTradingKeeper(mockCtrl)
	require.NotNil(t, trade)

	k, ctx := keeper2.RewardsKeeper(t, mockBank, mockDistr, mockEpoch, trade)
	suite.ctx = ctx
	suite.k = &k
	suite.bank = mockBank
	suite.distr = mockDistr
	suite.epoch = mockEpoch
	suite.trade = trade
	suite.msgServer = keeper.NewMsgServerImpl(k)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
