package keeper_test

import (
	"fmt"
	"testing"

	keeper2 "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/testutil"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	denomBze   = "ubze"
	denomStake = "stake"
)

func getMarketId() string {
	return fmt.Sprintf("%s/%s", market.Base, market.Quote)
}

var market = types.Market{
	Base:    denomStake,
	Quote:   denomBze,
	Creator: "bze1m33n82r5x3eyjmjtwjkl82zzdlrnv8pevd8u9r",
}

type IntegrationTestSuite struct {
	suite.Suite

	ctx         sdk.Context
	k           *keeper.Keeper
	msgServer   types.MsgServer
	bankMock    *testutil.MockBankKeeper
	accountMock *testutil.MockAccountKeeper
}

func (suite *IntegrationTestSuite) SetupTest() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	require.NotNil(t, mockBank)
	mockAccount := testutil.NewMockAccountKeeper(mockCtrl)
	require.NotNil(t, mockAccount)

	k, ctx := keeper2.TradebinKeeper(t, mockBank, mockAccount)
	suite.ctx = ctx
	suite.k = &k
	suite.bankMock = mockBank
	suite.accountMock = mockAccount
	suite.msgServer = keeper.NewMsgServerImpl(k)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
