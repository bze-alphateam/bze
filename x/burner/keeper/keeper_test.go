package keeper_test

import (
	keeper2 "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/burner/keeper"
	"github.com/bze-alphateam/bze/x/burner/testutil"
	"github.com/bze-alphateam/bze/x/burner/types"
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
	epoch     *testutil.MockEpochKeeper
	bank      *testutil.MockBankKeeper
	acc       *testutil.MockAccountKeeper
}

func (suite *IntegrationTestSuite) SetupTest() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	require.NotNil(t, mockBank)
	mockAcc := testutil.NewMockAccountKeeper(mockCtrl)
	require.NotNil(t, mockAcc)
	mockEpoch := testutil.NewMockEpochKeeper(mockCtrl)
	require.NotNil(t, mockEpoch)

	k, ctx := keeper2.BurnerKeeper(t, mockBank, mockAcc, mockEpoch)
	suite.ctx = ctx
	suite.k = &k
	suite.epoch = mockEpoch
	suite.bank = mockBank
	suite.acc = mockAcc
	suite.msgServer = keeper.NewMsgServerImpl(k)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
