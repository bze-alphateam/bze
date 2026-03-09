package keeper_test

import (
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	"github.com/bze-alphateam/bze/x/txfeecollector/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	denomBze   = "ubze"
	denomStake = "stake"
	denomOther = "other"
)

type IntegrationTestSuite struct {
	suite.Suite

	ctx         sdk.Context
	k           keeper.Keeper
	bankMock    *testutil.MockBankKeeper
	accountMock *testutil.MockAccountKeeper
	tradeMock   *testutil.MockTradeKeeper
	distrMock   *testutil.MockDistrKeeper
}

func (suite *IntegrationTestSuite) SetupTest() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	require.NotNil(t, mockBank)
	mockAccount := testutil.NewMockAccountKeeper(mockCtrl)
	require.NotNil(t, mockAccount)
	mockTrade := testutil.NewMockTradeKeeper(mockCtrl)
	require.NotNil(t, mockTrade)
	mockDistr := testutil.NewMockDistrKeeper(mockCtrl)
	require.NotNil(t, mockDistr)

	k, ctx := keepertest.TxfeecollectorKeeperWithMocks(t, mockBank, mockAccount, mockTrade, mockDistr)
	suite.ctx = ctx
	suite.k = k
	suite.bankMock = mockBank
	suite.accountMock = mockAccount
	suite.tradeMock = mockTrade
	suite.distrMock = mockDistr
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
