package txfeecollector_test

import (
	"bytes"
	"context"
	"errors"
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
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	txfeecollector "github.com/bze-alphateam/bze/x/txfeecollector/module"
	"github.com/bze-alphateam/bze/x/txfeecollector/testutil"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

type endBlockMocks struct {
	bank    *testutil.MockBankKeeper
	account *testutil.MockAccountKeeper
	trade   *testutil.MockTradeKeeper
	distr   *testutil.MockDistrKeeper
	cp      *testutil.MockDenomCommunityPoolProvider
}

var (
	feeAddr = sdk.AccAddress("txfeecollector______")
	cpAddr  = sdk.AccAddress("txfeecollector_cp___")
)

// newEndBlockModule builds the module on a real store with mocked keepers; the fee collector is empty.
func newEndBlockModule(t *testing.T) (txfeecollector.AppModule, keeper.Keeper, sdk.Context, endBlockMocks) {
	t.Helper()

	return newEndBlockModuleWithLogger(t, log.NewNopLogger())
}

// newEndBlockModuleWithLogger is newEndBlockModule with the keeper logging to logger.
func newEndBlockModuleWithLogger(t *testing.T, logger log.Logger) (txfeecollector.AppModule, keeper.Keeper, sdk.Context, endBlockMocks) {
	t.Helper()
	ctrl := gomock.NewController(t)
	bank := testutil.NewMockBankKeeper(ctrl)
	account := testutil.NewMockAccountKeeper(ctrl)
	trade := testutil.NewMockTradeKeeper(ctrl)
	distr := testutil.NewMockDistrKeeper(ctrl)
	cp := testutil.NewMockDenomCommunityPoolProvider(ctrl)

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		logger,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		bank,
		account,
		trade,
		distr,
		cp,
		nil,
	)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))
	am := txfeecollector.NewAppModule(cdc, k, account, bank, distr)

	account.EXPECT().GetModuleAddress(types.ModuleName).Return(feeAddr).AnyTimes()
	account.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpAddr).AnyTimes()
	bank.EXPECT().GetBalancesPage(gomock.Any(), feeAddr, "", gomock.Any()).Return(nil, "").AnyTimes()

	return am, k, ctx, endBlockMocks{bank: bank, account: account, trade: trade, distr: distr, cp: cp}
}

func paidEventCount(ctx sdk.Context) int {
	n := 0
	for _, e := range ctx.EventManager().Events() {
		if e.Type == proto.MessageName(&types.CommunityPoolFeePaidEvent{}) {
			n++
		}
	}

	return n
}

// TestEndBlock_FailedSendLeavesCoinsInPlace proves a failed send to a denom's community pool discards the whole CP
// pass and logs the error: no event, no write and no cursor move reach the block, so every coin stays in
// txfeecollector_cp, nothing swapped or burned. The next block's pass reads the same page again.
func TestEndBlock_FailedSendLeavesCoinsInPlace(t *testing.T) {
	var logs bytes.Buffer
	am, k, ctx, m := newEndBlockModuleWithLogger(t, log.NewLogger(&logs, log.ColorOption(false)))
	bank, cp, trade, distr := m.bank, m.cp, m.trade, m.distr
	tokenA := sdk.NewInt64Coin("factory/bze1creator/tokena", 100)
	tokenB := sdk.NewInt64Coin("factory/bze1creator/tokenb", 200)
	ibc := sdk.NewInt64Coin("ibc/AAA", 300)
	cpA := sdk.AccAddress("token_a_cp__________")
	cpB := sdk.AccAddress("token_b_cp__________")
	params := k.GetParams(ctx)
	params.MaxBalanceIterations = 3 // a full page: a kept pass would move the cursor to ibc/AAA
	require.NoError(t, k.SetParams(ctx, params))
	page := sdk.NewCoins(tokenA, tokenB, ibc)

	// block 1: token A is sent, token B's send fails, the IBC coin is never looked at
	bank.EXPECT().GetBalancesPage(gomock.Any(), cpAddr, "", 3).Return(page, ibc.Denom).Times(2)
	trade.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).Times(2 + 3)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), tokenA.Denom).Return(cpA, true).Times(2)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), tokenB.Denom).Return(cpB, true).Times(2)
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, cpA, sdk.NewCoins(tokenA)).
		DoAndReturn(func(c context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
			// stands in for the bank writes of the successful send: they must not reach the block state
			changed := params
			changed.MaxBalanceIterations = 7
			require.NoError(t, k.SetParams(sdk.UnwrapSDKContext(c), changed))

			return nil
		}).Times(1)
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, cpB, sdk.NewCoins(tokenB)).
		Return(errors.New("send failed")).Times(1)

	// no swap, burner send or funding expectation for block 1: a call would fail the test
	require.NoError(t, am.EndBlock(ctx))
	require.Zero(t, paidEventCount(ctx), "no event of the discarded pass reaches the block")
	require.Equal(t, params, k.GetParams(ctx), "nothing the discarded pass wrote reaches the block state")
	require.Contains(t, logs.String(), "ConvertCommunityPoolFees", "the failure is logged")
	require.Contains(t, logs.String(), "send failed")

	// block 2: the cursor didn't move, so the same page is read from the first denom; the sends work now
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, cpA, sdk.NewCoins(tokenA)).Return(nil).Times(1)
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, cpB, sdk.NewCoins(tokenB)).Return(nil).Times(1)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), ibc.Denom).Return(nil, false).Times(1)
	trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibc).Return(true).Times(1)
	trade.EXPECT().SafeModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(ibc)).
		Return(sdk.NewInt64Coin("ubze", 30), nil).Times(1)
	distr.EXPECT().FundCommunityPool(gomock.Any(), sdk.NewCoins(sdk.NewInt64Coin("ubze", 30)), cpAddr).Return(nil).Times(1)

	ctx = ctx.WithEventManager(sdk.NewEventManager())
	require.NoError(t, am.EndBlock(ctx))
	require.Equal(t, 2, paidEventCount(ctx))
}

// TestEndBlock_FailedSwapKeepsOnlyThatCoin proves a failed swap in the CP pass only fails that coin: it waits in the
// collector for the next pass, while the community pool sends made in the same pass are kept.
func TestEndBlock_FailedSwapKeepsOnlyThatCoin(t *testing.T) {
	am, _, ctx, m := newEndBlockModule(t)
	bank, cp, trade := m.bank, m.cp, m.trade
	token := sdk.NewInt64Coin("factory/bze1creator/token", 100)
	ibc := sdk.NewInt64Coin("ibc/ABC", 50)
	tokenCp := sdk.AccAddress("token_cp____________")

	bank.EXPECT().GetBalancesPage(gomock.Any(), cpAddr, "", gomock.Any()).Return(sdk.NewCoins(token, ibc), ibc.Denom).Times(1)
	trade.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).Times(2)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), token.Denom).Return(tokenCp, true).Times(1)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), ibc.Denom).Return(nil, false).Times(1)
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenCp, sdk.NewCoins(token)).Return(nil).Times(1)
	trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibc).Return(true).Times(1)
	trade.EXPECT().SafeModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(ibc)).
		Return(sdk.Coin{}, errors.New("swap failed")).Times(1)

	// no funding expectation: nothing was swapped, a call would fail the test
	require.NoError(t, am.EndBlock(ctx))
	require.Equal(t, 1, paidEventCount(ctx), "the community pool send is kept")
}
