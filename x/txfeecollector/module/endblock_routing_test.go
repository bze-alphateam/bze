package txfeecollector_test

import (
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

	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
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
	feeAddr    = sdk.AccAddress("txfeecollector______")
	burnerAddr = sdk.AccAddress("txfeecollector_burnr")
	cpAddr     = sdk.AccAddress("txfeecollector_cp___")
)

// newEndBlockModule builds the module on a real store with mocked keepers; the fee and burner collectors are empty.
func newEndBlockModule(t *testing.T) (txfeecollector.AppModule, keeper.Keeper, sdk.Context, endBlockMocks) {
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
		log.NewNopLogger(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		bank,
		account,
		trade,
		distr,
		cp,
	)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))
	am := txfeecollector.NewAppModule(cdc, k, account, bank, distr)

	account.EXPECT().GetModuleAddress(types.ModuleName).Return(feeAddr).AnyTimes()
	account.EXPECT().GetModuleAddress(types.BurnerFeeCollector).Return(burnerAddr).AnyTimes()
	account.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpAddr).AnyTimes()
	bank.EXPECT().GetAllBalances(gomock.Any(), feeAddr).Return(sdk.NewCoins()).Times(1)
	bank.EXPECT().GetAllBalances(gomock.Any(), burnerAddr).Return(sdk.NewCoins()).Times(1)

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

// TestEndBlock_FailedConversionKeepsRoutedFees proves the routing to denom community pools commits apart from the
// community pool conversion: EndBlock discards a conversion that fails, and with it everything it wrote, so a routing
// step inside it would be rolled back every block a coin of another denom fails to swap.
func TestEndBlock_FailedConversionKeepsRoutedFees(t *testing.T) {
	am, _, ctx, m := newEndBlockModule(t)
	bank, cp, trade := m.bank, m.cp, m.trade
	token := sdk.NewInt64Coin("factory/bze1creator/token", 100)
	ibc := sdk.NewInt64Coin("ibc/ABC", 50)
	tokenCp := sdk.AccAddress("token_cp____________")

	// routing sees both coins and forwards the token
	gomock.InOrder(
		bank.EXPECT().GetAllBalances(gomock.Any(), cpAddr).Return(sdk.NewCoins(token, ibc)).Times(1),
		bank.EXPECT().GetAllBalances(gomock.Any(), cpAddr).Return(sdk.NewCoins(ibc)).Times(1),
	)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), token.Denom).Return(tokenCp, true).Times(1)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), ibc.Denom).Return(nil, false).Times(2) // routing, then the conversion after a successful routing
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenCp, sdk.NewCoins(token)).Return(nil).Times(1)

	// the conversion of the IBC coin fails
	trade.EXPECT().IsNativeDenom(gomock.Any(), ibc.Denom).Return(false).Times(1)
	trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibc).Return(true).Times(1)
	trade.EXPECT().ModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(ibc)).
		Return(sdk.Coin{}, errors.New("swap failed")).Times(1)

	require.NoError(t, am.EndBlock(ctx))
	require.Equal(t, 1, paidEventCount(ctx), "the routed coin's event reaches the block although the conversion failed")
}

// TestEndBlock_FailedSendDiscardsRouting proves a failed send discards the whole routing step, the sends made before
// it included, and leaves every coin to the conversion in the same EndBlock.
func TestEndBlock_FailedSendDiscardsRouting(t *testing.T) {
	am, k, ctx, m := newEndBlockModule(t)
	bank, cp, trade := m.bank, m.cp, m.trade
	tokenA := sdk.NewInt64Coin("factory/bze1creator/tokena", 100)
	tokenB := sdk.NewInt64Coin("factory/bze1creator/tokenb", 200)
	cpA := sdk.AccAddress("token_a_cp__________")
	cpB := sdk.AccAddress("token_b_cp__________")
	paramsBefore := k.GetParams(ctx)

	// routing and then the conversion both see the two coins
	bank.EXPECT().GetAllBalances(gomock.Any(), cpAddr).Return(sdk.NewCoins(tokenA, tokenB)).Times(2)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), tokenA.Denom).Return(cpA, true).Times(1)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), tokenB.Denom).Return(cpB, true).Times(1)
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, cpA, sdk.NewCoins(tokenA)).
		DoAndReturn(func(c context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
			// stands in for the bank writes of the successful send: they must not reach the block state
			changed := paramsBefore
			changed.MaxBalanceIterations = 7
			require.NoError(t, k.SetParams(sdk.UnwrapSDKContext(c), changed))

			return nil
		}).Times(1)
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, cpB, sdk.NewCoins(tokenB)).
		Return(errors.New("send failed")).Times(1)

	// the conversion takes both: neither can be swapped, so both go to the burner as before
	trade.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).Times(2)
	trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), gomock.Any()).Return(false).Times(2)
	trade.EXPECT().HasDeepLiquidityWithNativeDenom(gomock.Any(), gomock.Any()).Return(false).Times(2)
	bank.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.CpFeeCollector, burnermoduletypes.ModuleName, sdk.NewCoins(tokenA, tokenB)).
		Return(nil).Times(1)

	require.NoError(t, am.EndBlock(ctx))
	require.Zero(t, paidEventCount(ctx), "no event of the discarded routing reaches the block")
	require.Equal(t, paramsBefore, k.GetParams(ctx), "nothing the discarded routing wrote reaches the block state")
}
