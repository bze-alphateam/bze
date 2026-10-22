package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// newTransferFeeApp returns an app with a factory token charging a 1% transfer fee, held by user.
func newTransferFeeApp(t *testing.T, withCp bool) (*app.App, sdk.Context, string, sdk.AccAddress) {
	t.Helper()

	bApp, ctx, admin, treasury := newAdminDaoApp(t)
	require.NoError(t, bApp.TxfeecollectorKeeper.SetParams(ctx, txfeecollectortypes.DefaultParams()))

	denom := "factory/" + admin.String() + "/fee"
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomAuthority(ctx, denom, tokenfactorytypes.DenomAuthority{Admin: admin.String(), Mintable: true}))
	cfg := tokenfactorytypes.DenomConfig{TransferFee: math.LegacyMustNewDecFromStr("0.01"), GasSurcharge: math.LegacyZeroDec()}
	if withCp {
		cfg.CommunityPool = treasury.String()
	}
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomConfig(ctx, denom, cfg))

	user := sdk.AccAddress("fee_user____________")
	coins := sdk.NewCoins(sdk.NewInt64Coin(denom, 10_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, tokenfactorytypes.ModuleName, user, coins))

	return bApp, ctx, denom, user
}

func moduleBalance(bApp *app.App, ctx sdk.Context, module, denom string) math.Int {
	return bApp.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(module), denom).Amount
}

// The app installs the transfer fee as x/bank's send restriction: a user send pays the fee on top, into the
// community pool collector.
func TestTransferFee_UserSendPaysOnTop(t *testing.T) {
	bApp, ctx, denom, user := newTransferFeeApp(t, true)
	recipient := sdk.AccAddress("fee_recipient_______")

	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, user, recipient, sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000))))

	require.Equal(t, math.NewInt(1_000), bApp.BankKeeper.GetBalance(ctx, recipient, denom).Amount)
	require.Equal(t, math.NewInt(8_990), bApp.BankKeeper.GetBalance(ctx, user, denom).Amount)
	require.Equal(t, math.NewInt(10), moduleBalance(bApp, ctx, txfeecollectortypes.CpFeeCollector, denom))
}

// A token without a community pool has its fee sent to the burner collector.
func TestTransferFee_NoCommunityPoolGoesToBurner(t *testing.T) {
	bApp, ctx, denom, user := newTransferFeeApp(t, false)

	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, user, sdk.AccAddress("fee_recipient_______"), sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000))))

	require.Equal(t, math.NewInt(10), moduleBalance(bApp, ctx, txfeecollectortypes.BurnerFeeCollector, denom))
	require.True(t, moduleBalance(bApp, ctx, txfeecollectortypes.CpFeeCollector, denom).IsZero())
}

// Module sends never pay, in either direction of a user flow (a user → module send pays once).
func TestTransferFee_ModuleSendsPayNothing(t *testing.T) {
	bApp, ctx, denom, user := newTransferFeeApp(t, true)

	require.NoError(t, bApp.BankKeeper.SendCoinsFromAccountToModule(ctx, user, tokenfactorytypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000))))
	require.Equal(t, math.NewInt(8_990), bApp.BankKeeper.GetBalance(ctx, user, denom).Amount)

	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, tokenfactorytypes.ModuleName, user, sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000))))
	require.Equal(t, math.NewInt(9_990), bApp.BankKeeper.GetBalance(ctx, user, denom).Amount)
	require.Equal(t, math.NewInt(10), moduleBalance(bApp, ctx, txfeecollectortypes.CpFeeCollector, denom))
}

// An ICS-20 escrow (a plain account in ibc-go v10) never pays: the app's escrow checker recognises it from the
// channels on the transfer port.
func TestTransferFee_EscrowSendsPayNothing(t *testing.T) {
	bApp, ctx, denom, user := newTransferFeeApp(t, true)
	bApp.IBCKeeper.ChannelKeeper.SetChannel(ctx, ibctransfertypes.PortID, "channel-7", channeltypes.Channel{
		State:          channeltypes.OPEN,
		Ordering:       channeltypes.UNORDERED,
		Counterparty:   channeltypes.NewCounterparty(ibctransfertypes.PortID, "channel-0"),
		ConnectionHops: []string{"connection-0"},
		Version:        ibctransfertypes.V1,
	})
	escrow := ibctransfertypes.GetEscrowAddress(ibctransfertypes.PortID, "channel-7")

	// the user's send into the escrow pays, as an IBC transfer out does
	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, user, escrow, sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000))))
	require.Equal(t, math.NewInt(8_990), bApp.BankKeeper.GetBalance(ctx, user, denom).Amount)

	// the refund or return transfer out of the escrow doesn't
	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, escrow, user, sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000))))
	require.Equal(t, math.NewInt(9_990), bApp.BankKeeper.GetBalance(ctx, user, denom).Amount)
	require.True(t, bApp.BankKeeper.GetBalance(ctx, escrow, denom).Amount.IsZero())
	require.Equal(t, math.NewInt(10), moduleBalance(bApp, ctx, txfeecollectortypes.CpFeeCollector, denom))
}

// The user can't send more than the balance minus the fee: the send is refused before anything is charged.
func TestTransferFee_InsufficientForFee(t *testing.T) {
	bApp, ctx, denom, user := newTransferFeeApp(t, true)

	err := bApp.BankKeeper.SendCoins(ctx, user, sdk.AccAddress("fee_recipient_______"), sdk.NewCoins(sdk.NewInt64Coin(denom, 10_000)))
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
	require.ErrorContains(t, err, "transfer fee")
	require.Equal(t, math.NewInt(10_000), bApp.BankKeeper.GetBalance(ctx, user, denom).Amount)
}

// A multisend pays the fee of every output. When each output fits with its fee but the total doesn't, x/bank's own
// debit refuses the send (its generic insufficient funds error): the tx reverts, so nothing is charged.
func TestTransferFee_Multisend(t *testing.T) {
	bApp, ctx, denom, user := newTransferFeeApp(t, true)
	r1 := sdk.AccAddress("fee_recipient_1_____")
	r2 := sdk.AccAddress("fee_recipient_2_____")

	out := func(addr sdk.AccAddress, amount int64) banktypes.Output {
		return banktypes.NewOutput(addr, sdk.NewCoins(sdk.NewInt64Coin(denom, amount)))
	}
	in := func(amount int64) banktypes.Input {
		return banktypes.NewInput(user, sdk.NewCoins(sdk.NewInt64Coin(denom, amount)))
	}

	require.NoError(t, bApp.BankKeeper.InputOutputCoins(ctx, in(3_000), []banktypes.Output{out(r1, 1_000), out(r2, 2_000)}))
	require.Equal(t, math.NewInt(1_000), bApp.BankKeeper.GetBalance(ctx, r1, denom).Amount)
	require.Equal(t, math.NewInt(2_000), bApp.BankKeeper.GetBalance(ctx, r2, denom).Amount)
	require.Equal(t, math.NewInt(6_970), bApp.BankKeeper.GetBalance(ctx, user, denom).Amount)
	require.Equal(t, math.NewInt(30), moduleBalance(bApp, ctx, txfeecollectortypes.CpFeeCollector, denom))

	// 6,970 left: each 3,460 output fits with its 35 fee, the total of 6,920 + 70 doesn't
	cacheCtx, _ := ctx.CacheContext()
	err := bApp.BankKeeper.InputOutputCoins(cacheCtx, in(6_920), []banktypes.Output{out(r1, 3_460), out(r2, 3_460)})
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
	require.Equal(t, math.NewInt(6_970), bApp.BankKeeper.GetBalance(ctx, user, denom).Amount)
}

// Tokens without rates behave exactly as before.
func TestTransferFee_TokenWithoutRatesUnchanged(t *testing.T) {
	bApp, ctx, denom, user := newTransferFeeApp(t, true)
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomConfig(ctx, denom, tokenfactorytypes.DefaultDenomConfig()))

	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, user, sdk.AccAddress("fee_recipient_______"), sdk.NewCoins(sdk.NewInt64Coin(denom, 10_000))))
	require.True(t, bApp.BankKeeper.GetBalance(ctx, user, denom).Amount.IsZero())
	require.True(t, moduleBalance(bApp, ctx, txfeecollectortypes.CpFeeCollector, denom).IsZero())
}
