package interchaintest_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// The pool prices gpd at 1000 ubze: a fee of gpPauseFee gpd pays for a send and
// is large enough to be swapped on its own. The pool's ubze side is above the
// tradebin MinNativeLiquidityForModuleSwap (1e11), so txfeecollector swaps it.
const (
	gpAdminFunds    = 2_000_000_000_000 // 2,000,000 BZE: a denom, a deep pool, proposals
	gpUserFunds     = 10_000_000_000
	gpGuardianFunds = 10_000_000_000
	gpNodeMinGas    = "0.001" + chain.Denom

	gpMint        = 1_000_000_000_000
	gpPoolGpd     = 200_000_000
	gpPoolBze     = 200_000_000_000
	gpPoolDest    = `{"treasury":"0","burner":"0","providers":"1"}`
	gpUserGpd     = 10_000_000
	gpPauseFee    = 1_000 // gpd per tx
	gpFeeTxs      = 2
	gpBurnerFunds = 5_000_000 // ubze funded to the burner during the pause

	gpSendGas = 200_000
	gpTxGas   = 500_000

	gpWaitBlocks = 3
	gpTickWait   = 6 * time.Minute

	gpCodespace        = "guardian"
	gpResumedEvent     = "bze.guardian.v1.AutomationResumedEvent"
	gpBurnedEvent      = "bze.burner.CoinsBurnedEvent"
	gpBurnerStore      = "burner"
	gpBurnQueuePrefix  = "pbq/" // the burner's periodic burn queue, present only while a burn is pending
	gpUnauthorizedCode = 1101
	gpAlreadyPaused    = 1130
)

// TestGuardianPause proves the guardian's protocol automation pause on a real
// network: a stranger can't pause; the guardian pauses (twice is refused);
// non-native tx fees then stay in the txfeecollector across blocks and a
// periodic burn week tick passes without burning the burner's coins; the
// guardian can't resume; a governance proposal resumes, after which the
// waiting fees are swapped and the burn left pending during the pause runs at
// once. Subtests run in order and share state.
func TestGuardianPause(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()

	guardianMnemonic, guardianAddr := newAccount(t, chain.Bech32Prefix)
	net := chain.Start(t, ctx,
		chain.WithMinGasPrices(gpNodeMinGas),
		chain.WithGenesis(
			cosmos.NewGenesisKV("app_state.guardian.params.guardian", guardianAddr),
			// every week tick marks a periodic burn
			cosmos.NewGenesisKV("app_state.burner.params.periodic_burning_weeks", "1"),
		),
	)
	s := setupGuardianPause(t, ctx, net.Chain, guardianMnemonic, guardianAddr)

	t.Run("1 a stranger can't pause", s.strangerRefused)
	t.Run("2 the guardian pauses", s.guardianPauses)
	t.Run("3 fees wait in the txfeecollector", s.feesWait)
	t.Run("4 the periodic burn waits", s.burnWaits)
	t.Run("5 the guardian can't resume", s.guardianCantResume)
	t.Run("6 governance resumes", s.govResumes)
}

type guardianPauseSuite struct {
	ctx context.Context
	bze *cosmos.CosmosChain

	admin, user, guardian ibc.Wallet
	gov, txFees, burner   string
	gpd                   string
	// waiting is the gpd the paused txfeecollector holds
	waiting sdkmath.Int
}

func setupGuardianPause(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain, guardianMnemonic, guardianAddr string) *guardianPauseSuite {
	t.Helper()
	s := &guardianPauseSuite{ctx: ctx, bze: bze, waiting: sdkmath.ZeroInt()}

	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "gp-admin", sdkmath.NewInt(gpAdminFunds), bze)[0]
	s.user = interchaintest.GetAndFundTestUsers(t, ctx, "gp-user", sdkmath.NewInt(gpUserFunds), bze)[0]
	var err error
	s.guardian, err = interchaintest.GetAndFundTestUserWithMnemonic(ctx, "gp-guardian", guardianMnemonic, sdkmath.NewInt(gpGuardianFunds), bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, s.guardian.FormattedAddress())
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	guardian, err := helpers.QueryGuardian(ctx, bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, guardian, "the genesis guardian")

	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)
	s.txFees, err = helpers.ModuleAddress(ctx, bze, "txfeecollector")
	require.NoError(t, err)
	s.burner, err = helpers.ModuleAddress(ctx, bze, "burner")
	require.NoError(t, err)

	factory := "factory/" + s.admin.FormattedAddress() + "/"
	s.gpd = factory + "gpd"
	s.mustExec(t, s.admin, nil, "tokenfactory", "create-denom", strings.TrimPrefix(s.gpd, factory))
	s.mustExec(t, s.admin, nil, "tokenfactory", "mint", coinStr(gpMint, s.gpd))
	s.mustExec(t, s.admin, []helpers.TxOption{helpers.WithGas(gpTxGas)},
		"tradebin", "create-liquidity-pool", s.gpd, chain.Denom, ammPoolFee, gpPoolDest, "false",
		strconv.Itoa(gpPoolGpd), strconv.Itoa(gpPoolBze))
	s.mustExec(t, s.admin, nil, "bank", "send", s.admin.FormattedAddress(), s.user.FormattedAddress(), coinStr(gpUserGpd, s.gpd))

	require.False(t, s.paused(t), "not paused at genesis")
	return s
}

// ---------------------------------------------------------------------------
// queries and transactions

func (s *guardianPauseSuite) mustExec(t *testing.T, user ibc.Wallet, opts []helpers.TxOption, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(s.ctx, s.bze, user, opts, args...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", args, res.RawLog)
	return res
}

// requireGuardianRefused asserts a tx was included and failed with the guardian code.
func requireGuardianRefused(t *testing.T, res helpers.TxResult, err error, code uint32) {
	t.Helper()
	require.NoError(t, err)
	require.NotZero(t, res.Height, "the tx is included: %s", res.RawLog)
	require.Equal(t, gpCodespace, res.Codespace, res.RawLog)
	require.EqualValues(t, code, res.Code, res.RawLog)
}

// paused reads `bzed q guardian automation-paused`; a false bool may be
// omitted from the output, which reads as false.
func (s *guardianPauseSuite) paused(t *testing.T) bool {
	t.Helper()
	res, err := helpers.Query[struct {
		Paused bool `json:"paused"`
	}](s.ctx, s.bze, "guardian", "automation-paused")
	require.NoError(t, err)
	return res.Paused
}

func (s *guardianPauseSuite) balance(t *testing.T, addr, denom string) sdkmath.Int {
	t.Helper()
	b, err := helpers.Balance(s.ctx, s.bze, addr, denom)
	require.NoError(t, err)
	return b
}

// payGpdFee sends 1 ubze from the user paying the tx fee in gpd.
func (s *guardianPauseSuite) payGpdFee(t *testing.T) helpers.TxResult {
	t.Helper()
	return s.mustExec(t, s.user, []helpers.TxOption{helpers.WithFees(coinStr(gpPauseFee, s.gpd)), helpers.WithGas(gpSendGas)},
		"bank", "send", s.user.FormattedAddress(), s.admin.FormattedAddress(), coinStr(1, chain.Denom))
}

// burnPending tells whether the burner's periodic burn queue holds a pending
// burn (the entry is removed once the burn is done).
func (s *guardianPauseSuite) burnPending(t *testing.T) bool {
	t.Helper()
	n, err := helpers.StorePrefixKeys(s.ctx, s.bze, gpBurnerStore, []byte(gpBurnQueuePrefix))
	require.NoError(t, err)
	return n > 0
}

// balanceAt is the address's balance of denom at the end of height.
func (s *guardianPauseSuite) balanceAt(t *testing.T, addr, denom string, height int64) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Balance struct {
			Amount string `json:"amount"`
		} `json:"balance"`
	}](s.ctx, s.bze, "bank", "balance", addr, denom, "--height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	if res.Balance.Amount == "" {
		return sdkmath.ZeroInt()
	}
	return parseInt(res.Balance.Amount)
}

// blockEvents returns the finalize-block events of eventType in heights
// [from, to], keyed by height.
func (s *guardianPauseSuite) blockEvents(t *testing.T, from, to int64, eventType string) map[int64][]helpers.Event {
	t.Helper()
	out := map[int64][]helpers.Event{}
	for h := from; h <= to; h++ {
		events, err := helpers.BlockEvents(s.ctx, s.bze, h, eventType)
		require.NoError(t, err)
		if len(events) > 0 {
			out[h] = events
		}
	}
	return out
}

func (s *guardianPauseSuite) height(t *testing.T) int64 {
	t.Helper()
	h, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	return h
}

func (s *guardianPauseSuite) waitBlocks(t *testing.T, n int) {
	t.Helper()
	require.NoError(t, testutil.WaitForBlocks(s.ctx, n, s.bze))
}

// ---------------------------------------------------------------------------
// steps

// 1. A stranger's pause is refused, nothing is paused.
func (s *guardianPauseSuite) strangerRefused(t *testing.T) {
	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.user, []helpers.TxOption{helpers.WithGas(gpTxGas)}, "guardian", "pause-automation")
	requireGuardianRefused(t, res, err, gpUnauthorizedCode)
	require.False(t, s.paused(t))
}

// 2. The guardian pauses; pausing again is refused.
func (s *guardianPauseSuite) guardianPauses(t *testing.T) {
	res := s.mustExec(t, s.guardian, nil, "guardian", "pause-automation")
	require.Len(t, helpers.TxEvents(res, "bze.guardian.v1.AutomationPausedEvent"), 1)
	require.True(t, s.paused(t))

	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.guardian, []helpers.TxOption{helpers.WithGas(gpTxGas)}, "guardian", "pause-automation")
	requireGuardianRefused(t, res, err, gpAlreadyPaused)
	require.True(t, s.paused(t))
}

// 3. gpd fees, swappable on their own, stay in the txfeecollector across
// blocks instead of being swapped and forwarded in EndBlock.
func (s *guardianPauseSuite) feesWait(t *testing.T) {
	start := s.balance(t, s.txFees, s.gpd)
	for i := 0; i < gpFeeTxs; i++ {
		s.payGpdFee(t)
		s.waitBlocks(t, gpWaitBlocks)
	}
	s.waiting = start.AddRaw(gpFeeTxs * gpPauseFee)
	require.Equal(t, s.waiting, s.balance(t, s.txFees, s.gpd), "the gpd fees wait in the txfeecollector")
}

// 4. The burner is funded during the pause; a week tick (a burning one, the
// interval is 1) passes and nothing is burned.
func (s *guardianPauseSuite) burnWaits(t *testing.T) {
	s.mustExec(t, s.user, []helpers.TxOption{helpers.WithGas(gpTxGas)}, "burner", "fund-burner", coinStr(gpBurnerFunds, chain.Denom))
	before := s.balance(t, s.burner, chain.Denom)
	require.True(t, before.GTE(sdkmath.NewInt(gpBurnerFunds)), "the burner holds the funds: %s", before)

	week, err := helpers.CurrentEpoch(s.ctx, s.bze, "week")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(s.ctx, gpTickWait)
	defer cancel()
	require.NoError(t, helpers.WaitForEpoch(ctx, s.bze, "week", week+1))
	s.waitBlocks(t, gpWaitBlocks)

	require.True(t, s.burnPending(t), "the burning week tick left a burn pending")
	require.True(t, s.balance(t, s.burner, chain.Denom).GTE(before), "nothing burned on the week tick")
	require.Equal(t, s.waiting, s.balance(t, s.txFees, s.gpd), "the fees still wait")
	require.True(t, s.paused(t))
}

// 5. The guardian signing MsgResumeAutomation itself is refused.
func (s *guardianPauseSuite) guardianCantResume(t *testing.T) {
	res, err := helpers.ExecMsgs(s.ctx, s.bze, s.guardian, []helpers.TxOption{helpers.WithGas(gpTxGas)}, map[string]any{
		"@type":     "/bze.guardian.v1.MsgResumeAutomation",
		"authority": s.guardian.FormattedAddress(),
	})
	requireGuardianRefused(t, res, err, gpUnauthorizedCode)
	require.True(t, s.paused(t))
}

// 6. A governance proposal resumes the automation: the next blocks swap the
// waiting fees and run the burn the week tick left pending, without waiting
// for the next tick. The burner may receive burn-bound fees again right after,
// so the burn is asserted on the burn events, not on an empty burner.
func (s *guardianPauseSuite) govResumes(t *testing.T) {
	from := s.height(t)
	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: []any{map[string]any{
			"@type":     "/bze.guardian.v1.MsgResumeAutomation",
			"authority": s.gov,
		}},
		Title:   "resume protocol automation",
		Summary: "e2e: governance resumes the protocol automation",
	})
	require.NoError(t, err)
	require.False(t, s.paused(t))

	s.waitBlocks(t, gpWaitBlocks)
	to := s.height(t)
	require.True(t, s.balance(t, s.txFees, s.gpd).IsZero(), "the waiting fees are swapped")

	resumed := s.blockEvents(t, from, to, gpResumedEvent)
	require.Len(t, resumed, 1, "one AutomationResumedEvent")
	var resumedAt int64
	for h, events := range resumed {
		require.Len(t, events, 1)
		require.Equal(t, s.gov, helpers.Unquote(events[0].Attributes["authority"]), "resumed by the gov authority")
		resumedAt = h
	}

	// the first burn after the resume is the pending one (a later week tick may burn again)
	burns := s.blockEvents(t, resumedAt, to, gpBurnedEvent)
	require.NotEmpty(t, burns, "the pending burn ran after the resume")
	first := to
	for h := range burns {
		first = min(first, h)
	}
	pending := s.balanceAt(t, s.burner, chain.Denom, first-1)
	require.True(t, pending.GTE(sdkmath.NewInt(gpBurnerFunds)), "the burner held the funds before the burn: %s", pending)

	events, err := helpers.BlockEvents(s.ctx, s.bze, first, "burn")
	require.NoError(t, err)
	_, burned, err := helpers.SupplyChange(events)
	require.NoError(t, err)
	require.Equal(t, pending, burned[s.burner].AmountOf(chain.Denom), "the pending burn burns the whole ubze balance at height %d", first)
	require.False(t, s.burnPending(t), "no burn pending after the resume")

	// fees paid now are swapped in their own block again
	res := s.payGpdFee(t)
	s.waitBlocks(t, 1)
	require.True(t, s.balance(t, s.txFees, s.gpd).IsZero(), "the fee of block %d is swapped", res.Height)
}
