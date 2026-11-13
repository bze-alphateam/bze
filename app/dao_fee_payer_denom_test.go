package app_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
)

// A proposal settled by another account pays its protocol fee in native, with the real keepers: the fee denom of the
// settler's tx says nothing about the token the DAO pays in. This test lives in app because the x/dao keeper suite
// mocks tradebin, which picks the denom.
func TestDaoSettleProposal_ProtocolFeeIgnoresSettlerFeeDenom(t *testing.T) {
	bApp, ctx, token, pool := transferFeeToken(t)
	admin := sdk.AccAddress("dao_admin___________")
	proposer := sdk.AccAddress("dao_proposer________")
	settler := sdk.AccAddress("dao_settler_________")

	_, err := runAsTx(bApp, ctx, &daotypes.MsgCreateDao{
		Creator: admin.String(),
		Type:    daotypes.DaoType_DAO_TYPE_STATIC_LIST,
		Name:    "spenders",
		Config:  newAdminDaoConfig(),
		Members: []string{proposer.String()},
	})
	require.NoError(t, err)
	// DAO 1 is the helper's DAO; this self-owned one is 2
	treasury := daotypes.DaoAddress(2)
	require.NoError(t, bApp.DaoKeeper.SetParams(ctx, daotypes.DefaultParams()))

	// the treasury can pay either way
	fee := daotypes.DefaultDaoCreationFee
	nativeFunds := sdk.NewCoins(fee.AddAmount(fee.Amount))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, nativeFunds))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, treasury, nativeFunds))
	fundTokenPayer(t, bApp, ctx, pool, fee, treasury)
	tokensBefore := bApp.BankKeeper.GetBalance(ctx, treasury, token)
	require.True(t, tokensBefore.IsPositive())

	submitMsg, err := daotypes.NewMsgSubmitProposal(proposer.String(), 2, "create", "", []sdk.Msg{createDaoMsg(treasury)}, false)
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, submitMsg)
	require.NoError(t, err)
	proposalId := bApp.DaoKeeper.GetProposalCounter(ctx)
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgVote(proposer.String(), proposalId, daotypes.VoteOption_YES))
	require.NoError(t, err)
	proposal, _ := bApp.DaoKeeper.GetProposal(ctx, proposalId)
	ctx = ctx.WithBlockTime(time.Unix(proposal.VotingEnd, 0))

	settleCtx := ctx.WithValue(tradebinkeeper.CtxFeeDenomKey, token).WithValue(tradebinkeeper.CtxFeePayerKey, settler)
	_, err = runAsTx(bApp, settleCtx, daotypes.NewMsgSettleProposal(settler.String(), proposalId))
	require.NoError(t, err)

	proposal, _ = bApp.DaoKeeper.GetProposal(ctx, proposalId)
	require.Equal(t, daotypes.ProposalStatus_EXECUTED, proposal.Status)
	require.Equal(t, tokensBefore, bApp.BankKeeper.GetBalance(ctx, treasury, token), "the treasury's token is untouched")
	require.Equal(t, fee, bApp.BankKeeper.GetBalance(ctx, treasury, fee.Denom), "the treasury paid the native fee")
}

const daoAnteChainId = "dao-ante-test"

// daoAnteKey is a signing key with an account in the auth keeper.
type daoAnteKey struct {
	priv   cryptotypes.PrivKey
	addr   sdk.AccAddress
	number uint64
}

func newDaoAnteKey(bApp *app.App, ctx sdk.Context, secret string) daoAnteKey {
	priv := secp256k1.GenPrivKeyFromSecret([]byte(secret))
	addr := sdk.AccAddress(priv.PubKey().Address())
	acc := bApp.AccountKeeper.NewAccountWithAddress(ctx, addr)
	bApp.AccountKeeper.SetAccount(ctx, acc)

	return daoAnteKey{priv: priv, addr: addr, number: acc.GetAccountNumber()}
}

// daoAnteDeliver signs msgs with keys (the first one pays the fee), runs the tx through the app's ante handler in
// deliver mode, then its messages through the router on the context the ante handler returned. State is written only
// when everything succeeded, as for a delivered tx.
func daoAnteDeliver(t *testing.T, bApp *app.App, ctx sdk.Context, fee sdk.Coins, msgs []sdk.Msg, keys ...daoAnteKey) error {
	t.Helper()
	numbers := make([]uint64, len(keys))
	seqs := make([]uint64, len(keys))
	privs := make([]cryptotypes.PrivKey, len(keys))
	for i, key := range keys {
		acc := bApp.AccountKeeper.GetAccount(ctx, key.addr)
		numbers[i], seqs[i], privs[i] = key.number, acc.GetSequence(), key.priv
	}
	tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), bApp.TxConfig(), msgs, fee, 50_000_000, daoAnteChainId, numbers, seqs, privs...)
	require.NoError(t, err)

	txCtx, write := ctx.CacheContext()
	txCtx, err = bApp.AnteHandler()(txCtx, tx, false)
	if err != nil {
		return err
	}
	for _, msg := range tx.GetMsgs() {
		if _, err := bApp.MsgServiceRouter().Handler(msg)(txCtx, msg); err != nil {
			return err
		}
	}
	write()

	return nil
}

// daoAnteSetup builds the app with a factory token that pays fees, and DAO 2 administered by a signing key, with a
// treasury that can pay the DAO creation fee in native or in the token.
func daoAnteSetup(t *testing.T) (*app.App, sdk.Context, daoAnteKey, string, tradebintypes.LiquidityPool) {
	t.Helper()
	bApp, ctx, token, pool := transferFeeToken(t)
	ctx = ctx.WithBlockHeight(10).WithChainID(daoAnteChainId)
	require.NoError(t, bApp.AccountKeeper.Params.Set(ctx, authtypes.DefaultParams()))

	admin := newDaoAnteKey(bApp, ctx, "dao_ante_admin")
	_, err := runAsTx(bApp, ctx, &daotypes.MsgCreateDao{
		Creator: admin.addr.String(),
		Admin:   admin.addr.String(),
		Type:    daotypes.DaoType_DAO_TYPE_STATIC_LIST,
		Name:    "fee payers",
		Config:  newAdminDaoConfig(),
		Members: []string{admin.addr.String()},
	})
	require.NoError(t, err)
	require.NoError(t, bApp.DaoKeeper.SetParams(ctx, daotypes.DefaultParams()))

	treasury := daotypes.DaoAddress(2)
	fee := daotypes.DefaultDaoCreationFee
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, sdk.NewCoins(fee)))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, treasury, sdk.NewCoins(fee)))
	fundTokenPayer(t, bApp, ctx, pool, fee, treasury)

	return bApp, ctx, admin, token, pool
}

// A signed tx paying its fee in a factory token, through the app's real ante handler: the DAO creation fee its
// admin-execute charges to the DAO is paid in native, the DAO's token balance is unchanged, and the fee payer paid
// exactly the tx fee in the token. A second signer that is not the fee payer changes nothing.
func TestDaoAdminExecute_AnteFeeDenomOnlyForTxFeePayer(t *testing.T) {
	for _, tc := range []struct {
		desc      string
		twoSigner bool
	}{
		{desc: "the admin pays the tx fee"},
		{desc: "another signer pays the tx fee", twoSigner: true},
	} {
		bApp, ctx, admin, token, pool := daoAnteSetup(t)
		treasury := daotypes.DaoAddress(2)
		fee := daotypes.DefaultDaoCreationFee

		execute, err := daotypes.NewMsgAdminExecute(admin.addr.String(), 2, []sdk.Msg{createDaoMsg(treasury)})
		require.NoError(t, err)
		msgs := []sdk.Msg{execute}
		keys := []daoAnteKey{admin}
		payer := admin
		if tc.twoSigner {
			payer = newDaoAnteKey(bApp, ctx, "dao_ante_payer")
			msgs = []sdk.Msg{banktypes.NewMsgSend(payer.addr, admin.addr, sdk.NewCoins(sdk.NewInt64Coin(token, 1))), execute}
			keys = []daoAnteKey{payer, admin}
		}
		fundTokenPayer(t, bApp, ctx, pool, fee, payer.addr)
		txFee := sdk.NewCoins(sdk.NewInt64Coin(token, 1_000))
		payerBefore := bApp.BankKeeper.GetBalance(ctx, payer.addr, token)
		treasuryTokens := bApp.BankKeeper.GetBalance(ctx, treasury, token)

		require.NoError(t, daoAnteDeliver(t, bApp, ctx, txFee, msgs, keys...), tc.desc)

		_, found := bApp.DaoKeeper.GetDao(ctx, 3)
		require.True(t, found, tc.desc)
		require.Equal(t, treasuryTokens, bApp.BankKeeper.GetBalance(ctx, treasury, token), tc.desc)
		require.True(t, bApp.BankKeeper.GetBalance(ctx, treasury, fee.Denom).IsZero(), "%s: the treasury paid the native fee", tc.desc)
		spent := txFee.AmountOf(token)
		if tc.twoSigner {
			spent = spent.AddRaw(1)
		}
		require.Equal(t, payerBefore.Amount.Sub(spent), bApp.BankKeeper.GetBalance(ctx, payer.addr, token).Amount, tc.desc)
	}
}

// The control of the case above: through the same ante handler, a protocol fee charged to the tx fee payer itself is
// paid in the token the tx fee is paid in.
func TestDaoCreateDao_AnteFeeDenomAppliesToTxFeePayer(t *testing.T) {
	bApp, ctx, admin, token, pool := daoAnteSetup(t)
	fee := daotypes.DefaultDaoCreationFee
	fundTokenPayer(t, bApp, ctx, pool, fee, admin.addr)
	nativeBefore := bApp.BankKeeper.GetBalance(ctx, admin.addr, fee.Denom)
	tokensBefore := bApp.BankKeeper.GetBalance(ctx, admin.addr, token)
	txFee := sdk.NewCoins(sdk.NewInt64Coin(token, 1_000))

	require.NoError(t, daoAnteDeliver(t, bApp, ctx, txFee, []sdk.Msg{createDaoMsg(admin.addr)}, admin))

	_, found := bApp.DaoKeeper.GetDao(ctx, 3)
	require.True(t, found)
	require.Equal(t, nativeBefore, bApp.BankKeeper.GetBalance(ctx, admin.addr, fee.Denom))
	require.True(t, bApp.BankKeeper.GetBalance(ctx, admin.addr, token).Amount.LT(tokensBefore.Amount.Sub(txFee.AmountOf(token))),
		"the creation fee was paid in the token")
}
