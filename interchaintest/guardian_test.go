package interchaintest_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	gdnFunds     = 1_000_000_000_000 // ubze for the lp (pool, gov deposits) and fees of everyone
	gdnUserFunds = 10_000_000_000
	gdnMint      = 1_000_000_000_000
	gdnUserUsd   = 1_000_000_000
	gdnPoolUsd   = 100_000_000_000
	gdnPoolBze   = 100_000_000_000
	gdnSwapIn    = 1_000_000
	gdnAddUsd    = 1_000_000
	gdnAddBze    = 1_000_000
	gdnGas       = 500_000

	gdnPoolFee = "0.002"
	gdnFeeDest = `{"treasury":"0","burner":"0.5","providers":"0.5"}`

	gdnSwapURL   = "/bze.tradebin.MsgMultiSwap"
	gdnResetURL  = "/cosmos.circuit.v1.MsgResetCircuitBreaker"
	gdnCircuitRe = "circuit breaker disables execution of this message: " + gdnSwapURL

	gdnCodespace       = "guardian"
	gdnUnauthorizedErr = 1101
	gdnProtectedErr    = 1111
)

// TestGuardian proves the guardian's emergency switches through the real
// binary. Message circuit breaker: the guardian disables MsgMultiSwap, a
// user's swap is refused and so is a passed governance proposal carrying it,
// while the other tradebin messages still work; a stranger cannot disable
// anything, the guardian cannot disable governance, circuit or guardian
// messages, and a governance proposal with x/circuit's MsgResetCircuitBreaker
// re-enables the swap. The state carries over between
// the subtests, which run in order.
func TestGuardian(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()

	guardianMnemonic, guardianAddr := newAccount(t, chain.Bech32Prefix)
	// the guardian is a test wallet set in genesis, so the suite does not
	// depend on the address the upgrade handler sets on mainnet
	net := chain.Start(t, ctx, chain.WithGenesis(cosmos.NewGenesisKV("app_state.guardian.params.guardian", guardianAddr)))
	s := setupGuardian(t, ctx, net.Chain, guardianMnemonic, guardianAddr)

	t.Run("1 a swap works before the breaker", s.swapWorks)
	t.Run("2 guardian disables MsgMultiSwap", s.disableSwap)
	t.Run("3 a swap is refused", s.swapRefused)
	t.Run("4 a passed proposal with a disabled message fails", s.govProposalFails)
	t.Run("5 other tradebin messages still work", s.otherTradebinMsgs)
	t.Run("6 a stranger cannot disable messages", s.strangerRefused)
	t.Run("7 protected messages cannot be disabled", s.protectedRefused)
	t.Run("8 governance resets the breaker", s.govResets)
}

type guardianSuite struct {
	ctx context.Context
	bze *cosmos.CosmosChain
	gov string

	guardian, lp, trader, stranger ibc.Wallet
	usd, pool                      string
}

func setupGuardian(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain, guardianMnemonic, guardianAddr string) *guardianSuite {
	t.Helper()
	s := &guardianSuite{ctx: ctx, bze: bze}

	var err error
	s.guardian, err = interchaintest.GetAndFundTestUserWithMnemonic(ctx, "guardian", guardianMnemonic, sdkmath.NewInt(gdnUserFunds), bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, s.guardian.FormattedAddress())
	s.lp = interchaintest.GetAndFundTestUsers(t, ctx, "gdn-lp", sdkmath.NewInt(gdnFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "gdn", sdkmath.NewInt(gdnUserFunds), bze, bze)
	s.trader, s.stranger = users[0], users[1]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	guardian, err := helpers.QueryGuardian(ctx, bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, guardian, "the genesis guardian")
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)

	// a factory denom and a usd/ubze pool; pool ids order the denoms
	// alphabetically: "factory/…" sorts before "ubze"
	s.usd = "factory/" + s.lp.FormattedAddress() + "/usd"
	s.pool = s.usd + "_" + chain.Denom
	s.mustTx(t, s.lp, "tokenfactory", "create-denom", "usd")
	s.mustTx(t, s.lp, "tokenfactory", "mint", coinStr(gdnMint, s.usd))
	s.mustTx(t, s.lp, "bank", "send", s.lp.FormattedAddress(), s.trader.FormattedAddress(), coinStr(gdnUserUsd, s.usd))
	s.mustTx(t, s.lp, "tradebin", "create-liquidity-pool", s.usd, chain.Denom, gdnPoolFee, gdnFeeDest, "false",
		sdkmath.NewInt(gdnPoolUsd).String(), sdkmath.NewInt(gdnPoolBze).String())

	return s
}

// 1. Baseline: the trader's swap goes through.
func (s *guardianSuite) swapWorks(t *testing.T) {
	res := s.mustMsgs(t, s.trader, s.swapMsg())
	require.NotEmpty(t, helpers.TxEvents(res, "bze.tradebin.SwapEvent"), "the swap ran")
}

// 2. The guardian adds MsgMultiSwap to the x/circuit disable list.
func (s *guardianSuite) disableSwap(t *testing.T) {
	require.Empty(t, s.disabledList(t))

	res := s.mustTx(t, s.guardian, "guardian", "disable-messages", gdnSwapURL)
	events := helpers.EventsWith(helpers.TxEvents(res, "bze.guardian.v1.MessagesDisabledEvent"), map[string]string{
		"signer": s.guardian.FormattedAddress(),
	})
	require.Len(t, events, 1, "one MessagesDisabledEvent from the guardian")
	require.Contains(t, events[0].Attributes["msg_type_urls"], gdnSwapURL)

	require.Equal(t, []string{gdnSwapURL}, s.disabledList(t))
}

// 3. The trader's swap is included and fails in the message router: the pool
// and the trader's balances are unchanged but for the tx fee.
func (s *guardianSuite) swapRefused(t *testing.T) {
	pool := s.poolState(t)
	ubze, usd := s.balance(t, s.trader, chain.Denom), s.balance(t, s.trader, s.usd)

	res, err := helpers.ExecMsgs(s.ctx, s.bze, s.trader, []helpers.TxOption{helpers.WithGas(gdnGas)}, s.swapMsg())
	reason, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, reason, gdnCircuitRe)
	require.Empty(t, helpers.TxEvents(res, "bze.tradebin.SwapEvent"), "no swap ran")

	require.Equal(t, pool, s.poolState(t), "the pool did not move")
	fee := sdkmath.ZeroInt()
	if res.Height > 0 {
		fee = s.txFee(t, res)
	}
	require.Equal(t, ubze.Sub(fee), s.balance(t, s.trader, chain.Denom), "the trader only paid the fee")
	require.Equal(t, usd, s.balance(t, s.trader, s.usd), "the trader kept the usd")
}

// 4. Governance executes through the same router: a proposal carrying a
// MsgMultiSwap passes the vote and fails on execution with the breaker's
// error, and the swap does not run.
func (s *guardianSuite) govProposalFails(t *testing.T) {
	pool := s.poolState(t)
	govUsd := s.balanceOf(t, s.gov, s.usd)

	id, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.lp, helpers.Proposal{
		Messages: []any{s.swapMsgFrom(s.gov)},
		Title:    "swap from governance",
		Summary:  "e2e: a disabled message in a passed proposal",
	})
	require.Error(t, err, "the proposal does not pass")
	require.NotZero(t, id, "the proposal was submitted")

	res, qerr := helpers.Query[struct {
		Proposal struct {
			Status       string `json:"status"`
			FailedReason string `json:"failed_reason"`
		} `json:"proposal"`
	}](s.ctx, s.bze, "gov", "proposal", strconv.FormatUint(id, 10))
	require.NoError(t, qerr)
	require.Equal(t, "PROPOSAL_STATUS_FAILED", res.Proposal.Status, "passed the vote, failed on execution: %v", err)
	require.Contains(t, res.Proposal.FailedReason, gdnCircuitRe)

	require.Equal(t, pool, s.poolState(t), "the pool did not move")
	require.Equal(t, govUsd, s.balanceOf(t, s.gov, s.usd), "the gov account swapped nothing")
}

// 5. Only the disabled type is refused: the trader adds liquidity to the same
// pool.
func (s *guardianSuite) otherTradebinMsgs(t *testing.T) {
	res := s.mustTx(t, s.trader, "tradebin", "add-liquidity", s.pool,
		sdkmath.NewInt(gdnAddUsd).String(), sdkmath.NewInt(gdnAddBze).String(), "1")
	require.NotEmpty(t, helpers.TxEvents(res, "bze.tradebin.LiquidityAddedEvent"), "liquidity was added")
}

// 6. An account that is neither the guardian nor governance is refused and
// the disable list does not change.
func (s *guardianSuite) strangerRefused(t *testing.T) {
	res := s.refusedMsg(t, s.stranger, s.disableMsg(s.stranger, "/cosmos.bank.v1beta1.MsgSend"))
	require.Equal(t, gdnCodespace, res.Codespace, res.RawLog)
	require.EqualValues(t, gdnUnauthorizedErr, res.Code, res.RawLog)
	require.Equal(t, []string{gdnSwapURL}, s.disabledList(t))
}

// 7. The guardian cannot disable governance, the circuit breaker's reset or
// its own messages: that would lock governance or itself out.
func (s *guardianSuite) protectedRefused(t *testing.T) {
	for _, url := range []string{
		"/cosmos.gov.v1.MsgVote",
		"/cosmos.gov.v1.MsgSubmitProposal",
		gdnResetURL,
		"/bze.guardian.v1.MsgDisableMessages",
	} {
		t.Run(url, func(t *testing.T) {
			// a valid url first: nothing is written for it either
			res := s.refusedMsg(t, s.guardian, s.disableMsg(s.guardian, "/cosmos.bank.v1beta1.MsgSend", url))
			require.Equal(t, gdnCodespace, res.Codespace, res.RawLog)
			require.EqualValues(t, gdnProtectedErr, res.Code, res.RawLog)
			require.Equal(t, []string{gdnSwapURL}, s.disabledList(t))
		})
	}
}

// 8. A governance proposal with x/circuit's MsgResetCircuitBreaker takes the
// type off the list and the swap works again.
func (s *guardianSuite) govResets(t *testing.T) {
	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.lp, helpers.Proposal{
		Messages: []any{map[string]any{
			"@type":         gdnResetURL,
			"authority":     s.gov,
			"msg_type_urls": []string{gdnSwapURL},
		}},
		Title:   "re-enable swaps",
		Summary: "e2e: reset the circuit breaker for MsgMultiSwap",
	})
	require.NoError(t, err)
	require.Empty(t, s.disabledList(t))

	res := s.mustMsgs(t, s.trader, s.swapMsg())
	require.NotEmpty(t, helpers.TxEvents(res, "bze.tradebin.SwapEvent"), "the swap ran again")
}

// ---------------------------------------------------------------------------
// messages, transactions and queries

func (s *guardianSuite) swapMsg() map[string]any {
	return s.swapMsgFrom(s.trader.FormattedAddress())
}

func (s *guardianSuite) swapMsgFrom(creator string) map[string]any {
	return map[string]any{
		"@type":      gdnSwapURL,
		"creator":    creator,
		"routes":     []string{s.pool},
		"input":      map[string]string{"denom": s.usd, "amount": sdkmath.NewInt(gdnSwapIn).String()},
		"min_output": map[string]string{"denom": chain.Denom, "amount": "1"},
	}
}

func (s *guardianSuite) disableMsg(signer ibc.Wallet, urls ...string) map[string]any {
	return map[string]any{
		"@type":         "/bze.guardian.v1.MsgDisableMessages",
		"signer":        signer.FormattedAddress(),
		"msg_type_urls": urls,
	}
}

// mustTx sends a CLI tx with a fixed gas limit and requires success.
func (s *guardianSuite) mustTx(t *testing.T, user ibc.Wallet, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(gdnGas)}, args...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", args, res.RawLog)
	return res
}

// mustMsgs signs and broadcasts msgs and requires success.
func (s *guardianSuite) mustMsgs(t *testing.T, user ibc.Wallet, msgs ...any) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(gdnGas)}, msgs...)
	require.NoError(t, err)
	require.Zero(t, res.Code, res.RawLog)
	return res
}

// refusedMsg signs and broadcasts msg (so the CLI's own checks do not stop
// it) and requires a refusal. A refusal in CheckTx (ValidateBasic) costs
// nothing; one in the block costs only the fee.
func (s *guardianSuite) refusedMsg(t *testing.T, user ibc.Wallet, msg any) helpers.TxResult {
	t.Helper()
	ubze := s.balance(t, user, chain.Denom)
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(gdnGas)}, msg)
	_, err = helpers.RejectionReason(res, err)
	require.NoError(t, err)

	fee := sdkmath.ZeroInt()
	if res.Height > 0 {
		fee = s.txFee(t, res)
	}
	require.Equal(t, ubze.Sub(fee), s.balance(t, user, chain.Denom), "a refused tx only costs the fee")
	return res
}

func (s *guardianSuite) txFee(t *testing.T, res helpers.TxResult) sdkmath.Int {
	t.Helper()
	fee, err := helpers.TxFee(res, chain.Denom)
	require.NoError(t, err)
	return fee
}

func (s *guardianSuite) balance(t *testing.T, user ibc.Wallet, denom string) sdkmath.Int {
	t.Helper()
	return s.balanceOf(t, user.FormattedAddress(), denom)
}

func (s *guardianSuite) balanceOf(t *testing.T, addr, denom string) sdkmath.Int {
	t.Helper()
	b, err := helpers.Balance(s.ctx, s.bze, addr, denom)
	require.NoError(t, err)
	return b
}

// disabledList is x/circuit's disable list.
func (s *guardianSuite) disabledList(t *testing.T) []string {
	t.Helper()
	res, err := helpers.Query[struct {
		DisabledList []string `json:"disabled_list"`
	}](s.ctx, s.bze, "circuit", "disabled-list")
	require.NoError(t, err)
	return res.DisabledList
}

// poolState is the pool as the chain prints it.
func (s *guardianSuite) poolState(t *testing.T) string {
	t.Helper()
	res, err := helpers.Query[struct {
		Pool json.RawMessage `json:"pool"`
	}](s.ctx, s.bze, "tradebin", "liquidity-pool", s.pool)
	require.NoError(t, err)
	require.NotEmpty(t, res.Pool, "pool %s not found", s.pool)
	return string(res.Pool)
}
