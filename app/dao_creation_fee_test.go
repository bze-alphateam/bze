package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// newDaoFeeApp builds the real app with the default x/dao params, so creating
// a DAO charges the default creation fee through the wired tradebin keeper.
func newDaoFeeApp(t *testing.T) (*app.App, sdk.Context) {
	t.Helper()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{Time: time.Unix(1_700_000_000, 0)})

	// no InitChain here: bank sends need the default params
	require.NoError(t, bApp.BankKeeper.SetParams(ctx, banktypes.DefaultParams()))
	require.NoError(t, bApp.DaoKeeper.SetParams(ctx, daotypes.DefaultParams()))

	return bApp, ctx
}

func createDaoMsg(creator sdk.AccAddress) *daotypes.MsgCreateDao {
	return &daotypes.MsgCreateDao{
		Creator: creator.String(),
		Type:    daotypes.DaoType_DAO_TYPE_STATIC_LIST,
		Name:    "builders",
		Config:  newAdminDaoConfig(),
		Members: []string{sdk.AccAddress("dao_member__________").String()},
	}
}

// Without a tx fee denom in the context, tradebin takes the creation fee in
// ubze and sends it to the burner fee collector.
func TestDaoCreationFee_CapturedThroughTradebin(t *testing.T) {
	bApp, ctx := newDaoFeeApp(t)
	creator := sdk.AccAddress("dao_fee_creator_____")
	fee := sdk.NewCoins(daotypes.DefaultDaoCreationFee)
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, fee))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, creator, fee))
	burner := authtypes.NewModuleAddress(txfeecollectortypes.BurnerFeeCollector)
	burnerBefore := bApp.BankKeeper.GetAllBalances(ctx, burner)

	msg := createDaoMsg(creator)
	_, err := bApp.MsgServiceRouter().Handler(msg)(ctx, msg)
	require.NoError(t, err)

	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, creator).IsZero())
	require.Equal(t, burnerBefore.Add(fee...), bApp.BankKeeper.GetAllBalances(ctx, burner))
	_, found := bApp.DaoKeeper.GetDao(ctx, 1)
	require.True(t, found)
}

// A creator who can't pay is refused with the wrapped bank error and no DAO.
func TestDaoCreationFee_InsufficientFundsRefused(t *testing.T) {
	bApp, ctx := newDaoFeeApp(t)
	creator := sdk.AccAddress("dao_fee_creator_____")

	msg := createDaoMsg(creator)
	_, err := bApp.MsgServiceRouter().Handler(msg)(ctx, msg)
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
	require.ErrorContains(t, err, "failed to pay the dao creation fee")

	_, found := bApp.DaoKeeper.GetDao(ctx, 1)
	require.False(t, found)
	require.Zero(t, bApp.DaoKeeper.GetDaoCounter(ctx))
}
