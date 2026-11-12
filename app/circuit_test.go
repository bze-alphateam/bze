package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	circuittypes "cosmossdk.io/x/circuit/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govmodule "github.com/cosmos/cosmos-sdk/x/gov"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

// A passed governance proposal whose message is disabled fails on execution in
// the real gov EndBlocker: the proposal ends FAILED with the breaker's error
// and its message does not run.
func TestCircuit_PassedGovProposalWithDisabledMsgFails(t *testing.T) {
	bApp, ctx, _, _ := newCircuitApp(t)
	gov := authtypes.NewModuleAddress(govtypes.ModuleName)

	// one bonded validator whose operator votes yes
	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = "ubze"
	require.NoError(t, bApp.StakingKeeper.SetParams(ctx, stakingParams))
	votingPeriod := time.Hour
	govParams := govv1.DefaultParams()
	govParams.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_000))
	govParams.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewInt64Coin("ubze", 2_000))
	govParams.VotingPeriod = &votingPeriod
	require.NoError(t, bApp.GovKeeper.Params.Set(ctx, govParams))
	require.NoError(t, bApp.DistrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool()))
	require.NoError(t, bApp.DistrKeeper.Params.Set(ctx, distrtypes.DefaultParams()))

	operator := sdk.AccAddress("circuit_validator___")
	funds := sdk.NewCoins(sdk.NewInt64Coin("ubze", 10_000_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, funds))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, operator, funds))
	bApp.AccountKeeper.SetAccount(ctx, bApp.AccountKeeper.NewAccountWithAddress(ctx, operator))

	createVal, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(operator).String(), ed25519.GenPrivKey().PubKey(), sdk.NewInt64Coin("ubze", 5_000_000),
		stakingtypes.Description{Moniker: "circuit"},
		stakingtypes.NewCommissionRates(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()), math.OneInt(),
	)
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, createVal)
	require.NoError(t, err)
	_, err = bApp.StakingKeeper.EndBlocker(ctx)
	require.NoError(t, err)

	disableSend(t, bApp, ctx, guardianSigner)

	// the gov account can pay the send: only the breaker stops it
	govFunds := sdk.NewCoins(sdk.NewInt64Coin("ubze", 10))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, govFunds))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, govtypes.ModuleName, govFunds))

	recipient := sdk.AccAddress("circuit_gov_rcpt____")
	submit, err := govv1.NewMsgSubmitProposal(
		[]sdk.Msg{banktypes.NewMsgSend(gov, recipient, sdk.NewCoins(sdk.NewInt64Coin("ubze", 1)))},
		govParams.MinDeposit, operator.String(), "", "send", "send from gov", false,
	)
	require.NoError(t, err)
	res, err := bApp.MsgServiceRouter().Handler(submit)(ctx, submit)
	require.NoError(t, err)
	var submitted govv1.MsgSubmitProposalResponse
	require.NoError(t, bApp.AppCodec().Unmarshal(res.MsgResponses[0].Value, &submitted))

	require.NoError(t, handle(bApp, ctx, govv1.NewMsgVote(operator, submitted.ProposalId, govv1.OptionYes, "")))

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(votingPeriod + time.Second))
	require.NoError(t, govmodule.EndBlocker(ctx, bApp.GovKeeper))

	proposal, err := bApp.GovKeeper.Proposals.Get(ctx, submitted.ProposalId)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusFailed, proposal.Status, "the proposal passed the vote and failed on execution")
	require.Contains(t, proposal.FailedReason, circuitRefusalMsg)
	require.True(t, bApp.BankKeeper.GetBalance(ctx, recipient, "ubze").IsZero(), "the disabled send did not run")
}
