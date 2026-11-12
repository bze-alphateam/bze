package txfeecollector_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	txfeecollector "github.com/bze-alphateam/bze/x/txfeecollector/module"
	"github.com/bze-alphateam/bze/x/txfeecollector/testutil"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// PauseTestSuite runs EndBlock with a mocked guardian. No keeper expectation is registered up front: while the
// automation is paused any bank, account, trade, distribution or community pool call fails the test.
type PauseTestSuite struct {
	suite.Suite

	ctx      sdk.Context
	k        keeper.Keeper
	am       txfeecollector.AppModule
	bank     *testutil.MockBankKeeper
	account  *testutil.MockAccountKeeper
	trade    *testutil.MockTradeKeeper
	distr    *testutil.MockDistrKeeper
	cp       *testutil.MockDenomCommunityPoolProvider
	guardian *testutil.MockGuardianKeeper
	paused   bool
}

func TestPauseTestSuite(t *testing.T) {
	suite.Run(t, new(PauseTestSuite))
}

func (s *PauseTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.bank = testutil.NewMockBankKeeper(ctrl)
	s.account = testutil.NewMockAccountKeeper(ctrl)
	s.trade = testutil.NewMockTradeKeeper(ctrl)
	s.distr = testutil.NewMockDistrKeeper(ctrl)
	s.cp = testutil.NewMockDenomCommunityPoolProvider(ctrl)
	s.guardian = testutil.NewMockGuardianKeeper(ctrl)
	s.paused = false
	s.guardian.EXPECT().IsAutomationPaused(gomock.Any()).DoAndReturn(func(_ any) bool { return s.paused }).AnyTimes()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	s.Require().NoError(stateStore.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	s.k = keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		s.bank,
		s.account,
		s.trade,
		s.distr,
		s.cp,
		s.guardian,
	)
	s.ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	s.Require().NoError(s.k.SetParams(s.ctx, types.DefaultParams()))
	s.am = txfeecollector.NewAppModule(cdc, s.k, s.account, s.bank, s.distr)
}

// expectBothPasses registers one EndBlock that converts gasFee in the gas fee pass (swap, then send to the fee
// collector) and funds the community pool with cpFee in the community pool pass.
func (s *PauseTestSuite) expectBothPasses(gasFee, cpFee sdk.Coin) {
	swapped := sdk.NewInt64Coin("ubze", 10)
	s.account.EXPECT().GetModuleAddress(types.ModuleName).Return(feeAddr).Times(1)
	s.bank.EXPECT().GetBalancesPage(gomock.Any(), feeAddr, "", gomock.Any()).Return(sdk.NewCoins(gasFee), gasFee.Denom).Times(1)
	s.trade.EXPECT().IsNativeDenom(gomock.Any(), gasFee.Denom).Return(false).Times(1)
	s.trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), gasFee).Return(true).Times(1)
	s.trade.EXPECT().SafeModuleSwapForNativeDenom(gomock.Any(), types.ModuleName, sdk.NewCoins(gasFee)).Return(swapped, nil).Times(1)
	s.bank.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, authtypes.FeeCollectorName, sdk.NewCoins(swapped)).Return(nil).Times(1)

	s.account.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpAddr).Times(1)
	s.bank.EXPECT().GetBalancesPage(gomock.Any(), cpAddr, "", gomock.Any()).Return(sdk.NewCoins(cpFee), cpFee.Denom).Times(1)
	s.trade.EXPECT().IsNativeDenom(gomock.Any(), cpFee.Denom).Return(true).Times(1)
	s.distr.EXPECT().FundCommunityPool(gomock.Any(), sdk.NewCoins(cpFee), cpAddr).Return(nil).Times(1)
}

// While paused, EndBlock reads no balance, swaps, sends and funds nothing, emits no event and writes no state (the
// collector cursors and the params stay as they were).
func (s *PauseTestSuite) TestPaused_DoesNothing() {
	s.paused = true
	params := s.k.GetParams(s.ctx)

	for i := 0; i < 3; i++ {
		s.Require().NoError(s.am.EndBlock(s.ctx))
	}

	s.Require().Empty(s.ctx.EventManager().Events())
	s.Require().Equal(params, s.k.GetParams(s.ctx))
	s.Require().True(s.k.IsAutomationPaused(s.ctx))
}

// Not paused, EndBlock runs both passes exactly as before.
func (s *PauseTestSuite) TestNotPaused_RunsBothPasses() {
	s.expectBothPasses(sdk.NewInt64Coin("ibc/AAA", 100), sdk.NewInt64Coin("ubze", 5))

	s.Require().NoError(s.am.EndBlock(s.ctx))
	s.Require().False(s.k.IsAutomationPaused(s.ctx))
}

// The coins collected during the pause are processed by the first EndBlock after the resume: the conversion is
// balance based, nothing was skipped.
func (s *PauseTestSuite) TestResume_ProcessesAccumulatedCoins() {
	s.paused = true
	s.Require().NoError(s.am.EndBlock(s.ctx))
	s.Require().NoError(s.am.EndBlock(s.ctx))

	s.paused = false
	s.expectBothPasses(sdk.NewInt64Coin("ibc/AAA", 300), sdk.NewInt64Coin("ubze", 15))
	s.Require().NoError(s.am.EndBlock(s.ctx))
}

// A keeper built without a guardian (nil) is never paused.
func TestKeeper_NilGuardianIsNeverPaused(t *testing.T) {
	k := keeper.NewKeeper(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(types.StoreKey)),
		log.NewNopLogger(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil, nil, nil, nil, nil, nil,
	)

	require.False(t, k.IsAutomationPaused(sdk.Context{}))
}
