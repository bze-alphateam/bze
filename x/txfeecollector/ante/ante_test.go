package ante_test

import (
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	"github.com/bze-alphateam/bze/x/txfeecollector/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

const (
	denomBze   = "ubze"
	denomUsd   = "usd"
	denomOther = "other"
)

type AnteTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	k              keeper.Keeper
	bankMock       *testutil.MockBankKeeper
	accountMock    *testutil.MockAccountKeeper
	tradeMock      *testutil.MockTradeKeeper
	feegrantMock   *testutil.MockFeegrantKeeper
	distrMock      *testutil.MockDistrKeeper
	mockCtrl       *gomock.Controller
	nextCalled     bool
	nextCalledWith sdk.Context
}

func (suite *AnteTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.bankMock = testutil.NewMockBankKeeper(suite.mockCtrl)
	require.NotNil(suite.T(), suite.bankMock)
	suite.accountMock = testutil.NewMockAccountKeeper(suite.mockCtrl)
	require.NotNil(suite.T(), suite.accountMock)
	suite.tradeMock = testutil.NewMockTradeKeeper(suite.mockCtrl)
	require.NotNil(suite.T(), suite.tradeMock)
	suite.feegrantMock = testutil.NewMockFeegrantKeeper(suite.mockCtrl)
	require.NotNil(suite.T(), suite.feegrantMock)
	suite.distrMock = testutil.NewMockDistrKeeper(suite.mockCtrl)
	require.NotNil(suite.T(), suite.distrMock)

	k, ctx := keepertest.TxfeecollectorKeeperWithMocks(suite.T(), suite.bankMock, suite.accountMock, suite.tradeMock, suite.distrMock)
	suite.ctx = ctx
	suite.k = k
	suite.nextCalled = false
	suite.nextCalledWith = sdk.Context{}
}

func (suite *AnteTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}

// mockNext is a simple ante handler that just records it was called
func (suite *AnteTestSuite) mockNext() sdk.AnteHandler {
	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		suite.nextCalled = true
		suite.nextCalledWith = ctx
		return ctx, nil
	}
}

// mockFeeTx implements sdk.FeeTx for testing
type mockFeeTx struct {
	fee        sdk.Coins
	gas        uint64
	feePayer   sdk.AccAddress
	feeGranter sdk.AccAddress
	msgs       []sdk.Msg
}

func (m *mockFeeTx) GetMsgs() []sdk.Msg {
	return m.msgs
}

func (m *mockFeeTx) GetMsgsV2() ([]proto.Message, error) {
	// Return empty slice for testing purposes
	// In real scenarios, msgs would be proto.Message types
	return []proto.Message{}, nil
}

func (m *mockFeeTx) ValidateBasic() error {
	return nil
}

func (m *mockFeeTx) GetGas() uint64 {
	return m.gas
}

func (m *mockFeeTx) GetFee() sdk.Coins {
	return m.fee
}

func (m *mockFeeTx) FeePayer() []byte {
	if len(m.feePayer) == 0 {
		return sdk.AccAddress("feepayer____________")
	}
	return m.feePayer
}

func (m *mockFeeTx) FeeGranter() []byte {
	return m.feeGranter
}
