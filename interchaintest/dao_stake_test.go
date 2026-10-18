package interchaintest_test

import (
	"fmt"
	"strconv"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// daoFixtures are reward programs shared by the refusal cases: srA, srB and
// srC stake stk with the minimum lock, srLow stakes stk with lock 0 (below
// the minimum, D28), srOth stakes oth (D29), srIdle stakes stk and never has
// a staker. Denom rewards exist for stk and oth.
type daoFixtures struct {
	srA, srB, srC, srLow, srOth, srIdle string
}

func (s *daoSuite) fixtures(t *testing.T) *daoFixtures {
	t.Helper()
	if s.fx != nil {
		return s.fx
	}
	fx := &daoFixtures{
		srA:    s.newSR(t, daoSrLock, s.stk, daoSrDuration),
		srB:    s.newSR(t, daoSrLock, s.stk, daoSrDuration),
		srC:    s.newSR(t, daoSrLock, s.stk, daoSrDuration),
		srLow:  s.newSR(t, 0, s.stk, daoSrDuration),
		srOth:  s.newSR(t, daoSrLock, s.oth, daoSrDuration),
		srIdle: s.newSR(t, daoSrLock, s.stk, daoSrDuration),
	}
	for _, d := range []string{s.stk, s.oth} {
		s.mustExec(t, s.funder, "rewards", "create-denom-reward", d)
	}
	s.fx = fx
	return fx
}

// newSR creates a staking reward of the funder and returns its id.
func (s *daoSuite) newSR(t *testing.T, lock int, denom string, duration int) string {
	t.Helper()
	res := s.mustExec(t, s.funder, "rewards", "create-staking-reward",
		strconv.Itoa(daoSrPrize), s.prz, denom, strconv.Itoa(duration), "1", strconv.Itoa(lock))
	events := helpers.TxEvents(res, "bze.rewards.StakingRewardCreateEvent")
	require.Len(t, events, 1)
	return events[0].Attributes["reward_id"]
}

func (s *daoSuite) joinSR(t *testing.T, user ibc.Wallet, id string, amount int64) helpers.TxResult {
	t.Helper()
	return s.run(t, user, nil, []daoFlow{flow(user.FormattedAddress(), s.rewards, coins(amount, s.stk))}, "rewards", "join-staking", id, strconv.FormatInt(amount, 10))
}

// exitSR exits a staking reward: the stake is queued for unlock (no
// transfer in the tx), pending prizes are paid in prz (left out of the audit).
func (s *daoSuite) exitSR(t *testing.T, user ibc.Wallet, id string) helpers.TxResult {
	t.Helper()
	return s.run(t, user, nil, nil, "rewards", "exit-staking", id)
}

func (s *daoSuite) joinDR(t *testing.T, user ibc.Wallet, denom string, amount int64) helpers.TxResult {
	t.Helper()
	return s.run(t, user, nil, []daoFlow{flow(user.FormattedAddress(), s.rewards, coins(amount, denom))}, "rewards", "join-denom-reward", denom, strconv.FormatInt(amount, 10))
}

func (s *daoSuite) exitDR(t *testing.T, user ibc.Wallet, denom string) helpers.TxResult {
	t.Helper()
	return s.run(t, user, nil, nil, "rewards", "exit-denom-reward", denom)
}

func (s *daoSuite) srStaked(t *testing.T, id string) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		StakingReward struct {
			StakedAmount string `json:"staked_amount"`
		} `json:"staking_reward"`
	}](s.ctx, s.bze, "rewards", "get-staking-reward", id)
	require.NoError(t, err)
	return parseInt(res.StakingReward.StakedAmount)
}

func (s *daoSuite) srExists(t *testing.T, id string) bool {
	t.Helper()
	_, err := helpers.Query[map[string]any](s.ctx, s.bze, "rewards", "get-staking-reward", id)
	return err == nil
}

func (s *daoSuite) drStaked(t *testing.T, denom string) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		DenomReward struct {
			StakedAmount string `json:"staked_amount"`
		} `json:"denom_reward"`
	}](s.ctx, s.bze, "rewards", "denom-reward", denom)
	require.NoError(t, err)
	return parseInt(res.DenomReward.StakedAmount)
}

// stakeSource is one electorate source: a staking reward (sr) or the denom
// reward of a staking denom (dr).
type stakeSource struct {
	s      *daoSuite
	sr, dr string
}

func (k stakeSource) key() string {
	if k.sr != "" {
		return srSource(k.sr)
	}
	return drSource(k.dr)
}

func (k stakeSource) electorate() daoElectorate {
	if k.sr != "" {
		return daoElectorate{SrIDs: []string{k.sr}}
	}
	return daoElectorate{DrDenom: k.dr}
}

func (k stakeSource) join(t *testing.T, user ibc.Wallet, amount int64) helpers.TxResult {
	t.Helper()
	if k.sr != "" {
		return k.s.joinSR(t, user, k.sr, amount)
	}
	return k.s.joinDR(t, user, k.dr, amount)
}

func (k stakeSource) exit(t *testing.T, user ibc.Wallet) helpers.TxResult {
	t.Helper()
	if k.sr != "" {
		return k.s.exitSR(t, user, k.sr)
	}
	return k.s.exitDR(t, user, k.dr)
}

func (k stakeSource) staked(t *testing.T) sdkmath.Int {
	t.Helper()
	if k.sr != "" {
		return k.s.srStaked(t, k.sr)
	}
	return k.s.drStaked(t, k.dr)
}

// joinMsg and exitMsg are the x/rewards messages of the source (gogoproto JSON).
func (k stakeSource) joinMsg(user string, amount int64) map[string]any {
	if k.sr != "" {
		return map[string]any{"@type": "/bze.rewards.MsgJoinStaking", "creator": user, "reward_id": k.sr, "amount": strconv.FormatInt(amount, 10)}
	}
	return map[string]any{"@type": "/bze.rewards.MsgJoinDenomReward", "creator": user, "denom": k.dr, "amount": strconv.FormatInt(amount, 10)}
}

func (k stakeSource) exitMsg(user string) map[string]any {
	if k.sr != "" {
		return map[string]any{"@type": "/bze.rewards.MsgExitStaking", "creator": user, "reward_id": k.sr}
	}
	return map[string]any{"@type": "/bze.rewards.MsgExitDenomReward", "creator": user, "denom": k.dr}
}

// passes is the regular tally rule of x/dao (D17: exact, ≥ on both bars).
func passes(t *testing.T, p daoProposal) bool {
	t.Helper()
	den, yes, no, abstain := parseInt(p.PinnedDenominator), parseInt(p.Yes), parseInt(p.No), parseInt(p.Abstain)
	decided := yes.Add(no)
	if !den.IsPositive() || !decided.IsPositive() {
		return false
	}
	q, th := ratio(t, p.PinnedConfig.MinQuorum), ratio(t, p.PinnedConfig.Threshold)
	quorum := sdkmath.LegacyNewDecFromInt(decided.Add(abstain)).GTE(q.MulInt(den))
	threshold := sdkmath.LegacyNewDecFromInt(yes).GTE(th.MulInt(decided))
	return quorum && threshold
}

// parallel runs fn for every wallet, 16 at a time.
func (s *daoSuite) parallel(t *testing.T, ws []ibc.Wallet, fn func(w ibc.Wallet) error) {
	t.Helper()
	var eg errgroup.Group
	eg.SetLimit(16)
	for _, w := range ws {
		eg.Go(func() error { return fn(w) })
	}
	require.NoError(t, eg.Wait())
}

func (s *daoSuite) execOK(user ibc.Wallet, args ...string) error {
	res, err := helpers.ExecTx(s.ctx, s.bze, user, args...)
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("tx %v of %s: code %d: %s", args, user.FormattedAddress(), res.Code, res.RawLog)
	}
	return nil
}

// A9. cleanup-proposal: every finished status (one with 0 ballots), the
// refusals, and a few hundred ballots cleaned in batches with a cursor
// (D19/D20), with the settle and exit gas measured on the way.
func (s *daoSuite) cleanupProposal(t *testing.T) {
	anyone := s.user(t)
	for _, key := range []string{"REJECTED", "EXECUTED", "EXPIRED", "CANCELLED", "CANCELLED-0-ballots"} {
		t.Run(key, func(t *testing.T) {
			pid, ok := s.terminal[key]
			require.True(t, ok, "an earlier case left no %s proposal", key)
			ballots := len(s.ballots(t, pid))
			if key == "CANCELLED-0-ballots" {
				require.Zero(t, ballots)
			}
			processed, done := s.cleanup(t, anyone, pid, 0)
			require.True(t, done)
			require.EqualValues(t, ballots, processed)
			require.False(t, s.proposalExists(t, pid), "a cleaned proposal answers NotFound")
			require.Empty(t, s.ballots(t, pid))
			s.refuse(t, anyone, daoCode(1123), "dao", "cleanup-proposal", daoU64(pid))
		})
	}
	t.Run("not finished", func(t *testing.T) {
		require.NotZero(t, s.failed)
		s.requireStatus(t, s.failed, "EXECUTION_FAILED")
		s.refuse(t, anyone, daoCode(1131), "dao", "cleanup-proposal", daoU64(s.failed))
		d := s.createStatic(t, anyone, defaultCfg(), addrs(anyone), "")
		pid := s.submit(t, anyone, d, "still voting", nil)
		s.refuse(t, anyone, daoCode(1131), "dao", "cleanup-proposal", daoU64(pid))
	})

	t.Run("D19 D20 a few hundred ballots", func(t *testing.T) {
		const voters = 200
		sr := s.newSR(t, daoSrLock, s.stk, daoSrDuration)
		whale := s.users(t, 1, daoUserFunds, coins(1_000_000, s.stk)...)[0]
		solo := s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0]
		crowd := s.users(t, voters, 5_000_000, coins(1_000, s.stk)...)
		s.joinSR(t, whale, sr, 1_000_000)
		s.joinSR(t, solo, sr, 1_000)
		s.parallel(t, crowd, func(w ibc.Wallet) error { return s.execOK(w, "rewards", "join-staking", sr, "1000") })

		cfg := defaultCfg()
		cfg.voting = "150s"
		d := s.createStake(t, whale, cfg, daoElectorate{SrIDs: []string{sr}}, "")
		many := s.submit(t, whale, d, "many ballots", nil)
		few := s.submit(t, whale, d, "few ballots", nil)
		require.Equal(t, "1201000", s.prop(t, many).PinnedDenominator)
		s.parallel(t, crowd, func(w ibc.Wallet) error { return s.execOK(w, "dao", "vote", daoU64(many), "yes") })
		s.vote(t, whale, many, "yes")
		s.vote(t, whale, few, "yes")
		v1, v2 := crowd[0], crowd[1]
		s.vote(t, v2, few, "yes")
		s.vote(t, solo, few, "yes")
		require.Len(t, s.ballots(t, many), voters+1)
		require.Len(t, s.ballots(t, few), 3)

		// an exit costs with the voter's own ballots only, not with the
		// ballots of others on the same proposals
		g1 := s.exitSR(t, v1, sr).GasUsed      // 1 ballot, on the 201-ballot proposal
		gSolo := s.exitSR(t, solo, sr).GasUsed // 1 ballot, on the 3-ballot proposal
		g2 := s.exitSR(t, v2, sr).GasUsed      // 2 ballots
		t.Logf("exit gas: 1 ballot among 201 %d, 1 ballot among 3 %d, 2 ballots %d", g1, gSolo, g2)
		require.InDelta(t, g1, gSolo, 3_000, "one ballot costs the same whatever the proposal's size")
		require.Greater(t, g2, g1, "an exit with two ballots costs more than with one")
		require.Equal(t, "0", s.mustBallot(t, many, v1.FormattedAddress()).Weight, "D2: an exit zeroes the ballot")

		s.waitVotingEnd(t, few)
		gMany := s.settleFirst(t, anyone, many, "EXECUTED").GasUsed
		gFew := s.settleFirst(t, anyone, few, "EXECUTED").GasUsed
		t.Logf("settle gas: %d ballots %d, %d ballots %d", voters+1, gMany, len(s.ballots(t, few)), gFew)
		require.InDelta(t, gFew, gMany, 3_000, "settle cost does not depend on the number of ballots (D19)")

		total := uint64(0)
		processed, done := s.cleanup(t, anyone, many, 500)
		require.False(t, done)
		require.EqualValues(t, daoCleanupBatch, processed, "--limit above cleanup_batch_size is capped")
		total += processed
		cursor := s.prop(t, many).CleanupCursor
		require.NotEmpty(t, cursor)
		processed, done = s.cleanup(t, anyone, many, 20)
		require.False(t, done)
		require.EqualValues(t, 20, processed)
		total += processed
		require.NotEqual(t, cursor, s.prop(t, many).CleanupCursor, "the cursor moves on")
		require.Len(t, s.ballots(t, many), voters+1-int(total))
		for !done {
			processed, done = s.cleanup(t, anyone, many, 0)
			total += processed
		}
		require.EqualValues(t, voters+1, total, "every ballot is deleted exactly once")
		require.False(t, s.proposalExists(t, many))
		s.refuse(t, anyone, daoCode(1123), "dao", "cleanup-proposal", daoU64(many))
	})
}

// C. Stake electorates (D2–D6), on a staking reward and on a denom reward.
func (s *daoSuite) stakeElectorate(t *testing.T) {
	s.fixtures(t)
	t.Run("staking reward", func(t *testing.T) {
		s.stakeCase(t, stakeSource{s: s, sr: s.newSR(t, daoSrLock, s.stk, daoSrDuration)}, false)
	})
	t.Run("denom reward", func(t *testing.T) {
		// dr/stk is pinned since A1, so every join below is recorded
		s.stakeCase(t, stakeSource{s: s, dr: s.stk}, true)
	})
}

func (s *daoSuite) stakeCase(t *testing.T, src stakeSource, pinned bool) {
	v1, v2, j, whale := s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0],
		s.users(t, 1, daoUserFunds, coins(1_500, s.stk)...)[0],
		s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0],
		s.users(t, 1, daoUserFunds, coins(1_000_000, s.stk)...)[0]
	res := src.join(t, v1, 1_000)
	src.join(t, v2, 1_000)
	h, found := s.joinRecord(t, src.key(), v1.FormattedAddress())
	if pinned {
		require.True(t, found, "a join on a pinned source is recorded")
		require.Equal(t, res.Height, h)
	} else {
		require.False(t, found, "a join on an unpinned source has no record")
	}

	cfg := defaultCfg()
	cfg.voting, cfg.open = "40s", true
	d := s.createStake(t, v1, cfg, src.electorate(), "")

	// D3: a join and a submit in one tx (the same block): the join weighs 0
	stakedBefore := src.staked(t)
	tx := s.runMsgs(t, j, 1_500_000, []string{d.addr}, []daoFlow{flow(j.FormattedAddress(), s.rewards, coins(1_000, s.stk))},
		src.joinMsg(j.FormattedAddress(), 1_000), submitMsg(j.FormattedAddress(), d.id, "join and submit", nil, false))
	pid := s.submitted(t, tx, d, j.FormattedAddress())
	p := s.prop(t, pid)
	require.Equal(t, stakedBefore.AddRaw(1_000).String(), p.PinnedDenominator, "the denominator is the stake at submit, the same-block join included")
	rec, found := s.joinRecord(t, src.key(), j.FormattedAddress())
	require.True(t, found)
	require.Equal(t, tx.Height, rec, "the join is recorded at the submit height")
	require.True(t, s.votingPower(t, d.id, j.FormattedAddress()).eligible(t, pid).IsZero())
	s.refuse(t, j, daoCode(1119), "dao", "vote", daoU64(pid), "yes")

	// D4 and D6: a whale joins after the submit, weighs 0, the denominator stays
	src.join(t, whale, 1_000_000)
	vp := s.votingPower(t, d.id, whale.FormattedAddress())
	require.Equal(t, "1000000", vp.Current, "the whale's stake counts in the dao now")
	require.True(t, vp.eligible(t, pid).IsZero(), "and not on the open proposal")
	s.refuse(t, whale, daoCode(1119), "dao", "vote", daoU64(pid), "no")
	require.Equal(t, p.PinnedDenominator, s.prop(t, pid).PinnedDenominator)

	// D2: vote, exit; the ballot is zeroed at once and the coins are locked
	require.Equal(t, "1000", s.voteWeight(t, v1, pid, "yes").String())
	stkBefore := s.bal(t, v1.FormattedAddress(), s.stk)
	out := src.exit(t, v1)
	reduced := s.event(t, out, "EventBallotReduced")
	require.Equal(t, []string{daoU64(pid), src.key(), "1000"}, []string{reduced.Attributes["proposal_id"], reduced.Attributes["source"], reduced.Attributes["amount"]})
	b := s.mustBallot(t, pid, v1.FormattedAddress())
	require.Equal(t, "0", b.Weight)
	require.Empty(t, b.Shares)
	require.Equal(t, "YES", b.Option, "the ballot keeps its option")
	s.requireTallyOf(t, pid, 0, 0, 0)
	require.Equal(t, stkBefore.String(), s.bal(t, v1.FormattedAddress(), s.stk).String(), "the exited stake is locked, not returned")
	s.refuse(t, v1, daoCode(1119), "dao", "vote", daoU64(pid), "no")
	_, found = s.joinRecord(t, src.key(), v1.FormattedAddress())
	require.False(t, found, "an exit forgets the join record")

	// D5: a top-up mid-vote is a new join. The cast ballot keeps its weight,
	// the topped-up stake has no weight on the open proposal, and the
	// pinned denominator does not move
	require.Equal(t, "1000", s.voteWeight(t, v2, pid, "yes").String())
	top := src.join(t, v2, 500)
	h, found = s.joinRecord(t, src.key(), v2.FormattedAddress())
	require.True(t, found, "the top-up is recorded")
	require.Equal(t, top.Height, h)
	require.True(t, s.votingPower(t, d.id, v2.FormattedAddress()).eligible(t, pid).IsZero())
	s.refuse(t, v2, daoCode(1125), "dao", "vote", daoU64(pid), "yes")
	s.refuse(t, v2, daoCode(1119), "dao", "vote", daoU64(pid), "no")
	require.Equal(t, "1000", s.mustBallot(t, pid, v2.FormattedAddress()).Weight)
	s.requireTallyOf(t, pid, 1000, 0, 0)
	require.Equal(t, p.PinnedDenominator, s.prop(t, pid).PinnedDenominator)

	s.waitVotingEnd(t, pid)
	p = s.prop(t, pid)
	status := "REJECTED"
	if passes(t, p) {
		status = "EXECUTED"
	}
	s.settleFirst(t, j, pid, status)
	t.Logf("%s: pinned denominator %s, yes %s: %s", src.key(), p.PinnedDenominator, p.Yes, status)

	// D3 on a later proposal: stake joined before it counts in full
	pid2 := s.submit(t, whale, d, "later", nil)
	require.Equal(t, "1000000", s.votingPower(t, d.id, whale.FormattedAddress()).eligible(t, pid2).String())
	require.Equal(t, "1000000", s.voteWeight(t, whale, pid2, "yes").String())
	require.Equal(t, "1000", s.voteWeight(t, j, pid2, "yes").String())
	s.waitVotingEnd(t, pid2)
	s.settleFirst(t, j, pid2, "EXECUTED")
	for _, w := range []ibc.Wallet{v2, j, whale} {
		src.exit(t, w)
	}
}

// E13. Pinned sources through x/rewards: a finished staking reward a DAO
// uses cannot be deleted (1107); its last exit still succeeds and only skips
// the deletion; once unpinned, the deletion goes through.
func (s *daoSuite) pinnedSources(t *testing.T) {
	admin, staker := s.user(t), s.users(t, 1, daoUserFunds, coins(1_000, s.stk)...)[0]
	sr := s.newSR(t, daoSrLock, s.stk, 1)
	s.joinSR(t, staker, sr, 1_000)
	d := s.createStake(t, admin, defaultCfg(), daoElectorate{SrIDs: []string{sr}}, admin.FormattedAddress())

	// one day tick pays the reward's only day
	cur, err := helpers.CurrentEpoch(s.ctx, s.bze, "day")
	require.NoError(t, err)
	require.NoError(t, helpers.WaitForEpoch(s.ctx, s.bze, "day", cur+1))

	require.EqualValues(t, 1, s.srPayouts(t, sr), "the one-day reward is finished")

	// the last exit of a finished reward the dao pins: the exit succeeds and
	// only the reward's automatic deletion is skipped
	s.exitSR(t, staker, sr)
	require.True(t, s.srExists(t, sr), "a pinned reward is not deleted by its last exit")
	require.True(t, s.srStaked(t, sr).IsZero())
	s.refuse(t, s.funder, daoCode(1107), "rewards", "delete-staking-reward", sr)

	// unpinned, the deletion is retried on the next final exit
	s.adminExec(t, admin, d, nil, daoMsg("MsgUpdateElectorate", map[string]any{"authority": d.addr, "dao_id": d.idArg()}))
	require.Nil(t, s.dao(t, d.id).Electorate)
	again := s.users(t, 1, daoUserFunds, coins(20, s.stk)...)[0]
	s.joinSR(t, again, sr, 10)
	s.exitSR(t, again, sr)
	require.False(t, s.srExists(t, sr), "the deletion went through once nothing pins the reward")

	// and the explicit delete works on an unpinned finished reward
	sr2 := s.newSR(t, daoSrLock, s.stk, 1)
	s.joinSR(t, again, sr2, 5)
	d2 := s.createStake(t, admin, defaultCfg(), daoElectorate{SrIDs: []string{sr2}}, admin.FormattedAddress())
	cur, err = helpers.CurrentEpoch(s.ctx, s.bze, "day")
	require.NoError(t, err)
	require.NoError(t, helpers.WaitForEpoch(s.ctx, s.bze, "day", cur+1))
	s.adminExec(t, admin, d2, nil, daoMsg("MsgUpdateElectorate", map[string]any{"authority": d2.addr, "dao_id": d2.idArg(), "electorate": daoElectorate{SrIDs: []string{s.fixtures(t).srA}}}))
	s.exitSR(t, again, sr2)
	require.False(t, s.srExists(t, sr2), "an unpinned finished reward is deleted by its last exit")
}

func (s *daoSuite) srPayouts(t *testing.T, id string) int64 {
	t.Helper()
	res, err := helpers.Query[struct {
		StakingReward struct {
			Payouts helpers.Int64 `json:"payouts"`
		} `json:"staking_reward"`
	}](s.ctx, s.bze, "rewards", "get-staking-reward", id)
	require.NoError(t, err)
	return int64(res.StakingReward.Payouts)
}
