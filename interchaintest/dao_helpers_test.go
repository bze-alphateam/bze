package interchaintest_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

// ---------------------------------------------------------------------------
// DAO config

// daoCfg is a DaoConfig. Ratios are written as decimals ("0.5") and
// converted to the form each path reads: the CLI (autocli, protojson) wants
// the 18-decimal integer of a LegacyDec, `tx sign` (gogoproto JSON) the
// decimal.
type daoCfg struct {
	quorum, threshold, expQuorum, expThreshold string
	voting, expedited                          string
	allowExpedite, open                        bool
	cost, deposit                              *daoCoin
	policy, costDest, keptDest                 string
}

func defaultCfg() daoCfg {
	return daoCfg{
		quorum: "0.5", threshold: "0.5", expQuorum: "0.6", expThreshold: "0.6",
		voting: daoVoting, expedited: daoExpedited,
		policy: "ALWAYS", costDest: "BURN", keptDest: "BURN",
	}
}

// ratioInt is the 18-decimal integer of a ratio: "0.5" -> 500000000000000000.
func ratioInt(r string) string {
	return sdkmath.LegacyMustNewDecFromStr(r).BigInt().String()
}

func (c daoCfg) fields(ratio func(string) string) map[string]any {
	m := map[string]any{
		"min_quorum":               ratio(c.quorum),
		"threshold":                ratio(c.threshold),
		"voting_period":            c.voting,
		"allow_expedite":           c.allowExpedite,
		"expedited_period":         c.expedited,
		"expedite_quorum":          ratio(c.expQuorum),
		"expedite_threshold":       ratio(c.expThreshold),
		"open_submission":          c.open,
		"deposit_refund_policy":    c.policy,
		"cost_destination":         c.costDest,
		"kept_deposit_destination": c.keptDest,
	}
	if c.cost != nil {
		m["proposal_cost"] = map[string]string{"denom": c.cost.Denom, "amount": c.cost.Amount}
	}
	if c.deposit != nil {
		m["proposal_deposit"] = map[string]string{"denom": c.deposit.Denom, "amount": c.deposit.Amount}
	}
	return m
}

// cli is the config JSON a CLI positional argument (autocli) reads.
func (c daoCfg) cli() string {
	return mustJSON(c.fields(ratioInt))
}

// gogo is the config a message signed with `tx sign` (gogoproto JSON) carries.
func (c daoCfg) gogo() map[string]any {
	return c.fields(func(r string) string { return sdkmath.LegacyMustNewDecFromStr(r).String() })
}

// coinPtr is a config coin; it may be invalid (a negative amount, a bad
// denom), which sdk.Coin can not hold.
func coinPtr(amount int64, denom string) *daoCoin {
	return &daoCoin{Denom: denom, Amount: strconv.FormatInt(amount, 10)}
}

func mustJSON(v any) string {
	bz, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(bz)
}

// ---------------------------------------------------------------------------
// records, as both the CLI (protojson, ratios as 18-decimal integers, zero
// values omitted) and REST (gogoproto JSON, ratios as decimals) print them

type daoRef struct {
	id   uint64
	addr string
}

func (d daoRef) idArg() string { return strconv.FormatUint(d.id, 10) }

// daoAddress is the account address of a DAO: address.Module("dao", id).
func daoAddress(id uint64) string {
	return sdk.MustBech32ifyAddressBytes(chain.Bech32Prefix, address.Module(daoModule, sdk.Uint64ToBigEndian(id)))
}

type daoCoin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

func (c *daoCoin) coins() sdk.Coins {
	if c == nil || c.Denom == "" {
		return sdk.NewCoins()
	}
	return sdk.NewCoins(sdk.NewCoin(c.Denom, parseInt(c.Amount)))
}

type daoCfgRec struct {
	MinQuorum              string   `json:"min_quorum"`
	Threshold              string   `json:"threshold"`
	VotingPeriod           string   `json:"voting_period"`
	AllowExpedite          bool     `json:"allow_expedite"`
	ExpeditedPeriod        string   `json:"expedited_period"`
	ExpediteQuorum         string   `json:"expedite_quorum"`
	ExpediteThreshold      string   `json:"expedite_threshold"`
	OpenSubmission         bool     `json:"open_submission"`
	ProposalCost           *daoCoin `json:"proposal_cost"`
	ProposalDeposit        *daoCoin `json:"proposal_deposit"`
	DepositRefundPolicy    string   `json:"deposit_refund_policy"`
	CostDestination        string   `json:"cost_destination"`
	KeptDepositDestination string   `json:"kept_deposit_destination"`
}

type daoElectorate struct {
	SrIDs   []string `json:"sr_ids"`
	DrDenom string   `json:"dr_denom"`
}

type daoRec struct {
	ID           helpers.Int64  `json:"id"`
	Address      string         `json:"address"`
	Admin        string         `json:"admin"`
	PendingAdmin string         `json:"pending_admin"`
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Config       daoCfgRec      `json:"config"`
	Members      []string       `json:"members"`
	Electorate   *daoElectorate `json:"electorate"`
	CreatedAt    helpers.Int64  `json:"created_at"`
}

func (d daoRec) daoType() string { return enumOr(d.Type, "DAO_TYPE_STATIC_LIST") }

type daoShare struct {
	Source string `json:"source"`
	Amount string `json:"amount"`
}

type daoBallot struct {
	ProposalID helpers.Int64 `json:"proposal_id"`
	Voter      string        `json:"voter"`
	Option     string        `json:"option"`
	Weight     string        `json:"weight"`
	Shares     []daoShare    `json:"shares"`
}

type daoProposal struct {
	ID                helpers.Int64     `json:"id"`
	DaoID             helpers.Int64     `json:"dao_id"`
	Proposer          string            `json:"proposer"`
	Title             string            `json:"title"`
	Summary           string            `json:"summary"`
	Msgs              []json.RawMessage `json:"msgs"`
	Expedited         bool              `json:"expedited"`
	Status            string            `json:"status"`
	SubmitHeight      helpers.Int64     `json:"submit_height"`
	SubmitTime        helpers.Int64     `json:"submit_time"`
	VotingEnd         helpers.Int64     `json:"voting_end"`
	PinnedConfig      daoCfgRec         `json:"pinned_config"`
	PinnedElectorate  *daoElectorate    `json:"pinned_electorate"`
	PinnedDenominator string            `json:"pinned_denominator"`
	PinnedDeposit     *daoCoin          `json:"pinned_deposit"`
	Yes               string            `json:"yes"`
	No                string            `json:"no"`
	Abstain           string            `json:"abstain"`
	CleanupCursor     string            `json:"cleanup_cursor"`
}

func (p daoProposal) status() string { return enumOr(p.Status, "VOTING") }

type daoProposalResp struct {
	Proposal           daoProposal   `json:"proposal"`
	VotingEnded        bool          `json:"voting_ended"`
	RetryDeadline      helpers.Int64 `json:"retry_deadline"`
	ExpediteSettleable bool          `json:"expedite_settleable"`
}

type daoVotingPower struct {
	Current   string `json:"current"`
	Proposals []struct {
		ProposalID     helpers.Int64 `json:"proposal_id"`
		EligibleWeight string        `json:"eligible_weight"`
		Shares         []daoShare    `json:"shares"`
	} `json:"proposals"`
}

// eligible returns the weight a vote on the proposal would count now.
func (v daoVotingPower) eligible(t *testing.T, pid uint64) sdkmath.Int {
	t.Helper()
	for _, p := range v.Proposals {
		if uint64(p.ProposalID) == pid {
			return parseInt(p.EligibleWeight)
		}
	}
	t.Fatalf("proposal %d is not in the voting power answer %+v", pid, v)
	return sdkmath.Int{}
}

type daoController struct {
	Controller string   `json:"controller"`
	EndReason  string   `json:"end_reason"`
	Path       []string `json:"path"`
}

type daoPage struct {
	NextKey string `json:"next_key"`
	Total   string `json:"total"`
}

type daoParams struct {
	MaxSrPerDao               helpers.Int64 `json:"max_sr_per_dao"`
	MaxStaticMembers          helpers.Int64 `json:"max_static_members"`
	MaxOpenProposals          helpers.Int64 `json:"max_open_proposals"`
	MaxMsgsPerProposal        helpers.Int64 `json:"max_msgs_per_proposal"`
	MaxProposalBytes          helpers.Int64 `json:"max_proposal_bytes"`
	MaxTitleLen               helpers.Int64 `json:"max_title_len"`
	MaxSummaryLen             helpers.Int64 `json:"max_summary_len"`
	MinVotingPeriod           string        `json:"min_voting_period"`
	MaxVotingPeriod           string        `json:"max_voting_period"`
	ExecutionWindow           string        `json:"execution_window"`
	CleanupBatchSize          helpers.Int64 `json:"cleanup_batch_size"`
	DaoCreationFee            sdk.Coin      `json:"dao_creation_fee"`
	DaoCreationFeeDestination string        `json:"dao_creation_fee_destination"`
	MinElectorateLockDays     helpers.Int64 `json:"min_electorate_lock_days"`
}

// enumOr reads an enum protojson omitted because it is the zero value.
func enumOr(v, zero string) string {
	if v == "" {
		return zero
	}
	return v
}

// ratio reads a LegacyDec printed as an 18-decimal integer (CLI) or as a
// decimal (REST).
func ratio(t *testing.T, v string) sdkmath.LegacyDec {
	t.Helper()
	if strings.Contains(v, ".") {
		d, err := sdkmath.LegacyNewDecFromStr(v)
		require.NoError(t, err)
		return d
	}
	return sdkmath.LegacyNewDecFromBigIntWithPrec(parseInt(v).BigInt(), sdkmath.LegacyPrecision)
}

// ---------------------------------------------------------------------------
// REST reads (the gRPC gateway of the node, reached from the host). Cases read
// state through REST because it costs no CLI process; section B runs every
// query through the CLI too.

// restGet decodes the gateway's JSON answer at path into out. It returns the
// HTTP status; a non-200 answer leaves out untouched.
func (s *daoSuite) restGet(t *testing.T, path string, out any) int {
	t.Helper()
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		res, err := http.Get(s.api + path) //nolint:noctx // a bounded local call
		if err != nil {
			lastErr = err
			time.Sleep(300 * time.Millisecond)
			continue
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		require.NoError(t, err)
		if res.StatusCode >= 500 && res.StatusCode != http.StatusNotImplemented {
			// the gateway answers NotFound/InvalidArgument with 404/400; a 5xx
			// may be a node still committing, try again
			lastErr = fmt.Errorf("GET %s: %d %s", path, res.StatusCode, body)
			time.Sleep(300 * time.Millisecond)
			continue
		}
		if res.StatusCode == http.StatusOK && out != nil {
			require.NoError(t, json.Unmarshal(body, out), "GET %s: %s", path, body)
		}
		return res.StatusCode
	}
	t.Fatalf("GET %s: %v", path, lastErr)
	return 0
}

func (s *daoSuite) mustRest(t *testing.T, path string, out any) {
	t.Helper()
	require.Equal(t, http.StatusOK, s.restGet(t, path, out), "GET %s", path)
}

func (s *daoSuite) dao(t *testing.T, id uint64) daoRec {
	t.Helper()
	var res struct {
		Dao daoRec `json:"dao"`
	}
	s.mustRest(t, fmt.Sprintf("/bze/dao/dao/%d", id), &res)
	return res.Dao
}

func (s *daoSuite) proposal(t *testing.T, id uint64) daoProposalResp {
	t.Helper()
	var res daoProposalResp
	s.mustRest(t, fmt.Sprintf("/bze/dao/proposal/%d", id), &res)
	return res
}

func (s *daoSuite) prop(t *testing.T, id uint64) daoProposal {
	t.Helper()
	return s.proposal(t, id).Proposal
}

func (s *daoSuite) proposalExists(t *testing.T, id uint64) bool {
	t.Helper()
	code := s.restGet(t, fmt.Sprintf("/bze/dao/proposal/%d", id), nil)
	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, code)
	return code == http.StatusOK
}

func (s *daoSuite) ballot(t *testing.T, pid uint64, voter string) (daoBallot, bool) {
	t.Helper()
	var res struct {
		Ballot daoBallot `json:"ballot"`
	}
	code := s.restGet(t, fmt.Sprintf("/bze/dao/ballot/%d/%s", pid, voter), &res)
	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, code)
	return res.Ballot, code == http.StatusOK
}

func (s *daoSuite) mustBallot(t *testing.T, pid uint64, voter string) daoBallot {
	t.Helper()
	b, ok := s.ballot(t, pid, voter)
	require.True(t, ok, "no ballot of %s on proposal %d", voter, pid)
	return b
}

// paged walks a paginated REST listing; items decodes one page and returns
// its next_key.
func (s *daoSuite) paged(t *testing.T, path string, items func([]byte) string) {
	t.Helper()
	key := ""
	for {
		p := path
		sep := "?"
		if strings.Contains(p, "?") {
			sep = "&"
		}
		p += sep + "pagination.limit=200"
		if key != "" {
			p += "&pagination.key=" + url.QueryEscape(key)
		}
		var raw json.RawMessage
		s.mustRest(t, p, &raw)
		key = items(raw)
		if key == "" {
			return
		}
	}
}

func (s *daoSuite) ballots(t *testing.T, pid uint64) []daoBallot {
	t.Helper()
	var all []daoBallot
	s.paged(t, fmt.Sprintf("/bze/dao/ballots/%d", pid), func(raw []byte) string {
		var page struct {
			Ballots    []daoBallot `json:"ballots"`
			Pagination daoPage     `json:"pagination"`
		}
		require.NoError(t, json.Unmarshal(raw, &page))
		all = append(all, page.Ballots...)
		return page.Pagination.NextKey
	})
	return all
}

func (s *daoSuite) allDaos(t *testing.T) []daoRec {
	t.Helper()
	var all []daoRec
	s.paged(t, "/bze/dao/daos", func(raw []byte) string {
		var page struct {
			Daos       []daoRec `json:"daos"`
			Pagination daoPage  `json:"pagination"`
		}
		require.NoError(t, json.Unmarshal(raw, &page))
		all = append(all, page.Daos...)
		return page.Pagination.NextKey
	})
	return all
}

func (s *daoSuite) daosByAdmin(t *testing.T, admin string) []daoRec {
	t.Helper()
	var all []daoRec
	s.paged(t, "/bze/dao/daos_by_admin/"+admin, func(raw []byte) string {
		var page struct {
			Daos       []daoRec `json:"daos"`
			Pagination daoPage  `json:"pagination"`
		}
		require.NoError(t, json.Unmarshal(raw, &page))
		all = append(all, page.Daos...)
		return page.Pagination.NextKey
	})
	return all
}

// daoProposals lists a DAO's proposals, optionally only those with the given
// stored statuses (VOTING, REJECTED, …).
func (s *daoSuite) daoProposals(t *testing.T, daoID uint64, statuses ...string) []daoProposal {
	t.Helper()
	path := fmt.Sprintf("/bze/dao/proposals/%d", daoID)
	for i, st := range statuses {
		sep := "&"
		if i == 0 {
			sep = "?"
		}
		path += sep + "statuses=" + st
	}
	var all []daoProposal
	s.paged(t, path, func(raw []byte) string {
		var page struct {
			Proposals  []daoProposal `json:"proposals"`
			Pagination daoPage       `json:"pagination"`
		}
		require.NoError(t, json.Unmarshal(raw, &page))
		all = append(all, page.Proposals...)
		return page.Pagination.NextKey
	})
	return all
}

func (s *daoSuite) votingPower(t *testing.T, daoID uint64, addr string) daoVotingPower {
	t.Helper()
	var res daoVotingPower
	s.mustRest(t, fmt.Sprintf("/bze/dao/voting_power/%d/%s", daoID, addr), &res)
	return res
}

// joinRecord returns the recorded join height of addr on source, if any.
func (s *daoSuite) joinRecord(t *testing.T, source, addr string) (int64, bool) {
	t.Helper()
	var res struct {
		Height helpers.Int64 `json:"height"`
	}
	code := s.restGet(t, "/bze/dao/join_record?source="+url.QueryEscape(source)+"&address="+addr, &res)
	require.Contains(t, []int{http.StatusOK, http.StatusNotFound}, code)
	return int64(res.Height), code == http.StatusOK
}

func (s *daoSuite) controller(t *testing.T, id uint64) daoController {
	t.Helper()
	var res daoController
	s.mustRest(t, fmt.Sprintf("/bze/dao/dao_controller/%d", id), &res)
	return res
}

func (s *daoSuite) queryParams(t *testing.T) daoParams {
	t.Helper()
	var res struct {
		Params daoParams `json:"params"`
	}
	s.mustRest(t, "/bze/dao/params", &res)
	return res.Params
}

func (s *daoSuite) allBalances(t *testing.T, addr string) sdk.Coins {
	t.Helper()
	var res struct {
		Balances []daoCoin `json:"balances"`
	}
	s.mustRest(t, "/cosmos/bank/v1beta1/balances/"+addr+"?pagination.limit=1000", &res)
	out := sdk.NewCoins()
	for _, c := range res.Balances {
		out = out.Add(sdk.NewCoin(c.Denom, parseInt(c.Amount)))
	}
	return out
}

func (s *daoSuite) bal(t *testing.T, addr, denom string) sdkmath.Int {
	t.Helper()
	return s.allBalances(t, addr).AmountOf(denom)
}

// account returns the @type and account number of the account at addr.
func (s *daoSuite) account(t *testing.T, addr string) (string, int64, bool) {
	t.Helper()
	var res struct {
		Account map[string]json.RawMessage `json:"account"`
	}
	code := s.restGet(t, "/cosmos/auth/v1beta1/accounts/"+addr, &res)
	if code == http.StatusNotFound {
		return "", 0, false
	}
	require.Equal(t, http.StatusOK, code)
	var typ string
	require.NoError(t, json.Unmarshal(res.Account["@type"], &typ))
	number := func(raw json.RawMessage) int64 {
		var n helpers.Int64
		require.NoError(t, json.Unmarshal(raw, &n))
		return int64(n)
	}
	if raw, ok := res.Account["account_number"]; ok {
		return typ, number(raw), true
	}
	// a vesting account nests its base account
	var bva struct {
		BaseAccount struct {
			AccountNumber helpers.Int64 `json:"account_number"`
		} `json:"base_account"`
	}
	require.NoError(t, json.Unmarshal(res.Account["base_vesting_account"], &bva))
	return typ, int64(bva.BaseAccount.AccountNumber), true
}

// blockTime returns the latest block's height and time (unix seconds).
func (s *daoSuite) blockTime(t *testing.T) (int64, int64) {
	t.Helper()
	st, err := s.bze.GetNode().Client.Status(s.ctx)
	require.NoError(t, err)
	return st.SyncInfo.LatestBlockHeight, st.SyncInfo.LatestBlockTime.Unix()
}

// waitTime waits until the latest block's time is at least unix: every tx
// sent after it lands in a block at or after unix.
func (s *daoSuite) waitTime(t *testing.T, unix int64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Minute)
	for {
		_, now := s.blockTime(t)
		if now >= unix {
			return
		}
		require.True(t, time.Now().Before(deadline), "block time did not reach %d (at %d)", unix, now)
		time.Sleep(300 * time.Millisecond)
	}
}

// waitVotingEnd waits until the proposal's voting has ended.
func (s *daoSuite) waitVotingEnd(t *testing.T, pid uint64) {
	t.Helper()
	s.waitTime(t, int64(s.prop(t, pid).VotingEnd))
}

// ---------------------------------------------------------------------------
// txs

func (s *daoSuite) mustExec(t *testing.T, user ibc.Wallet, args ...string) helpers.TxResult {
	t.Helper()
	return s.mustExecWith(t, user, nil, args...)
}

func (s *daoSuite) mustExecWith(t *testing.T, user ibc.Wallet, opts []helpers.TxOption, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(s.ctx, s.bze, user, opts, args...)
	require.NoError(t, err, "tx %v", args)
	require.Zero(t, res.Code, "tx %v failed: %s", args, res.RawLog)
	return res
}

func (s *daoSuite) mustExecMsgs(t *testing.T, user ibc.Wallet, gas uint64, msgs ...any) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(gas)}, msgs...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx of %d msgs failed: %s", len(msgs), res.RawLog)
	return res
}

// daoFlow is one bank transfer a tx is expected to make.
type daoFlow struct {
	from, to string
	amt      sdk.Coins
}

func flow(from, to string, amt sdk.Coins) daoFlow { return daoFlow{from: from, to: to, amt: amt} }

// audit is the funds audit of one included tx. The tx's bank transfer events
// must be exactly the fee payment plus want (in any order), and every
// account in watch (plus the fee payer, the parties of want that are not
// module collectors, and the escrow) must have changed by exactly what those
// transfers say. Transfers in ignore denoms (prize claims of x/rewards) are
// left out of both checks.
type daoAudit struct {
	s      *daoSuite
	watch  []string
	before map[string]sdk.Coins
	ignore []string
}

func (s *daoSuite) startAudit(t *testing.T, watch ...string) *daoAudit {
	t.Helper()
	a := &daoAudit{s: s, before: map[string]sdk.Coins{}, ignore: []string{s.prz}}
	for _, w := range append(watch, s.escrow) {
		if w == "" || slices.Contains(a.watch, w) {
			continue
		}
		a.watch = append(a.watch, w)
		a.before[w] = s.allBalances(t, w)
	}
	return a
}

func (a *daoAudit) strip(c sdk.Coins) sdk.Coins {
	out := sdk.NewCoins()
	for _, coin := range c {
		if !slices.Contains(a.ignore, coin.Denom) {
			out = out.Add(coin)
		}
	}
	return out
}

func (a *daoAudit) check(t *testing.T, res helpers.TxResult, want ...daoFlow) {
	t.Helper()
	s := a.s
	payer, fee := s.feeOf(t, res)

	type tr struct{ from, to, amt string }
	var got []tr
	feeSeen := false
	for _, e := range helpers.TxEvents(res, "transfer") {
		amt, err := sdk.ParseCoinsNormalized(e.Attributes["amount"])
		require.NoError(t, err)
		if !feeSeen && e.Attributes["recipient"] == s.feeCollector && e.Attributes["sender"] == payer && amt.Equal(fee) {
			feeSeen = true
			continue
		}
		if amt = a.strip(amt); amt.IsZero() {
			continue
		}
		got = append(got, tr{e.Attributes["sender"], e.Attributes["recipient"], amt.String()})
	}
	require.True(t, fee.IsZero() || feeSeen, "tx %s: no fee transfer of %s from %s", res.TxHash, fee, payer)
	var exp []tr
	for _, f := range want {
		if amt := a.strip(f.amt); !amt.IsZero() {
			exp = append(exp, tr{f.from, f.to, amt.String()})
		}
	}
	key := func(x tr) string { return x.from + ">" + x.to + ":" + x.amt }
	sort.Slice(got, func(i, j int) bool { return key(got[i]) < key(got[j]) })
	sort.Slice(exp, func(i, j int) bool { return key(exp[i]) < key(exp[j]) })
	require.Equal(t, exp, got, "tx %s: its bank transfers must be exactly the fee and the intended moves", res.TxHash)

	for _, w := range a.watch {
		delta := map[string]sdkmath.Int{}
		add := func(c sdk.Coins, sign int64) {
			for _, coin := range a.strip(c) {
				d, ok := delta[coin.Denom]
				if !ok {
					d = sdkmath.ZeroInt()
				}
				delta[coin.Denom] = d.Add(coin.Amount.MulRaw(sign))
			}
		}
		for _, f := range want {
			if f.to == w {
				add(f.amt, 1)
			}
			if f.from == w {
				add(f.amt, -1)
			}
		}
		if w == payer {
			add(fee, -1)
		}
		after := s.allBalances(t, w)
		for _, denom := range unionDenoms(a.before[w], after, delta) {
			if slices.Contains(a.ignore, denom) {
				continue
			}
			want := a.before[w].AmountOf(denom)
			if d, ok := delta[denom]; ok {
				want = want.Add(d)
			}
			require.Equal(t, want.String(), after.AmountOf(denom).String(), "tx %s: balance of %s in %s", res.TxHash, w, denom)
		}
	}
}

func unionDenoms(a, b sdk.Coins, d map[string]sdkmath.Int) []string {
	var out []string
	for _, c := range append(a, b...) {
		if !slices.Contains(out, c.Denom) {
			out = append(out, c.Denom)
		}
	}
	for denom := range d {
		if !slices.Contains(out, denom) {
			out = append(out, denom)
		}
	}
	return out
}

// feeOf returns who paid the tx fee and how much (the tx event's fee and
// fee_payer).
func (s *daoSuite) feeOf(t *testing.T, res helpers.TxResult) (string, sdk.Coins) {
	t.Helper()
	for _, e := range helpers.TxEvents(res, "tx") {
		v, ok := e.Attributes["fee"]
		if !ok {
			continue
		}
		fee, err := sdk.ParseCoinsNormalized(v)
		require.NoError(t, err)
		return e.Attributes["fee_payer"], fee
	}
	t.Fatalf("tx %s has no fee attribute", res.TxHash)
	return "", nil
}

// run sends `bzed tx <args>` signed by user and requires it to succeed, with
// the funds audit: want are the transfers the tx must make, watch the extra
// accounts whose balances must follow them.
func (s *daoSuite) run(t *testing.T, user ibc.Wallet, watch []string, want []daoFlow, args ...string) helpers.TxResult {
	t.Helper()
	return s.runWith(t, user, nil, watch, want, args...)
}

func (s *daoSuite) runWith(t *testing.T, user ibc.Wallet, opts []helpers.TxOption, watch []string, want []daoFlow, args ...string) helpers.TxResult {
	t.Helper()
	a := s.startAudit(t, s.auditParties(user, watch, want)...)
	res := s.mustExecWith(t, user, opts, args...)
	a.check(t, res, want...)
	return res
}

// runMsgs is run for a tx of raw messages (gogoproto JSON), signed with `tx sign`.
func (s *daoSuite) runMsgs(t *testing.T, user ibc.Wallet, gas uint64, watch []string, want []daoFlow, msgs ...any) helpers.TxResult {
	t.Helper()
	a := s.startAudit(t, s.auditParties(user, watch, want)...)
	res := s.mustExecMsgs(t, user, gas, msgs...)
	a.check(t, res, want...)
	return res
}

func (s *daoSuite) auditParties(user ibc.Wallet, watch []string, want []daoFlow) []string {
	parties := append([]string{user.FormattedAddress()}, watch...)
	modules := []string{s.burner, s.cp, s.feeCollector, s.rewards}
	for _, f := range want {
		for _, a := range []string{f.from, f.to} {
			if !slices.Contains(modules, a) {
				parties = append(parties, a)
			}
		}
	}
	return parties
}

// refusal is what a refused tx must leave behind: the chain's dao state and
// the balances of the escrow and of every watched account, except the fee.
// A zero code accepts any code of the codespace.
type refusal struct {
	codespace string
	code      uint32
	reason    string
	watch     []string
}

func daoCode(code uint32) refusal { return refusal{codespace: daoModule, code: code} }

// daoErrText is the registered message of each dao error (x/dao/types/errors.go).
var daoErrText = map[uint32]string{
	1100: "expected gov account as only signer for proposal message",
	1101: "invalid dao type",
	1102: "invalid dao metadata",
	1103: "invalid dao config",
	1104: "invalid dao members",
	1105: "invalid dao electorate",
	1106: "an account already exists at the dao address",
	1107: "the reward is used by a dao electorate",
	1108: "dao not found",
	1109: "signer is not the dao's admin",
	1110: "invalid dao messages",
	1111: "message must be signed by the dao account only",
	1112: "an admin nomination is already pending",
	1113: "the dao electorate cannot vote",
	1114: "invalid admin nominee",
	1115: "no admin nomination is pending",
	1116: "signer is not the dao's pending admin",
	1117: "invalid proposal",
	1118: "signer cannot submit proposals to this dao",
	1119: "no voting power",
	1120: "the dao has too many open proposals",
	1121: "the dao does not allow expedited proposals",
	1122: "invalid vote",
	1123: "proposal not found",
	1124: "the proposal is not open for voting",
	1125: "the ballot already has this option",
	1126: "signer is not the proposal's proposer",
	1127: "the proposal's voting period has not ended",
	1128: "the proposal is finished",
	1129: "a dao message panicked",
	1130: "the expedited proposal does not meet the expedite bars yet",
	1131: "the proposal is not finished",
}

// refuse sends `bzed tx <args>` with a fixed gas limit (no simulation, so the
// tx is included) and requires the refusal: its code, no dao state change,
// and no fund movement but the fee.
func (s *daoSuite) refuse(t *testing.T, user ibc.Wallet, r refusal, args ...string) helpers.TxResult {
	t.Helper()
	return s.refuseSend(t, user, r, args, func() (helpers.TxResult, error) {
		return helpers.ExecTxWith(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(daoRefuseGas)}, args...)
	})
}

// refuseMsgs is refuse for raw messages signed with `tx sign`, which skips
// the CLI's own ValidateBasic: a stateless refusal then comes from CheckTx
// (never included, nothing charged).
func (s *daoSuite) refuseMsgs(t *testing.T, user ibc.Wallet, r refusal, msgs ...any) helpers.TxResult {
	t.Helper()
	return s.refuseSend(t, user, r, msgs, func() (helpers.TxResult, error) {
		return helpers.ExecMsgs(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(daoRefuseGas)}, msgs...)
	})
}

func (s *daoSuite) refuseSend(t *testing.T, user ibc.Wallet, r refusal, what any, send func() (helpers.TxResult, error)) helpers.TxResult {
	t.Helper()
	a := s.startAudit(t, append([]string{user.FormattedAddress()}, r.watch...)...)
	digest := s.stateDigest(t)
	res, err := send()
	if err != nil {
		// the CLI ran the message's ValidateBasic itself and broadcast nothing
		text, known := daoErrText[r.code]
		require.True(t, r.codespace == daoModule && known, "tx %v: %v", what, err)
		require.Contains(t, err.Error(), text, "tx %v", what)
		for _, w := range a.watch {
			require.Equal(t, a.before[w].String(), s.allBalances(t, w).String(), "a tx the CLI refused moves nothing")
		}
		require.Equal(t, digest, s.stateDigest(t))
		return res
	}
	require.NotZero(t, res.Code, "tx %v must be refused, it was accepted at height %d", what, res.Height)
	require.Equal(t, r.codespace, res.Codespace, "tx %v: %s", what, res.RawLog)
	if r.code != 0 {
		require.Equal(t, r.code, res.Code, "tx %v: %s", what, res.RawLog)
	}
	if r.reason != "" {
		require.Contains(t, res.RawLog, r.reason)
	}
	if res.Height > 0 {
		a.check(t, res)
	} else {
		for _, w := range a.watch {
			require.Equal(t, a.before[w].String(), s.allBalances(t, w).String(), "a tx refused in CheckTx moves nothing")
		}
	}
	require.Equal(t, digest, s.stateDigest(t), "a refused tx must not change the dao state")
	return res
}

// stateDigest is every DAO with its proposals, as REST prints them: a refused
// tx must leave it as it was.
func (s *daoSuite) stateDigest(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	for _, d := range s.allDaos(t) {
		b.WriteString(mustJSON(d))
		for _, p := range s.daoProposals(t, uint64(d.ID)) {
			b.WriteString(mustJSON(p))
		}
	}
	return b.String()
}

// events returns the tx's typed dao events of a kind (EventDaoCreated, …).
func daoEvents(res helpers.TxResult, name string) []helpers.Event {
	return helpers.TxEvents(res, daoEventPrefix+name)
}

func (s *daoSuite) event(t *testing.T, res helpers.TxResult, name string) helpers.Event {
	t.Helper()
	events := daoEvents(res, name)
	require.Len(t, events, 1, "want one %s in tx %s, got %+v", name, res.TxHash, events)
	return events[0]
}

func atoiU(t *testing.T, v string) uint64 {
	t.Helper()
	n, err := strconv.ParseUint(v, 10, 64)
	require.NoError(t, err, "parse %q", v)
	return n
}

// ---------------------------------------------------------------------------
// messages, as JSON Anys (the CLI and `tx sign` read the same shape for these)

func bankSend(from, to string, amt sdk.Coins) map[string]any {
	return map[string]any{"@type": "/cosmos.bank.v1beta1.MsgSend", "from_address": from, "to_address": to, "amount": amt}
}

func daoMsg(typ string, fields map[string]any) map[string]any {
	m := map[string]any{"@type": "/bze.dao." + typ}
	for k, v := range fields {
		m[k] = v
	}
	return m
}

func msgArgs(msgs ...map[string]any) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = mustJSON(m)
	}
	return out
}

func addrs(ws ...ibc.Wallet) []string {
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = w.FormattedAddress()
	}
	return out
}

func srSource(id string) string       { return "sr/" + id }
func drSource(denom string) string    { return "dr/" + denom }
func ubze(amount int64) sdk.Coins     { return coins(amount, chain.Denom) }
func daoU64(v uint64) string          { return strconv.FormatUint(v, 10) }
func idOf(e helpers.Event) string     { return e.Attributes["id"] }
func statusOf(e helpers.Event) string { return e.Attributes["status"] }
