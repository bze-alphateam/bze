package interchaintest_test

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// The dao genesis params of the suite. They are shortened from the defaults
// (within their validation bounds) so every cap is reachable with a few txs
// and every period passes in seconds.
const (
	daoMaxSr          = 2
	daoMaxMembers     = 6
	daoMaxOpen        = 3
	daoMaxMsgs        = 3
	daoMaxBytes       = 4_000
	daoMaxTitle       = 40
	daoMaxSummary     = 80
	daoMinVoting      = "4s"
	daoMaxVoting      = "600s"
	daoExecWindow     = "40s"
	daoExecWindowSecs = 40
	daoCleanupBatch   = 50
	daoCreationFee    = 1_000_000 // 1 BZE
	daoMinLockDays    = 1
	// an expedited proposal settles early only this long after its submission
	daoExpediteDelay     = "8s"
	daoExpediteDelaySecs = 8

	// the default DAO config of the cases
	daoVoting    = "15s"
	daoExpedited = "6s"

	daoFunderFunds = 1_500_000_000_000 // 1,500,000 BZE: about 20 reward programs at 25,000 BZE, DAOs, users
	daoUserFunds   = 200_000_000       // 200 BZE each
	daoStakeMint   = 1_000_000_000_000 // of each factory denom

	// staking rewards of the stake cases
	daoSrPrize    = 10
	daoSrDuration = 1_000 // days of 30 s: they outlive the suite
	daoSrLock     = 1     // day, the min_electorate_lock_days of the suite

	// daoRefuseGas is the fixed gas limit of txs that must be refused: a fixed
	// limit skips the simulation, so the tx is included and its code read
	daoRefuseGas = 1_500_000

	daoModule      = "dao"
	daoEventPrefix = "bze.dao."
)

// TestDao drives every x/dao message and query through the real binary on
// one shared chain, with sequential sub-tests: every message's success path
// and refusals, the exploit catalog D1–D29 and the attack vectors beyond it,
// a funds audit on every tx (the tx's bank transfers are exactly the fee and
// the intended moves, and every account involved changed by exactly that),
// the invariants I1, I2, I3, I5 and I7 after every sub-test, and I4, I6 and
// `genesis validate` on an export at the end. Every sub-test uses fresh
// accounts and DAOs and asserts deltas; a failed sub-test does not stop the
// ones after it.
func TestDao(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx, chain.WithGenesis(daoGenesis()...))
	s := setupDao(t, ctx, net)

	steps := []struct {
		name string
		run  func(*testing.T)
	}{
		{"E9 treasury squatting", s.treasurySquatting},
		{"A1 create-dao", s.createDao},
		{"A2 admin-execute", s.adminExecute},
		{"A3 admin handover", s.adminHandover},
		{"A4 submit-proposal", s.submitProposal},
		{"A5 vote", s.voting},
		{"A6 cancel-proposal", s.cancelProposal},
		{"A7 settle-proposal", s.settleProposal},
		{"A8 expedite", s.expedite},
		{"A9 cleanup-proposal", s.cleanupProposal},
		{"A10 config and metadata", s.updateConfigAndMetadata},
		{"A10 members", s.updateMembers},
		{"A10 electorate", s.updateElectorate},
		{"C stake electorate", s.stakeElectorate},
		{"E1 multi-message txs", s.multiMessageTxs},
		{"E2 settle gas griefing", s.settleGasGriefing},
		{"E3 escrow and module account", s.escrowAccount},
		{"E4 cross-DAO", s.crossDao},
		{"E5 a DAO voting in another DAO", s.daoAsVoter},
		{"E6 self-referencing electorates", s.selfReferencing},
		{"E7 authz and feegrant", s.authzAndFeegrant},
		{"E8 denoms and amounts", s.denomsAndAmounts},
		{"E10 proposer leaves mid-vote", s.proposerLeaves},
		{"E11 cancel games", s.cancelGames},
		{"E12 open-proposal slot griefing", s.slotGriefing},
		{"E13 pinned sources via x/rewards", s.pinnedSources},
		{"B queries", s.queries},
		{"F CLI facts", s.cliFacts},
		{"A11 update-params and E14 mid-flight changes", s.updateParams},
		{"D export and genesis validate", s.exportAndValidate},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			step.run(t)
			if !t.Failed() && step.name != "D export and genesis validate" {
				s.requireInvariants(t)
			}
		})
	}
}

// daoGenesis returns the dao params overrides of the suite.
func daoGenesis() []cosmos.GenesisKV {
	p := "app_state.dao.params."
	return []cosmos.GenesisKV{
		cosmos.NewGenesisKV(p+"max_sr_per_dao", strconv.Itoa(daoMaxSr)),
		cosmos.NewGenesisKV(p+"max_static_members", strconv.Itoa(daoMaxMembers)),
		cosmos.NewGenesisKV(p+"max_open_proposals", strconv.Itoa(daoMaxOpen)),
		cosmos.NewGenesisKV(p+"max_msgs_per_proposal", strconv.Itoa(daoMaxMsgs)),
		cosmos.NewGenesisKV(p+"max_proposal_bytes", strconv.Itoa(daoMaxBytes)),
		cosmos.NewGenesisKV(p+"max_title_len", strconv.Itoa(daoMaxTitle)),
		cosmos.NewGenesisKV(p+"max_summary_len", strconv.Itoa(daoMaxSummary)),
		cosmos.NewGenesisKV(p+"min_voting_period", daoMinVoting),
		cosmos.NewGenesisKV(p+"max_voting_period", daoMaxVoting),
		cosmos.NewGenesisKV(p+"execution_window", daoExecWindow),
		cosmos.NewGenesisKV(p+"cleanup_batch_size", strconv.Itoa(daoCleanupBatch)),
		cosmos.NewGenesisKV(p+"dao_creation_fee", map[string]string{"denom": chain.Denom, "amount": strconv.Itoa(daoCreationFee)}),
		cosmos.NewGenesisKV(p+"dao_creation_fee_destination", "burner"),
		cosmos.NewGenesisKV(p+"min_electorate_lock_days", daoMinLockDays),
		cosmos.NewGenesisKV(p+"min_expedite_delay", daoExpediteDelay),
	}
}

// daoSuite holds the shared chain, the module accounts every audit reads and
// the counters of the ids the chain must hand out next (I5).
type daoSuite struct {
	ctx    context.Context
	bze    *cosmos.CosmosChain
	api    string
	funder ibc.Wallet

	escrow, burner, cp, feeCollector, gov, rewards string

	// factory denoms of the funder: stk is the staking denom of every stake
	// electorate, oth another staking denom (D29), prz the prize of the
	// staking rewards, tok a token for costs and deposits (E8)
	stk, oth, prz, tok string

	params daoParams // the params the chain must hold now

	daos      uint64 // DAOs created so far: their ids are 1..daos
	proposals uint64 // proposals submitted so far: ids 1..proposals
	cleaned   map[uint64]bool
	keySeq    atomic.Int64

	// proposals left by earlier cases for the cleanup case (A9): one per
	// finished status, and one that stays EXECUTION_FAILED
	terminal map[string]uint64
	failed   uint64

	fx *daoFixtures
}

func setupDao(t *testing.T, ctx context.Context, net chain.Network) *daoSuite {
	bze := net.Chain
	s := &daoSuite{ctx: ctx, bze: bze, api: bze.GetHostAPIAddress(), cleaned: map[uint64]bool{}, terminal: map[string]uint64{}}
	s.funder = interchaintest.GetAndFundTestUsers(t, ctx, "dao-funder", sdkmath.NewInt(daoFunderFunds), bze)[0]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	for name, addr := range map[string]*string{
		daoModule:               &s.escrow,
		"txfeecollector_burner": &s.burner,
		"txfeecollector_cp":     &s.cp,
		"fee_collector":         &s.feeCollector,
		"rewards":               &s.rewards,
	} {
		var err error
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	var err error
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)

	// the genesis overrides must have landed: every later case relies on them
	s.params = s.queryParams(t)
	require.EqualValues(t, daoMaxSr, s.params.MaxSrPerDao)
	require.EqualValues(t, daoMaxMembers, s.params.MaxStaticMembers)
	require.EqualValues(t, daoMaxOpen, s.params.MaxOpenProposals)
	require.EqualValues(t, daoMaxMsgs, s.params.MaxMsgsPerProposal)
	require.EqualValues(t, daoMaxBytes, s.params.MaxProposalBytes)
	require.EqualValues(t, daoMaxTitle, s.params.MaxTitleLen)
	require.EqualValues(t, daoMaxSummary, s.params.MaxSummaryLen)
	require.Equal(t, daoMinVoting, s.params.MinVotingPeriod)
	require.Equal(t, daoMaxVoting, s.params.MaxVotingPeriod)
	require.Equal(t, daoExecWindow, s.params.ExecutionWindow)
	require.EqualValues(t, daoCleanupBatch, s.params.CleanupBatchSize)
	require.Equal(t, coinStr(daoCreationFee, chain.Denom), s.params.DaoCreationFee.String())
	require.Equal(t, "burner", s.params.DaoCreationFeeDestination)
	require.EqualValues(t, daoMinLockDays, s.params.MinElectorateLockDays)

	factory := "factory/" + s.funder.FormattedAddress() + "/"
	s.stk, s.oth, s.prz, s.tok = factory+"stk", factory+"oth", factory+"prz", factory+"tok"
	for _, d := range []string{s.stk, s.oth, s.prz, s.tok} {
		s.mustExec(t, s.funder, "tokenfactory", "create-denom", d[len(factory):])
		s.mustExec(t, s.funder, "tokenfactory", "mint", coinStr(daoStakeMint, d))
	}
	return s
}

// users creates n fresh accounts and funds each with ubze (and extra) in one
// MsgMultiSend from the funder.
func (s *daoSuite) users(t *testing.T, n int, ubze int64, extra ...sdk.Coin) []ibc.Wallet {
	t.Helper()
	wallets := make([]ibc.Wallet, n)
	each := sdk.NewCoins(sdk.NewInt64Coin(chain.Denom, ubze)).Add(extra...)
	outputs := make([]map[string]any, n)
	for i := range wallets {
		w, err := helpers.NewWallet(s.ctx, s.bze, "dao-"+strconv.FormatInt(s.keySeq.Add(1), 10))
		require.NoError(t, err)
		wallets[i] = w
		outputs[i] = map[string]any{"address": w.FormattedAddress(), "coins": each}
	}
	total := sdk.NewCoins()
	for range wallets {
		total = total.Add(each...)
	}
	send := map[string]any{
		"@type":   "/cosmos.bank.v1beta1.MsgMultiSend",
		"inputs":  []map[string]any{{"address": s.funder.FormattedAddress(), "coins": total}},
		"outputs": outputs,
	}
	s.mustExecMsgs(t, s.funder, uint64(200_000+40_000*n), send)
	return wallets
}

// user is users(t, 1, daoUserFunds)[0].
func (s *daoSuite) user(t *testing.T) ibc.Wallet {
	t.Helper()
	return s.users(t, 1, daoUserFunds)[0]
}
