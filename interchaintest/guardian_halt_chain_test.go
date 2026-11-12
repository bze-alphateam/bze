package interchaintest_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	ghFunds = 100_000_000_000 // 100,000 BZE each: tx fees and a gov deposit
	ghGas   = 300_000         // fixed, so a refused tx is included and its code read

	ghCodespace        = "guardian"
	ghUnauthorizedCode = 1101

	// the release plan governance schedules first, far in the future
	ghReleasePlan  = "v99.0.0"
	ghReleaseDelta = 100_000

	ghReason      = "e2e: emergency halt"
	ghHaltTimeout = 2 * time.Minute
	// how long the chain must stay at the halt height
	ghHaltSettle = 15 * time.Second

	ghEventType = "bze.guardian.v1.ChainHaltScheduledEvent"
)

// TestGuardianHaltChain stops a chain with the guardian's MsgHaltChain: a
// stranger is refused and the scheduled release plan stays; the guardian's
// halt replaces it with a plan at the next block, visible through the upgrade
// query, and the chain stops producing blocks at that height. The chain is
// not restarted. It runs on its own network, which ends halted.
func TestGuardianHaltChain(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()

	guardianMnemonic, guardianAddr := newAccount(t, chain.Bech32Prefix)
	net := chain.Start(t, ctx, chain.WithGenesis(cosmos.NewGenesisKV("app_state.guardian.params.guardian", guardianAddr)))
	s := setupGuardianHalt(t, ctx, net.Chain, guardianMnemonic, guardianAddr)

	t.Run("1 governance schedules a release", s.scheduleRelease)
	t.Run("2 a stranger cannot halt", s.strangerRefused)
	t.Run("3 the guardian halts the chain", s.guardianHalts)
}

type guardianHaltSuite struct {
	ctx      context.Context
	bze      *cosmos.CosmosChain
	guardian ibc.Wallet
	stranger ibc.Wallet
	gov      string
}

// ghPlan is the output of `bzed q upgrade plan`.
type ghPlan struct {
	Plan *struct {
		Name   string        `json:"name"`
		Height helpers.Int64 `json:"height"`
		Info   string        `json:"info"`
	} `json:"plan"`
}

func setupGuardianHalt(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain, guardianMnemonic, guardianAddr string) *guardianHaltSuite {
	t.Helper()
	s := &guardianHaltSuite{ctx: ctx, bze: bze}

	var err error
	s.guardian, err = interchaintest.GetAndFundTestUserWithMnemonic(ctx, "guardian", guardianMnemonic, sdkmath.NewInt(ghFunds), bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, s.guardian.FormattedAddress())
	s.stranger = interchaintest.GetAndFundTestUsers(t, ctx, "stranger", sdkmath.NewInt(ghFunds), bze)[0]

	guardian, err := helpers.QueryGuardian(ctx, bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, guardian, "the genesis guardian")

	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)

	return s
}

func (s *guardianHaltSuite) plan(t *testing.T) ghPlan {
	t.Helper()
	p, err := helpers.Query[ghPlan](s.ctx, s.bze, "upgrade", "plan")
	require.NoError(t, err)
	require.NotNil(t, p.Plan, "a plan is scheduled")
	return p
}

func (s *guardianHaltSuite) haltTx(t *testing.T, user ibc.Wallet) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(ghGas)}, "guardian", "halt-chain", ghReason)
	require.NoError(t, err)
	return res
}

// 1. A release plan through governance, so the halt can be seen replacing it.
func (s *guardianHaltSuite) scheduleRelease(t *testing.T) {
	height, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	_, err = helpers.SubmitAndPassProposal(s.ctx, s.bze, s.stranger, helpers.Proposal{
		Messages: []any{map[string]any{
			"@type":     "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
			"authority": s.gov,
			"plan":      map[string]any{"name": ghReleasePlan, "height": strconv.FormatInt(height+ghReleaseDelta, 10), "info": "e2e"},
		}},
		Title:   "Upgrade " + ghReleasePlan,
		Summary: "e2e: a release far in the future",
	})
	require.NoError(t, err)
	require.Equal(t, ghReleasePlan, s.plan(t).Plan.Name)
}

// 2. Only the guardian and the gov authority can halt: a stranger's tx is
// included and refused, the release plan is untouched and blocks go on.
func (s *guardianHaltSuite) strangerRefused(t *testing.T) {
	res := s.haltTx(t, s.stranger)
	require.NotZero(t, res.Height, "the refused tx is included")
	require.EqualValues(t, ghUnauthorizedCode, res.Code, res.RawLog)
	require.Equal(t, ghCodespace, res.Codespace)
	require.Empty(t, helpers.TxEvents(res, ghEventType))

	require.Equal(t, ghReleasePlan, s.plan(t).Plan.Name)
	ctx, cancel := context.WithTimeout(s.ctx, ghHaltTimeout)
	defer cancel()
	_, err := helpers.WaitForHeight(ctx, s.bze, res.Height+2)
	require.NoError(t, err, "the chain keeps producing blocks")
}

// 3. The guardian's halt replaces the release plan with emergency-halt-H at
// H = its tx height + 1, and the chain stops at H: the block store reaches H
// and no block follows. The plan is read from the last state, H - 1.
func (s *guardianHaltSuite) guardianHalts(t *testing.T) {
	res := s.haltTx(t, s.guardian)
	require.Zero(t, res.Code, res.RawLog)
	haltHeight := res.Height + 1
	name := "emergency-halt-" + strconv.FormatInt(haltHeight, 10)

	events := helpers.TxEvents(res, ghEventType)
	require.Len(t, events, 1)
	require.Equal(t, map[string]string{
		"signer": s.guardian.FormattedAddress(),
		"name":   name,
		"height": strconv.FormatInt(haltHeight, 10),
		"reason": ghReason,
	}, withoutMode(events[0].Attributes))

	ctx, cancel := context.WithTimeout(s.ctx, ghHaltTimeout)
	defer cancel()
	require.NoError(t, helpers.WaitForHalt(ctx, s.bze, haltHeight, ghHaltSettle))
	t.Logf("halted at %d", haltHeight)

	p := s.plan(t)
	require.Equal(t, name, p.Plan.Name, "the halt replaced the release plan")
	require.Equal(t, haltHeight, int64(p.Plan.Height))
	require.Equal(t, "emergency halt", p.Plan.Info, "the reason never goes into the plan info")
}
