package app_test

import (
	"testing"

	circuittypes "cosmossdk.io/x/circuit/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
	guardiantypes "github.com/bze-alphateam/bze/x/guardian/types"
)

const (
	sendURL           = "/cosmos.bank.v1beta1.MsgSend"
	circuitRefusalMsg = "circuit breaker disables execution of this message: " + sendURL
)

// newCircuitApp is the real app with DAO 1 (admin + 100ubze treasury) and the
// guardian set.
func newCircuitApp(t *testing.T) (*app.App, sdk.Context, sdk.AccAddress, sdk.AccAddress) {
	t.Helper()
	bApp, ctx, admin, treasury := newAdminDaoApp(t)
	require.NoError(t, bApp.GuardianKeeper.SetParams(ctx, guardiantypes.NewParams(guardianSigner)))
	return bApp, ctx, admin, treasury
}

func disableSend(t *testing.T, bApp *app.App, ctx sdk.Context, signer string) {
	t.Helper()
	require.NoError(t, handle(bApp, ctx, &guardiantypes.MsgDisableMessages{Signer: signer, MsgTypeUrls: []string{sendURL}}))
}

func sendFrom(treasury sdk.AccAddress, amount int64) *banktypes.MsgSend {
	return banktypes.NewMsgSend(treasury, sdk.AccAddress("circuit_recipient___"), sdk.NewCoins(sdk.NewInt64Coin("ubze", amount)))
}

// x/circuit's authority is governance; nobody else can trip, reset or grant
// circuit permissions.
func TestCircuit_AuthorityIsGovernance(t *testing.T) {
	bApp, _, _, _ := newCircuitApp(t)
	require.Equal(t, []byte(authtypes.NewModuleAddress(govtypes.ModuleName)), bApp.CircuitBreakerKeeper.GetAuthority())
}

// The guardian disables MsgSend: the router refuses it until governance resets
// the breaker with x/circuit's own message.
func TestCircuit_GuardianDisablesAndGovernanceResets(t *testing.T) {
	bApp, ctx, _, treasury := newCircuitApp(t)

	disableSend(t, bApp, ctx, guardianSigner)

	allowed, err := bApp.CircuitBreakerKeeper.IsAllowed(ctx, sendURL)
	require.NoError(t, err)
	require.False(t, allowed)

	err = handle(bApp, ctx, sendFrom(treasury, 10))
	require.ErrorContains(t, err, circuitRefusalMsg)
	require.Equal(t, int64(100), bApp.BankKeeper.GetBalance(ctx, treasury, "ubze").Amount.Int64())

	// other messages are unaffected
	_, err = runAsTx(bApp, ctx, &banktypes.MsgMultiSend{
		Inputs:  []banktypes.Input{{Address: treasury.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin("ubze", 1))}},
		Outputs: []banktypes.Output{{Address: sdk.AccAddress("circuit_recipient___").String(), Coins: sdk.NewCoins(sdk.NewInt64Coin("ubze", 1))}},
	})
	require.NoError(t, err)

	require.NoError(t, handle(bApp, ctx, &circuittypes.MsgResetCircuitBreaker{Authority: govAuthority(), MsgTypeUrls: []string{sendURL}}))
	_, err = runAsTx(bApp, ctx, sendFrom(treasury, 10))
	require.NoError(t, err)
	require.Equal(t, int64(89), bApp.BankKeeper.GetBalance(ctx, treasury, "ubze").Amount.Int64())
}

func TestCircuit_GovernanceCanDisableToo(t *testing.T) {
	bApp, ctx, _, treasury := newCircuitApp(t)

	disableSend(t, bApp, ctx, govAuthority())
	require.ErrorContains(t, handle(bApp, ctx, sendFrom(treasury, 10)), circuitRefusalMsg)
}

// The guardian holds no x/circuit permission: it cannot reset, and it cannot
// trip through x/circuit either. A stranger cannot use x/guardian.
func TestCircuit_GuardianCannotResetAndStrangerCannotDisable(t *testing.T) {
	bApp, ctx, _, _ := newCircuitApp(t)
	disableSend(t, bApp, ctx, guardianSigner)

	err := handle(bApp, ctx, &circuittypes.MsgResetCircuitBreaker{Authority: guardianSigner, MsgTypeUrls: []string{sendURL}})
	require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
	err = handle(bApp, ctx, &circuittypes.MsgTripCircuitBreaker{Authority: guardianSigner, MsgTypeUrls: []string{"/cosmos.bank.v1beta1.MsgMultiSend"}})
	require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)

	stranger := sdk.AccAddress("stranger_address_____").String()
	err = handle(bApp, ctx, &guardiantypes.MsgDisableMessages{Signer: stranger, MsgTypeUrls: []string{"/cosmos.bank.v1beta1.MsgMultiSend"}})
	require.ErrorIs(t, err, guardiantypes.ErrUnauthorized)

	allowed, err := bApp.CircuitBreakerKeeper.IsAllowed(ctx, sendURL)
	require.NoError(t, err)
	require.False(t, allowed, "still disabled")
	allowed, err = bApp.CircuitBreakerKeeper.IsAllowed(ctx, "/cosmos.bank.v1beta1.MsgMultiSend")
	require.NoError(t, err)
	require.True(t, allowed)
}

// The real router and the real disable list back the guardian's checks:
// protected, unknown and already disabled types are refused and nothing is
// written.
func TestCircuit_GuardianRefusals(t *testing.T) {
	bApp, ctx, _, _ := newCircuitApp(t)
	disableSend(t, bApp, ctx, guardianSigner)

	for _, tc := range []struct {
		urls []string
		err  error
	}{
		{[]string{"/cosmos.bank.v1beta1.MsgMultiSend", "/cosmos.circuit.v1.MsgResetCircuitBreaker"}, guardiantypes.ErrProtectedMsgType},
		{[]string{"/cosmos.bank.v1beta1.MsgMultiSend", "/bze.guardian.v1.MsgDisableMessages"}, guardiantypes.ErrProtectedMsgType},
		{[]string{"/cosmos.bank.v1beta1.MsgMultiSend", "/cosmos.gov.v1.MsgVote"}, guardiantypes.ErrProtectedMsgType},
		{[]string{"/cosmos.bank.v1beta1.MsgMultiSend", "/bze.dao.MsgVote"}, guardiantypes.ErrProtectedMsgType},
		{[]string{"/cosmos.bank.v1beta1.MsgMultiSend", "/cosmos.bank.v1beta1.MsgNope"}, guardiantypes.ErrUnknownMsgType},
		// a registered type that is not a Msg service method
		{[]string{"/cosmos.bank.v1beta1.MsgMultiSend", "/cosmos.bank.v1beta1.SendAuthorization"}, guardiantypes.ErrUnknownMsgType},
		{[]string{"/cosmos.bank.v1beta1.MsgMultiSend", sendURL}, guardiantypes.ErrAlreadyDisabled},
	} {
		err := handle(bApp, ctx, &guardiantypes.MsgDisableMessages{Signer: guardianSigner, MsgTypeUrls: tc.urls})
		require.ErrorIs(t, err, tc.err, "%v", tc.urls)

		allowed, err := bApp.CircuitBreakerKeeper.IsAllowed(ctx, "/cosmos.bank.v1beta1.MsgMultiSend")
		require.NoError(t, err)
		require.True(t, allowed, "nothing written for %v", tc.urls)
	}
}

// Messages executed by other modules go through the same router, so a
// disabled type is refused inside authz MsgExec and x/dao execution, and x/gov
// executes proposals with the app's router too.
func TestCircuit_NestedExecutionIsRefused(t *testing.T) {
	bApp, ctx, admin, treasury := newCircuitApp(t)
	disableSend(t, bApp, ctx, guardianSigner)

	exec := authz.NewMsgExec(treasury, []sdk.Msg{sendFrom(treasury, 10)})
	_, err := runAsTx(bApp, ctx, &exec)
	require.ErrorContains(t, err, circuitRefusalMsg)

	daoMsg, err := daotypes.NewMsgAdminExecute(admin.String(), 1, []sdk.Msg{sendFrom(treasury, 10)})
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, daoMsg)
	require.ErrorContains(t, err, circuitRefusalMsg)

	require.Equal(t, int64(100), bApp.BankKeeper.GetBalance(ctx, treasury, "ubze").Amount.Int64())
	require.True(t, bApp.GovKeeper.Router() == bApp.MsgServiceRouter(), "x/gov executes proposals through the app's router")

	// after the reset both paths work again
	require.NoError(t, handle(bApp, ctx, &circuittypes.MsgResetCircuitBreaker{Authority: govAuthority(), MsgTypeUrls: []string{sendURL}}))
	_, err = runAsTx(bApp, ctx, &exec)
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, daoMsg)
	require.NoError(t, err)
	require.Equal(t, int64(80), bApp.BankKeeper.GetBalance(ctx, treasury, "ubze").Amount.Int64())
}
