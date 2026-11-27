package interchaintest_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	// daoClosedChannel is the transfer channel TestDao's genesis closes
	daoClosedChannel = "channel-0"
	// the deposit of the two proposals carried through the export
	daoImpDeposit = 7_000
	// what the imported proposal that executes sends from its treasury
	daoImpSend = 300
	// the guardian error of a pause while paused
	daoAlreadyPausedErr = 1130
	// how long the node may take to produce blocks from the export
	daoImpStartTimeout = 3 * time.Minute
)

// daoImport is what beforeExport leaves open for the node start from the
// export and what the chain showed right before the export.
type daoImport struct {
	members   []ibc.Wallet
	rcpt      string
	short     daoRef // the DAO of the proposal that executes after the start
	cancelPid uint64 // a proposal in voting until long after the start
	settlePid uint64 // a passed proposal whose voting ends before the settle

	digest   string
	escrow   sdk.Coins
	treasury sdk.Coins
	ballots  map[uint64]string
	guardian string
}

// setupGuardian funds the genesis guardian of TestDao.
func (s *daoSuite) setupGuardian(t *testing.T, mnemonic, addr string) {
	t.Helper()
	var err error
	s.guardian, err = interchaintest.GetAndFundTestUserWithMnemonic(s.ctx, "dao-guardian", mnemonic, sdkmath.NewInt(daoUserFunds), s.bze)
	require.NoError(t, err)
	require.Equal(t, addr, s.guardian.FormattedAddress())
	guardian, err := helpers.QueryGuardian(s.ctx, s.bze)
	require.NoError(t, err)
	require.Equal(t, addr, guardian, "the genesis guardian")
	require.Equal(t, []string{daoClosedChannel}, s.closedChannels(t), "the genesis closed channel")
	require.False(t, s.automationPaused(t), "not paused at genesis")
}

// beforeExport leaves two proposals with a deposit open through the export:
// one with a 600 s voting period and a no ballot, cancelled after the start,
// and one passed by both members, settled after the start. The guardian
// pauses the protocol automation. What the chain shows now is kept for
// startFromExport.
func (s *daoSuite) beforeExport(t *testing.T) {
	m := s.users(t, 2, daoUserFunds)
	rcpt := s.user(t)
	cfg := defaultCfg()
	cfg.deposit = coinPtr(daoImpDeposit, chain.Denom)

	cfg.voting = daoMaxVoting
	long := s.createStatic(t, m[0], cfg, addrs(m...), "")
	cfg.voting = daoVoting
	short := s.createStatic(t, m[0], cfg, addrs(m...), "")
	s.fund(t, short.addr, ubze(daoImpSend))

	cancelPid := s.submit(t, m[0], long, "import cancel", nil)
	s.vote(t, m[1], cancelPid, "no")
	settlePid := s.submit(t, m[1], short, "import settle",
		[]map[string]any{bankSend(short.addr, rcpt.FormattedAddress(), ubze(daoImpSend))})
	s.vote(t, m[0], settlePid, "yes")
	s.vote(t, m[1], settlePid, "yes")

	res := s.mustExec(t, s.guardian, "guardian", "pause-automation")
	require.Len(t, helpers.EventsWith(helpers.TxEvents(res, "bze.guardian.v1.AutomationPausedEvent"), map[string]string{
		"signer": s.guardian.FormattedAddress(),
	}), 1, "one AutomationPausedEvent from the guardian")
	require.True(t, s.automationPaused(t))

	imp := &daoImport{
		members: m, rcpt: rcpt.FormattedAddress(), short: short,
		cancelPid: cancelPid, settlePid: settlePid,
		ballots: map[uint64]string{},
	}
	s.snapshotImport(t, imp)
	require.True(t, imp.escrow.AmountOf(chain.Denom).GTE(sdkmath.NewInt(2*daoImpDeposit)), "the escrow holds both deposits")
	s.imp = imp
}

func (s *daoSuite) snapshotImport(t *testing.T, imp *daoImport) {
	t.Helper()
	imp.digest = s.stateDigest(t)
	imp.escrow = s.allBalances(t, s.escrow)
	imp.treasury = s.allBalances(t, imp.short.addr)
	for _, pid := range []uint64{imp.cancelPid, imp.settlePid} {
		imp.ballots[pid] = mustJSON(s.ballots(t, pid))
	}
	var err error
	imp.guardian, err = helpers.QueryGuardian(s.ctx, s.bze)
	require.NoError(t, err)
}

// startFromExport starts the validator again from the export D validated:
// its data is reset, the export becomes its genesis (with a fresh genesis
// time, as a real restart from an export sets one) and the chain continues at
// the export's initial height. Every DAO, proposal, ballot and the escrow are as they were,
// the invariants hold, the guardian params, the pause and the closed channel
// are live (a second pause is refused as already paused), and the two
// imported proposals release their deposits from the imported escrow: one is
// cancelled, the other settles and executes.
func (s *daoSuite) startFromExport(t *testing.T) {
	if s.imp == nil || s.exported == nil {
		t.Skip("needs the steps before and the export")
	}
	imp := s.imp
	node := s.bze.Validators[0]
	initial := genesisInitialHeight(t, s.exported)

	require.NoError(t, node.UnsafeResetAll(s.ctx))
	require.NoError(t, node.OverwriteGenesisFile(s.ctx, setGenesisTime(t, s.exported, time.Now().UTC())))
	require.NoError(t, s.bze.StartAllNodes(s.ctx))
	// a new container: its host ports are new too
	s.api = s.bze.GetHostAPIAddress()

	ctx, cancel := context.WithTimeout(s.ctx, daoImpStartTimeout)
	defer cancel()
	_, err := helpers.WaitForHeight(ctx, s.bze, initial+2)
	require.NoError(t, err, "the node does not produce blocks from the export")
	st, err := node.Client.Status(s.ctx)
	require.NoError(t, err)
	require.Equal(t, initial, st.SyncInfo.EarliestBlockHeight, "the chain started at the export's initial height")

	after := &daoImport{short: imp.short, cancelPid: imp.cancelPid, settlePid: imp.settlePid, ballots: map[uint64]string{}}
	s.snapshotImport(t, after)
	require.Equal(t, imp.digest, after.digest, "every dao and proposal as before the export")
	require.Equal(t, imp.ballots, after.ballots, "the ballots as before the export")
	require.Equal(t, imp.escrow.String(), after.escrow.String(), "the escrow as before the export")
	require.Equal(t, imp.treasury.String(), after.treasury.String(), "the treasury as before the export")
	require.Equal(t, imp.guardian, after.guardian, "the guardian as before the export")
	require.Equal(t, s.guardian.FormattedAddress(), after.guardian)
	require.True(t, s.automationPaused(t), "still paused")
	require.Equal(t, []string{daoClosedChannel}, s.closedChannels(t), "still closed")
	s.refuse(t, s.guardian, refusal{codespace: gdnCodespace, code: daoAlreadyPausedErr}, "guardian", "pause-automation")
	s.requireInvariants(t)

	s.cancel(t, imp.members[0], imp.cancelPid)
	s.waitVotingEnd(t, imp.settlePid)
	res := s.settleFirst(t, imp.members[0], imp.settlePid, "EXECUTED", flow(imp.short.addr, imp.rcpt, ubze(daoImpSend)))
	require.Equal(t, "true", s.event(t, res, "EventProposalExecuted").Attributes["success"])
	require.Equal(t, imp.escrow.Sub(ubze(2*daoImpDeposit)...).String(), s.allBalances(t, s.escrow).String(),
		"both deposits left the imported escrow")
}

// automationPaused reads `bzed q guardian automation-paused`; a false bool may
// be omitted from the output, which reads as false.
func (s *daoSuite) automationPaused(t *testing.T) bool {
	t.Helper()
	res, err := helpers.Query[struct {
		Paused bool `json:"paused"`
	}](s.ctx, s.bze, "guardian", "automation-paused")
	require.NoError(t, err)
	return res.Paused
}

func (s *daoSuite) closedChannels(t *testing.T) []string {
	t.Helper()
	res, err := helpers.Query[struct {
		ChannelIds []string `json:"channel_ids"`
	}](s.ctx, s.bze, "guardian", "closed-channels")
	require.NoError(t, err)
	return res.ChannelIds
}

// genesisInitialHeight is the initial_height of a genesis document.
func genesisInitialHeight(t *testing.T, genesis []byte) int64 {
	t.Helper()
	var doc struct {
		InitialHeight json.RawMessage `json:"initial_height"`
	}
	require.NoError(t, json.Unmarshal(genesis, &doc))
	var h helpers.Int64
	require.NoError(t, json.Unmarshal(doc.InitialHeight, &h), "initial_height %s", doc.InitialHeight)
	require.Positive(t, int64(h))
	return int64(h)
}

// setGenesisTime returns the genesis document with only genesis_time changed.
func setGenesisTime(t *testing.T, genesis []byte, at time.Time) []byte {
	t.Helper()
	var doc map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(genesis, &doc))
	require.Contains(t, doc, "genesis_time")
	doc["genesis_time"] = json.RawMessage(strconv.Quote(at.Format(time.RFC3339Nano)))
	out, err := json.Marshal(doc)
	require.NoError(t, err)
	return out
}
