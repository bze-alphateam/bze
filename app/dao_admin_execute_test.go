package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
)

// newAdminDaoApp builds the real app and creates DAO 1 with an external admin
// and a treasury of 100ubze.
func newAdminDaoApp(t *testing.T) (*app.App, sdk.Context, sdk.AccAddress, sdk.AccAddress) {
	t.Helper()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	bApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)
	ctx := bApp.NewUncachedContext(false, cmtproto.Header{Time: time.Unix(1_700_000_000, 0)})

	// no InitChain here: bank sends need the default params
	require.NoError(t, bApp.BankKeeper.SetParams(ctx, banktypes.DefaultParams()))
	params := daotypes.DefaultParams()
	params.DaoCreationFee = sdk.NewInt64Coin("ubze", 0)
	require.NoError(t, bApp.DaoKeeper.SetParams(ctx, params))

	admin := sdk.AccAddress("dao_admin___________")
	member := sdk.AccAddress("dao_member__________")
	res, err := bApp.MsgServiceRouter().Handler(&daotypes.MsgCreateDao{})(ctx, &daotypes.MsgCreateDao{
		Creator: admin.String(),
		Admin:   admin.String(),
		Type:    daotypes.DaoType_DAO_TYPE_STATIC_LIST,
		Name:    "builders",
		Config: daotypes.DaoConfig{
			MinQuorum:         math.LegacyMustNewDecFromStr("0.4"),
			Threshold:         math.LegacyMustNewDecFromStr("0.5"),
			VotingPeriod:      72 * time.Hour,
			ExpeditedPeriod:   24 * time.Hour,
			ExpediteQuorum:    math.LegacyMustNewDecFromStr("0.4"),
			ExpediteThreshold: math.LegacyMustNewDecFromStr("0.5"),
		},
		Members: []string{member.String()},
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	dao, found := bApp.DaoKeeper.GetDao(ctx, 1)
	require.True(t, found)
	treasury := sdk.MustAccAddressFromBech32(dao.Address)
	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 100))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, treasury, coins))

	return bApp, ctx, admin, treasury
}

// runAsTx runs msg through the app's router on a cached context that is written
// only on success, the way baseapp runs a transaction's messages.
func runAsTx(bApp *app.App, ctx sdk.Context, msg sdk.Msg) (sdk.Context, error) {
	cacheCtx, write := ctx.CacheContext()
	_, err := bApp.MsgServiceRouter().Handler(msg)(cacheCtx, msg)
	if err == nil {
		write()
	}

	return cacheCtx, err
}

// D10 through the real wiring: depinject hands x/dao the app's router, and the
// admin moves treasury funds with the real bank keeper.
func TestDaoAdminExecute_SpendsTreasury(t *testing.T) {
	bApp, ctx, admin, treasury := newAdminDaoApp(t)
	recipient := sdk.AccAddress("dao_recipient_______")

	msg, err := daotypes.NewMsgAdminExecute(admin.String(), 1, []sdk.Msg{
		banktypes.NewMsgSend(treasury, recipient, sdk.NewCoins(sdk.NewInt64Coin("ubze", 40))),
	})
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, msg)
	require.NoError(t, err)

	require.Equal(t, int64(60), bApp.BankKeeper.GetBalance(ctx, treasury, "ubze").Amount.Int64())
	require.Equal(t, int64(40), bApp.BankKeeper.GetBalance(ctx, recipient, "ubze").Amount.Int64())
}

// Atomicity: the second msg fails, so the error is returned and the first
// msg's transfer, which did run, is dropped with the transaction.
func TestDaoAdminExecute_FailureRevertsEarlierMsgs(t *testing.T) {
	bApp, ctx, admin, treasury := newAdminDaoApp(t)
	recipient := sdk.AccAddress("dao_recipient_______")

	msg, err := daotypes.NewMsgAdminExecute(admin.String(), 1, []sdk.Msg{
		banktypes.NewMsgSend(treasury, recipient, sdk.NewCoins(sdk.NewInt64Coin("ubze", 40))),
		banktypes.NewMsgSend(treasury, recipient, sdk.NewCoins(sdk.NewInt64Coin("ubze", 61))),
	})
	require.NoError(t, err)
	txCtx, err := runAsTx(bApp, ctx, msg)
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)

	// the first send ran inside the transaction...
	require.Equal(t, int64(40), bApp.BankKeeper.GetBalance(txCtx, recipient, "ubze").Amount.Int64())
	// ...and nothing of it is left once the transaction failed
	require.Equal(t, int64(100), bApp.BankKeeper.GetBalance(ctx, treasury, "ubze").Amount.Int64())
	require.True(t, bApp.BankKeeper.GetBalance(ctx, recipient, "ubze").IsZero())
}
