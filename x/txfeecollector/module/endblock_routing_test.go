package txfeecollector_test

import (
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

// TestEndBlock_FailedConversionKeepsRoutedFees proves the routing to denom community pools commits apart from the
// community pool conversion: EndBlock discards a conversion that fails, and with it everything it wrote, so a routing
// step inside it would be rolled back every block a coin of another denom fails to swap.
func TestEndBlock_FailedConversionKeepsRoutedFees(t *testing.T) {
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

	feeAddr := sdk.AccAddress("txfeecollector______")
	burnerAddr := sdk.AccAddress("txfeecollector_burnr")
	cpAddr := sdk.AccAddress("txfeecollector_cp___")
	token := sdk.NewInt64Coin("factory/bze1creator/token", 100)
	ibc := sdk.NewInt64Coin("ibc/ABC", 50)
	tokenCp := sdk.AccAddress("token_cp____________")

	// the other two collectors are empty
	account.EXPECT().GetModuleAddress(types.ModuleName).Return(feeAddr).AnyTimes()
	account.EXPECT().GetModuleAddress(types.BurnerFeeCollector).Return(burnerAddr).AnyTimes()
	account.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpAddr).AnyTimes()
	bank.EXPECT().GetAllBalances(gomock.Any(), feeAddr).Return(sdk.NewCoins()).Times(1)
	bank.EXPECT().GetAllBalances(gomock.Any(), burnerAddr).Return(sdk.NewCoins()).Times(1)

	// routing sees both coins and forwards the token
	gomock.InOrder(
		bank.EXPECT().GetAllBalances(gomock.Any(), cpAddr).Return(sdk.NewCoins(token, ibc)).Times(1),
		bank.EXPECT().GetAllBalances(gomock.Any(), cpAddr).Return(sdk.NewCoins(ibc)).Times(1),
	)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), token.Denom).Return(tokenCp, true).Times(1)
	cp.EXPECT().GetDenomCommunityPool(gomock.Any(), ibc.Denom).Return(nil, false).Times(1)
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenCp, sdk.NewCoins(token)).Return(nil).Times(1)

	// the conversion of the IBC coin fails
	trade.EXPECT().IsNativeDenom(gomock.Any(), ibc.Denom).Return(false).Times(1)
	trade.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibc).Return(true).Times(1)
	trade.EXPECT().ModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(ibc)).
		Return(sdk.Coin{}, errors.New("swap failed")).Times(1)

	require.NoError(t, am.EndBlock(ctx))

	var paid []types.CommunityPoolFeePaidEvent
	for _, e := range ctx.EventManager().Events() {
		if e.Type == proto.MessageName(&types.CommunityPoolFeePaidEvent{}) {
			paid = append(paid, types.CommunityPoolFeePaidEvent{})
		}
	}
	require.Len(t, paid, 1, "the routed coin's event reaches the block although the conversion failed")
}
