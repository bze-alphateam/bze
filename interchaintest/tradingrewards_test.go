package interchaintest_test

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"testing"
	"time"

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

// Trading rewards pay their prize per slot to the top traders of one market
// (an order-book market or a liquidity pool, by id) once their duration has
// passed. Durations are in days of 24 hour epochs: 2 days are 48 hour epochs,
// about 4–5 minutes with the fast epochs. Every prize is paid in a factory
// denom of its own, so winner payouts and the rewards module escrow are
// asserted on that denom alone.
const (
	trCreatorFunds = 1_000_000_000_000 // 1,000,000 BZE: two denoms, a market, a pool, four rewards, proposals
	trUserFunds    = 10_000_000_000    // 10,000 BZE each
	trMint         = 100_000_000_000
	trMakerBase    = 100_000_000 // tr sent to the maker

	// the tr/ubze pool, created with the AMM suite's fee and fee destination
	trPoolBase  = 10_000_000_000
	trPoolQuote = 20_000_000_000
	trSwapBze   = 100_000_000 // ubze t2 swaps for tr
	trSwapBase  = 1_000       // tr t2 swaps back

	// the maker's resting orders: every taker buy fills the sell, t1's sell
	// fills the buy
	trSellPrice  = "2"
	trBuyPrice   = "1"
	trMakerSell  = 10_000
	trMakerBuy   = 1_000
	trPendingBuy = 100 // a trade while the rewards are still pending
	trFewerBuy   = 200 // t1, the only trader on reward D

	trDuration       = 2 // days
	trPendingDays    = 30
	trDisplayEpochs  = 7 * 24 // a paid reward stays queryable one more week
	trTxGas          = 500_000
	rewardsCodespace = "rewards"

	// how long the suite waits for the payout and for the cleanup
	trPayoutWait  = 10 * time.Minute
	trCleanupWait = 25 * time.Minute
)

// the rewards of the suite: prize per slot and slots
var (
	trRewardA = trSpec{prize: 1_000_000, slots: 2} // the tr/ubze market
	trRewardB = trSpec{prize: 700_000, slots: 1}   // the tr_ubze pool
	trRewardD = trSpec{prize: 300_000, slots: 3}   // a second pending reward on the market, activated once A has paid out
	trRewardE = trSpec{prize: 400_000, slots: 1}   // created on the market once A has paid out, never activated
	trRewardF = trSpec{prize: 500_000, slots: 2}   // created on the pool once B has paid out, activated, never traded
)

// TestTradingRewards runs the trading rewards of the rewards module end to
// end: create (escrow, fee, pending record), the create refusals, activation
// by a real gov proposal and its refusals, the tradebin hooks that credit
// order-book takers and AMM swappers, the countdown that starts at the first
// trade, the leaderboard (order, ties, trim to the slots), the payout at
// expiry, the display week and the cleanup after it, the burn of the slots
// nobody filled and a reward nobody trades on waiting for its first trade.
// Every step asserts the exact balances of the parties, the prize paid to
// each winner and the rewards module escrow. Subtests run in order and share
// state; the cleanup step waits a week of hour epochs (about 15 minutes).
func TestTradingRewards(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()
	net := chain.Start(t, ctx)
	s := setupTradingRewards(t, ctx, net.Chain)

	t.Run("1 create", s.create)
	t.Run("2 create refusals", s.createRefusals)
	t.Run("3 activate by gov", s.activate)
	t.Run("4 hooks credit takers and swappers", s.hooks)
	t.Run("5 trim to slots", s.trim)
	t.Run("6 payout at expiry", s.payout)
	t.Run("7 cleanup after the display week", s.cleanup)
	t.Run("8 fewer traders than slots", s.fewerTraders)
	t.Run("9 no trades at all", s.noTrades)
	t.Run("10 escrow conservation", s.conservation)
}

type trSpec struct {
	prize, slots int64
}

func (r trSpec) escrow() int64 { return r.prize * r.slots }

type tradingRewardsSuite struct {
	// the AMM suite's query and tx helpers (balances, events, pools, params)
	*ammSuite

	creator, t1, t2, t3, maker ibc.Wallet
	tr, prize                  string
	trMarket, trPool           string
	gov, rewards               string
	createFee                  sdk.Coin
	baseline                   sdk.Coins // rewards module before the first trading reward

	rewardA, rewardB, rewardD, rewardE, rewardF string
	live                                        map[string]trSpec // rewards whose escrow the module still holds
	expireAtA, expireAtB                        int64             // set by the first trade on each reward
	activatedDF                                 int64             // hour epoch of the proposal that activated D and F
	swapped                                     int64             // tr t2 swapped on the pool: reward B's credit
}

func setupTradingRewards(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain) *tradingRewardsSuite {
	s := &tradingRewardsSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}, live: map[string]trSpec{}}

	s.creator = interchaintest.GetAndFundTestUsers(t, ctx, "tr-creator", sdkmath.NewInt(trCreatorFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "tr", sdkmath.NewInt(trUserFunds), bze, bze, bze, bze)
	s.t1, s.t2, s.t3, s.maker = users[0], users[1], users[2], users[3]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	var err error
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)
	for name, addr := range map[string]*string{
		"tradebin":              &s.tradebin,
		"burner":                &s.burner,
		"txfeecollector_burner": &s.burnerFees,
		"txfeecollector_cp":     &s.cpFees,
		"rewards":               &s.rewards,
	} {
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	s.params = s.queryParams(t)
	s.createFee = s.tradingRewardFee(t)

	factory := "factory/" + s.creator.FormattedAddress() + "/"
	s.tr, s.prize = factory+"tr", factory+"prize"
	for _, sub := range []string{"tr", "prize"} {
		s.mustExec(t, s.creator, "tokenfactory", "create-denom", sub)
		s.mustExec(t, s.creator, "tokenfactory", "mint", coinStr(trMint, factory+sub))
	}
	s.mustExec(t, s.creator, "bank", "send", s.creator.FormattedAddress(), s.maker.FormattedAddress(), coinStr(trMakerBase, s.tr))

	// pool ids order the denoms alphabetically: "factory/…" sorts before "ubze"
	s.trMarket = s.tr + "/" + chain.Denom
	s.trPool = s.tr + "_" + chain.Denom
	gas := []helpers.TxOption{helpers.WithGas(trTxGas)}
	s.mustExecWith(t, s.creator, gas, "tradebin", "create-market", s.tr, chain.Denom)
	s.mustExecWith(t, s.creator, gas, "tradebin", "create-liquidity-pool", s.tr, chain.Denom, ammPoolFee, ammFeeDest, "false",
		strconv.Itoa(trPoolBase), strconv.Itoa(trPoolQuote))
	s.mustExec(t, s.maker, "tradebin", "create-order", "sell", strconv.Itoa(trMakerSell), trSellPrice, s.trMarket)
	s.mustExec(t, s.maker, "tradebin", "create-order", "buy", strconv.Itoa(trMakerBuy), trBuyPrice, s.trMarket)

	s.baseline = s.moduleBalances(t)
	return s
}

// 1. The creator escrows prize × slots in the rewards module and pays the
// create fee, which goes through the rewards module to the community-pool
// fee collector. The reward is pending, expiring 30 days of hour epochs after
// the block that created it. A second pending reward on the same market is
// allowed (only an active one blocks the market), and trades on a market
// whose rewards are pending credit nobody.
func (s *tradingRewardsSuite) create(t *testing.T) {
	s.rewardA = s.createReward(t, trRewardA, s.trMarket)
	s.rewardB = s.createReward(t, trRewardB, s.trPool)
	s.rewardD = s.createReward(t, trRewardD, s.trMarket)
	s.requireEscrow(t)

	s.trade(t, s.t3, "buy", trPendingBuy)
	for _, id := range []string{s.rewardA, s.rewardD} {
		_, found := s.leaderboard(t, id)
		require.False(t, found, "pending reward %s has a leaderboard", id)
	}
}

// 2. Every refusal leaves the creator with only the tx fee paid (none when
// CheckTx refused the msg) and the rewards module and the pending rewards
// untouched.
func (s *tradingRewardsSuite) createRefusals(t *testing.T) {
	prize, market := strconv.FormatInt(trRewardA.prize, 10), s.trMarket
	for _, c := range []struct {
		name                        string
		user                        ibc.Wallet
		prize, denom, duration, mkt string
		slots, codespace, reason    string
		code                        uint32
	}{
		{"unknown market", s.creator, prize, s.prize, "2", "nope/" + chain.Denom, "2", rewardsCodespace, "invalid market_id", 5007},
		{"prize denom without supply", s.creator, prize, "factory/" + s.creator.FormattedAddress() + "/ghost", "2", market, "2", rewardsCodespace, "invalid prize denom", 5002},
		{"duration 0", s.creator, prize, s.prize, "0", market, "2", rewardsCodespace, "invalid duration", 5005},
		{"duration 366", s.creator, prize, s.prize, "366", market, "2", rewardsCodespace, "invalid duration", 5005},
		{"slots 0", s.creator, prize, s.prize, "2", market, "0", rewardsCodespace, "invalid slots", 5008},
		{"slots 11", s.creator, prize, s.prize, "2", market, "11", rewardsCodespace, "invalid slots", 5008},
		// t1 holds no prize denom; the msg server refuses with a plain error
		{"balance too low", s.t1, prize, s.prize, "2", market, "1", "undefined", "user balance is too low", 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			before, pending := s.balances(t, c.user, chain.Denom, s.prize), s.rewardIDs(t, "pending")
			res, err := helpers.ExecTxWith(s.ctx, s.bze, c.user, []helpers.TxOption{helpers.WithGas(trTxGas)},
				"rewards", "create-trading-reward", c.prize, c.denom, c.duration, c.mkt, c.slots)
			s.requireRefused(t, res, err, c.codespace, c.code, c.reason)
			require.Equal(t, before.Sub(s.paidFee(t, res)...), s.balances(t, c.user, chain.Denom, s.prize), "balances of %s", c.user.KeyName())
			require.Equal(t, pending, s.rewardIDs(t, "pending"), "pending rewards")
			s.requireEscrow(t)
		})
	}
}

// 3. Only governance activates a reward. A proposal activating an unknown
// reward fails; the same msg signed by a user is refused. One proposal
// activates A and B: they move from pending to active and their markets map
// to them, but their countdown waits for the first trade (expire_at 0).
// Activating A again fails, and so does D, pending on a market that now has
// an active reward; a new reward on that market is refused.
func (s *tradingRewardsSuite) activate(t *testing.T) {
	t.Run("unknown reward", func(t *testing.T) {
		s.requireProposalFails(t, "activate an unknown trading reward", "trading reward not found", s.activateMsg("999999999999"))
	})

	t.Run("signed by a user", func(t *testing.T) {
		msg := s.activateMsg(s.rewardA)
		msg["creator"] = s.t1.FormattedAddress()
		before := s.balances(t, s.t1, chain.Denom)
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.t1, []helpers.TxOption{helpers.WithGas(trTxGas)}, msg)
		s.requireRefused(t, res, err, rewardsCodespace, 5011, "invalid authority")
		require.Equal(t, before.Sub(s.paidFee(t, res)...), s.balances(t, s.t1, chain.Denom), "t1 balances")
		require.Contains(t, s.rewardIDs(t, "pending"), s.rewardA)
	})

	t.Run("activate A and B", func(t *testing.T) {
		_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.creator, helpers.Proposal{
			Messages: []any{s.activateMsg(s.rewardA), s.activateMsg(s.rewardB)},
			Title:    "activate trading rewards",
			Summary:  "e2e: activate the market and the pool trading rewards",
		})
		require.NoError(t, err)

		height := s.activationHeight(t, s.rewardA, s.rewardB)
		require.Empty(t, s.rewardEvents(t, height, "TradingRewardStartEvent"), "activation starts no countdown")
		for id, market := range map[string]string{s.rewardA: s.trMarket, s.rewardB: s.trPool} {
			spec := s.live[id]
			require.Equal(t, s.wantReward(id, spec, market, 0), s.reward(t, id), "active reward %s", id)
			mapped, found := s.marketReward(t, market)
			require.True(t, found, "market %s has no trading reward", market)
			require.Equal(t, id, mapped)
		}
		require.Equal(t, []string{s.rewardD}, s.rewardIDs(t, "pending"), "pending rewards")
		require.ElementsMatch(t, []string{s.rewardA, s.rewardB}, s.rewardIDs(t, "active"), "active rewards")
		s.requireEscrow(t)
	})

	t.Run("already active", func(t *testing.T) {
		s.requireProposalFails(t, "activate an active trading reward", "trading reward already active", s.activateMsg(s.rewardA))
	})

	t.Run("market already has an active reward", func(t *testing.T) {
		s.requireProposalFails(t, "activate a second trading reward on a market", "a reward is already running for this market", s.activateMsg(s.rewardD))
		require.Equal(t, []string{s.rewardD}, s.rewardIDs(t, "pending"), "D stays pending")
		mapped, _ := s.marketReward(t, s.trMarket)
		require.Equal(t, s.rewardA, mapped)
	})

	t.Run("create on a market with an active reward", func(t *testing.T) {
		before := s.balances(t, s.creator, chain.Denom, s.prize)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.creator, []helpers.TxOption{helpers.WithGas(trTxGas)},
			"rewards", "create-trading-reward", "1000", s.prize, "2", s.trMarket, "1")
		s.requireRefused(t, res, err, rewardsCodespace, 5010, "a reward is already running for this market")
		require.Equal(t, before.Sub(s.paidFee(t, res)...), s.balances(t, s.creator, chain.Denom, s.prize), "creator balances")
		s.requireEscrow(t)
	})
}

// 4. The order-book hook credits the taker with the base amount executed,
// buying or selling, never the maker. The first trade on each reward starts
// its countdown: it expires 2 days of hour epochs after the block of that
// trade, and later trades leave it alone. t3 trades first, t1 trades more; when
// t3 tops up to t1's amount the tie goes to t3, whose leaderboard entry is
// older. Swaps on the pool credit the pool's reward B with the base amount
// (the swap output when buying tr, the input when selling it) and never the
// market's reward A: a reward counts trades on its own market id only.
func (s *tradingRewardsSuite) hooks(t *testing.T) {
	height := s.trade(t, s.t3, "buy", 500)
	s.expireAtA = s.hourAt(t, height) + trDuration*24
	s.requireStarted(t, s.rewardEvents(t, height, "TradingRewardStartEvent"), s.rewardA, s.expireAtA)
	s.requireLeaderboard(t, s.rewardA, trPlace{s.t3, 500})
	// the next entry must be created in a later second than t3's
	require.NoError(t, testutil.WaitForBlocks(s.ctx, 2, s.bze))

	s.trade(t, s.t1, "buy", 2000)
	s.requireLeaderboard(t, s.rewardA, trPlace{s.t1, 2000}, trPlace{s.t3, 500})
	s.trade(t, s.t1, "sell", 1000)
	s.requireLeaderboard(t, s.rewardA, trPlace{s.t1, 3000}, trPlace{s.t3, 500})

	s.trade(t, s.t3, "buy", 2500)
	board := s.requireLeaderboard(t, s.rewardA, trPlace{s.t3, 3000}, trPlace{s.t1, 3000})
	require.Less(t, board[0].CreatedAt, board[1].CreatedAt, "the tie goes to the older entry")

	out, res := s.swap(t, s.t2, sdk.NewInt64Coin(chain.Denom, trSwapBze))
	s.expireAtB = s.hourAt(t, res.Height) + trDuration*24
	s.requireStarted(t, helpers.TxEvents(res, drEventPrefix+"TradingRewardStartEvent"), s.rewardB, s.expireAtB)
	s.requireLeaderboard(t, s.rewardB, trPlace{s.t2, out.Int64()})
	s.swap(t, s.t2, sdk.NewInt64Coin(s.tr, trSwapBase))
	s.swapped = out.Int64() + trSwapBase
	s.requireLeaderboard(t, s.rewardB, trPlace{s.t2, s.swapped})

	// the maker filled every trade above and the swaps never reached reward A
	s.requireLeaderboard(t, s.rewardA, trPlace{s.t3, 3000}, trPlace{s.t1, 3000})
	// later trades did not move the countdowns
	require.Equal(t, s.wantReward(s.rewardA, trRewardA, s.trMarket, s.expireAtA), s.reward(t, s.rewardA))
	require.Equal(t, s.wantReward(s.rewardB, trRewardB, s.trPool, s.expireAtB), s.reward(t, s.rewardB))
}

// 5. With two slots, a third trader is cut from the leaderboard but keeps
// the amount traded: when t2 trades again the total puts it back on top and
// t1 is cut.
func (s *tradingRewardsSuite) trim(t *testing.T) {
	s.trade(t, s.t2, "buy", 1000)
	s.requireLeaderboard(t, s.rewardA, trPlace{s.t3, 3000}, trPlace{s.t1, 3000})

	s.trade(t, s.t2, "buy", 2500)
	s.requireLeaderboard(t, s.rewardA, trPlace{s.t2, 3500}, trPlace{s.t3, 3000})

	cur, err := helpers.CurrentEpoch(s.ctx, s.bze, "hour")
	require.NoError(t, err)
	require.Less(t, cur, min(s.expireAtA, s.expireAtB), "the trades must land before the rewards expire")
}

// 6. At the end of the hour epoch each reward expires on, every winner gets
// the prize per slot: t2 and t3 for A, t2 for B, nothing for t1. Every slot
// was filled, so nothing is burnt. The markets are free again (a new reward
// on the market is accepted), trades no longer count, and the paid rewards
// and their leaderboards stay queryable for the display week.
func (s *tradingRewardsSuite) payout(t *testing.T) {
	before := map[string]sdkmath.Int{}
	for _, u := range []ibc.Wallet{s.t1, s.t2, s.t3} {
		before[u.FormattedAddress()] = s.balance(t, u.FormattedAddress(), s.prize)
	}

	s.epochEndHeight(t, max(s.expireAtA, s.expireAtB), trPayoutWait)
	var all []helpers.Event
	heights := map[int64]bool{}
	for id, want := range map[string]struct {
		spec     trSpec
		expireAt int64
		winners  []ibc.Wallet
	}{
		s.rewardA: {trRewardA, s.expireAtA, []ibc.Wallet{s.t2, s.t3}},
		s.rewardB: {trRewardB, s.expireAtB, []ibc.Wallet{s.t2}},
	} {
		height := s.epochStartHeight(t, want.expireAt+1)
		events := s.rewardEvents(t, height, "TradingRewardDistributionEvent")
		ev := helpers.EventsWith(events, map[string]string{
			"reward_id":    id,
			"prize_amount": coinStr(want.spec.prize, s.prize),
			"prize_denom":  s.prize,
		})
		require.Len(t, ev, 1, "distribution of %s at height %d: %+v", id, height, events)
		var winners []string
		require.NoError(t, json.Unmarshal([]byte(ev[0].Attributes["winners"]), &winners), "winners %q", ev[0].Attributes["winners"])
		require.Equal(t, addresses(want.winners...), winners, "winners of %s", id)
		require.Empty(t, ev[0].Attributes["unpaid_burnt"], "every slot of %s was filled", id)
		if !heights[height] {
			heights[height] = true
			all = append(all, s.blockEvents(t, height)...)
		}
	}
	require.Zero(t, s.transferred(t, all, s.rewards, s.burner, s.prize).Int64(), "nothing burnt")

	for _, w := range []struct {
		user ibc.Wallet
		won  int64
	}{
		{s.t1, 0},
		{s.t2, trRewardA.prize + trRewardB.prize},
		{s.t3, trRewardA.prize},
	} {
		addr := w.user.FormattedAddress()
		require.Equal(t, sdkmath.NewInt(w.won), s.transferred(t, all, s.rewards, addr, s.prize), "prize paid to %s", w.user.KeyName())
		require.Equal(t, before[addr].AddRaw(w.won), s.balance(t, addr, s.prize), "prize balance of %s", w.user.KeyName())
	}
	delete(s.live, s.rewardA)
	delete(s.live, s.rewardB)
	s.requireEscrow(t)

	for _, r := range []struct {
		id, market string
		spec       trSpec
		expireAt   int64
	}{
		{s.rewardA, s.trMarket, trRewardA, s.expireAtA},
		{s.rewardB, s.trPool, trRewardB, s.expireAtB},
	} {
		_, found := s.marketReward(t, r.market)
		require.False(t, found, "market %s still mapped after the payout", r.market)
		require.Equal(t, s.wantReward(r.id, r.spec, r.market, r.expireAt), s.reward(t, r.id), "paid reward %s stays queryable", r.id)
	}
	require.ElementsMatch(t, []string{s.rewardA, s.rewardB}, s.rewardIDs(t, "active"), "active rewards during the display week")
	s.requireLeaderboard(t, s.rewardB, trPlace{s.t2, s.swapped})

	s.trade(t, s.t1, "buy", 100)
	s.requireLeaderboard(t, s.rewardA, trPlace{s.t2, 3500}, trPlace{s.t3, 3000})

	s.rewardE = s.createReward(t, trRewardE, s.trMarket)
	s.requireEscrow(t)
}

// 7. A week of hour epochs after the payout the paid rewards and their
// leaderboards are removed, and nobody is paid twice.
func (s *tradingRewardsSuite) cleanup(t *testing.T) {
	before := s.prizeBalances(t)
	s.epochEndHeight(t, max(s.expireAtA, s.expireAtB)+trDisplayEpochs, trCleanupWait)
	for _, expireAt := range []int64{s.expireAtA, s.expireAtB} {
		height := s.epochStartHeight(t, expireAt+trDisplayEpochs+1)
		require.Empty(t, s.rewardEvents(t, height, "TradingRewardDistributionEvent"), "no second distribution at height %d", height)
	}

	for _, id := range []string{s.rewardA, s.rewardB} {
		_, err := s.queryReward(id)
		require.ErrorContains(t, err, "not found", "reward %s after the display week", id)
		_, found := s.leaderboard(t, id)
		require.False(t, found, "leaderboard of %s after the display week", id)
	}
	require.Empty(t, s.rewardIDs(t, "active"), "active rewards")
	require.ElementsMatch(t, []string{s.rewardD, s.rewardE}, s.rewardIDs(t, "pending"), "pending rewards")
	require.Equal(t, before, s.prizeBalances(t), "prize balances")
	s.requireEscrow(t)
}

// 8. One proposal activates D (3 slots, on the now free market) and F (on
// the now free pool). Only t1 trades on D: at expiry t1 gets one slot's
// prize, the prize of the two slots nobody filled goes to the burner, and
// the market is free again.
func (s *tradingRewardsSuite) fewerTraders(t *testing.T) {
	s.rewardF = s.createReward(t, trRewardF, s.trPool)
	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.creator, helpers.Proposal{
		Messages: []any{s.activateMsg(s.rewardD), s.activateMsg(s.rewardF)},
		Title:    "activate trading rewards D and F",
		Summary:  "e2e: a reward with fewer traders than slots and a reward nobody trades on",
	})
	require.NoError(t, err)
	s.activatedDF = s.hourAt(t, s.activationHeight(t, s.rewardD, s.rewardF))

	before := s.balance(t, s.t1.FormattedAddress(), s.prize)
	height := s.trade(t, s.t1, "buy", trFewerBuy)
	expireAt := s.hourAt(t, height) + trDuration*24
	s.requireStarted(t, s.rewardEvents(t, height, "TradingRewardStartEvent"), s.rewardD, expireAt)
	s.requireLeaderboard(t, s.rewardD, trPlace{s.t1, trFewerBuy})

	s.epochEndHeight(t, expireAt, trPayoutWait)
	height = s.epochStartHeight(t, expireAt+1)
	unfilled := coinStr(trRewardD.prize*(trRewardD.slots-1), s.prize)
	events := helpers.EventsWith(s.rewardEvents(t, height, "TradingRewardDistributionEvent"), map[string]string{
		"reward_id":    s.rewardD,
		"prize_amount": coinStr(trRewardD.prize, s.prize),
		"unpaid_burnt": unfilled,
	})
	require.Len(t, events, 1, "distribution of D at height %d", height)

	all := s.blockEvents(t, height)
	addr := s.t1.FormattedAddress()
	require.Equal(t, sdkmath.NewInt(trRewardD.prize), s.transferred(t, all, s.rewards, addr, s.prize), "prize paid to t1")
	require.Equal(t, before.AddRaw(trRewardD.prize), s.balance(t, addr, s.prize), "prize balance of t1")
	require.Equal(t, sdkmath.NewInt(trRewardD.prize*(trRewardD.slots-1)), s.transferred(t, all, s.rewards, s.burner, s.prize), "unfilled slots burnt")
	delete(s.live, s.rewardD)
	s.requireEscrow(t)

	_, found := s.marketReward(t, s.trMarket)
	require.False(t, found, "market still mapped after D's payout")
}

// 9. F, activated with D, saw no trade: well past its duration it is still
// active with no countdown (expire_at 0), keeps its escrow and keeps the
// pool, so a new reward on the pool is refused.
func (s *tradingRewardsSuite) noTrades(t *testing.T) {
	cur, err := helpers.CurrentEpoch(s.ctx, s.bze, "hour")
	require.NoError(t, err)
	require.Greater(t, cur, s.activatedDF+trDuration*24, "F must be past the expiry its activation would have set")

	require.Equal(t, s.wantReward(s.rewardF, trRewardF, s.trPool, 0), s.reward(t, s.rewardF), "F waits for its first trade")
	require.Contains(t, s.rewardIDs(t, "active"), s.rewardF)
	_, found := s.leaderboard(t, s.rewardF)
	require.False(t, found, "F has no leaderboard")
	mapped, found := s.marketReward(t, s.trPool)
	require.True(t, found, "the pool stays mapped to F")
	require.Equal(t, s.rewardF, mapped)

	before := s.balances(t, s.creator, chain.Denom, s.prize)
	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.creator, []helpers.TxOption{helpers.WithGas(trTxGas)},
		"rewards", "create-trading-reward", "1000", s.prize, "2", s.trPool, "1")
	s.requireRefused(t, res, err, rewardsCodespace, 5010, "a reward is already running for this market")
	require.Equal(t, before.Sub(s.paidFee(t, res)...), s.balances(t, s.creator, chain.Denom, s.prize), "creator balances")
	s.requireEscrow(t)
}

// 10. The rewards module holds exactly the escrow of the rewards not paid
// yet (E pending, F active with no trades); fees only passed through it and
// D's unfilled slots left it for the burner.
func (s *tradingRewardsSuite) conservation(t *testing.T) {
	require.Equal(t, map[string]trSpec{s.rewardE: trRewardE, s.rewardF: trRewardF}, s.live)
	s.requireEscrow(t)
}

// createReward creates a trading reward of the suite's duration on market
// and asserts the escrow, the fee path, the event and the pending record.
func (s *tradingRewardsSuite) createReward(t *testing.T, spec trSpec, market string) string {
	t.Helper()
	creator := s.creator.FormattedAddress()
	before := s.balances(t, s.creator, chain.Denom, s.prize)

	res := s.mustExecWith(t, s.creator, []helpers.TxOption{helpers.WithGas(trTxGas)}, "rewards", "create-trading-reward",
		strconv.FormatInt(spec.prize, 10), s.prize, strconv.Itoa(trDuration), market, strconv.FormatInt(spec.slots, 10))

	events := helpers.EventsWith(helpers.TxEvents(res, drEventPrefix+"TradingRewardCreateEvent"), map[string]string{
		"creator":      creator,
		"prize_amount": strconv.FormatInt(spec.prize, 10),
		"prize_denom":  s.prize,
		"duration":     strconv.Itoa(trDuration),
		"market_id":    market,
		"slots":        strconv.FormatInt(spec.slots, 10),
	})
	require.Len(t, events, 1, "TradingRewardCreateEvent: %+v", helpers.TxEvents(res, drEventPrefix+"TradingRewardCreateEvent"))
	id := events[0].Attributes["reward_id"]
	require.Len(t, id, 12, "reward id %q", id)

	fee := s.createFee.Amount
	require.Equal(t, sdkmath.NewInt(spec.escrow()), s.transferred(t, res.Events, creator, s.rewards, s.prize), "escrow to the rewards module")
	require.Equal(t, fee, s.transferred(t, res.Events, creator, s.rewards, chain.Denom), "create fee captured")
	require.Equal(t, fee, s.transferred(t, res.Events, s.rewards, s.cpFees, chain.Denom), "create fee to the community pool collector")
	want := before.Sub(sdk.NewInt64Coin(s.prize, spec.escrow()), sdk.NewCoin(chain.Denom, fee.Add(s.txFee(t, res))))
	require.Equal(t, want, s.balances(t, s.creator, chain.Denom, s.prize), "creator balances")

	require.Equal(t, s.wantReward(id, spec, market, s.hourAt(t, res.Height)+trPendingDays*24), s.reward(t, id), "pending reward")
	require.Contains(t, s.rewardIDs(t, "pending"), id)
	require.NotContains(t, s.rewardIDs(t, "active"), id)
	_, found := s.marketReward(t, market)
	require.False(t, found, "a pending reward does not map its market")

	s.live[id] = spec
	return id
}

// trade sends a taker order against the maker's resting order at the price
// for its side, asserts the execution and the taker's balances and returns
// the height of the block that executed it.
func (s *tradingRewardsSuite) trade(t *testing.T, u ibc.Wallet, typ string, amount int64) int64 {
	t.Helper()
	price := trSellPrice
	if typ == "sell" {
		price = trBuyPrice
	}
	amt := strconv.FormatInt(amount, 10)
	before := s.balances(t, u, chain.Denom, s.tr)

	res := s.mustExec(t, u, "tradebin", "create-order", typ, amt, price, s.trMarket)

	executed := helpers.EventsWith(filterType(s.blockEvents(t, res.Height), tradebinEventPrefix+"OrderExecutedEvent"), map[string]string{
		"market_id": s.trMarket, "taker": u.FormattedAddress(), "maker": s.maker.FormattedAddress(), "amount": amt, "price": price,
	})
	require.Len(t, executed, 1, "%s %s %s@%s executes against the maker", u.KeyName(), typ, amt, price)

	quote := mulPrice(amount, price)
	fee := s.params.MarketTakerFee.Amount.Add(s.txFee(t, res))
	want := before.Add(sdk.NewInt64Coin(s.tr, amount)).Sub(sdk.NewCoin(chain.Denom, quote.Add(fee)))
	if typ == "sell" {
		want = before.Add(sdk.NewCoin(chain.Denom, quote)).Sub(sdk.NewInt64Coin(s.tr, amount), sdk.NewCoin(chain.Denom, fee))
	}
	require.Equal(t, want, s.balances(t, u, chain.Denom, s.tr), "balances of %s", u.KeyName())
	return res.Height
}

// swap swaps in on the tr/ubze pool with the exact output, asserts the
// swapper's balances and returns the output amount and the tx.
func (s *tradingRewardsSuite) swap(t *testing.T, u ibc.Wallet, in sdk.Coin) (sdkmath.Int, helpers.TxResult) {
	t.Helper()
	pool := s.pool(t, s.trPool)
	q, outDenom := quoteSwap(in.Amount, pool.reserveQuote(), pool.reserveBase()), s.tr
	if in.Denom == s.tr {
		q, outDenom = quoteSwap(in.Amount, pool.reserveBase(), pool.reserveQuote()), chain.Denom
	}
	before := s.balances(t, u, chain.Denom, s.tr)

	res := s.mustExec(t, u, "tradebin", "multi-swap", s.trPool, in.String(), coinStr(1, outDenom))

	out := sdk.NewCoin(outDenom, q.out)
	s.requireSwap(t, res.Events, u.FormattedAddress(), s.trPool, in, out)
	fee := sdk.NewCoin(chain.Denom, s.params.MarketTakerFee.Amount.Add(s.txFee(t, res)))
	require.Equal(t, before.Add(out).Sub(in).Sub(fee), s.balances(t, u, chain.Denom, s.tr), "balances of %s", u.KeyName())
	return q.out, res
}

type trReward struct {
	RewardID    string        `json:"reward_id"`
	PrizeAmount string        `json:"prize_amount"`
	PrizeDenom  string        `json:"prize_denom"`
	Duration    helpers.Int64 `json:"duration"`
	MarketID    string        `json:"market_id"`
	Slots       helpers.Int64 `json:"slots"`
	ExpireAt    helpers.Int64 `json:"expire_at"`
}

func (s *tradingRewardsSuite) wantReward(id string, spec trSpec, market string, expireAt int64) trReward {
	return trReward{
		RewardID:    id,
		PrizeAmount: strconv.FormatInt(spec.prize, 10),
		PrizeDenom:  s.prize,
		Duration:    trDuration,
		MarketID:    market,
		Slots:       helpers.Int64(spec.slots),
		ExpireAt:    helpers.Int64(expireAt),
	}
}

func (s *tradingRewardsSuite) queryReward(id string) (trReward, error) {
	res, err := helpers.Query[struct {
		TradingReward trReward `json:"trading_reward"`
	}](s.ctx, s.bze, "rewards", "trading-reward", id)
	return res.TradingReward, err
}

func (s *tradingRewardsSuite) reward(t *testing.T, id string) trReward {
	t.Helper()
	r, err := s.queryReward(id)
	require.NoError(t, err)
	return r
}

// rewardIDs lists the ids of the pending or active trading rewards, sorted.
func (s *tradingRewardsSuite) rewardIDs(t *testing.T, state string) []string {
	t.Helper()
	res, err := helpers.Query[struct {
		List []trReward `json:"list"`
	}](s.ctx, s.bze, "rewards", "all-trading-rewards", "--state", state)
	require.NoError(t, err)
	ids := []string{}
	for _, r := range res.List {
		ids = append(ids, r.RewardID)
	}
	slices.Sort(ids)
	return ids
}

// marketReward returns the reward id a market or pool is mapped to.
func (s *tradingRewardsSuite) marketReward(t *testing.T, market string) (string, bool) {
	t.Helper()
	res, err := helpers.Query[struct {
		MarketReward struct {
			RewardID string `json:"reward_id"`
			MarketID string `json:"market_id"`
		} `json:"market_reward"`
	}](s.ctx, s.bze, "rewards", "market-trading-reward", market)
	if err != nil {
		require.ErrorContains(t, err, "not found")
		return "", false
	}
	require.Equal(t, market, res.MarketReward.MarketID)
	return res.MarketReward.RewardID, true
}

type trEntry struct {
	Amount    string        `json:"amount"`
	Address   string        `json:"address"`
	CreatedAt helpers.Int64 `json:"created_at"`
}

type trPlace struct {
	user   ibc.Wallet
	amount int64
}

func (s *tradingRewardsSuite) leaderboard(t *testing.T, id string) ([]trEntry, bool) {
	t.Helper()
	res, err := helpers.Query[struct {
		Leaderboard struct {
			RewardID string    `json:"reward_id"`
			List     []trEntry `json:"list"`
		} `json:"leaderboard"`
	}](s.ctx, s.bze, "rewards", "trading-reward-leaderboard", id)
	if err != nil {
		require.ErrorContains(t, err, "not found")
		return nil, false
	}
	require.Equal(t, id, res.Leaderboard.RewardID)
	return res.Leaderboard.List, true
}

// requireLeaderboard asserts the leaderboard's traders and amounts in order
// and returns its entries.
func (s *tradingRewardsSuite) requireLeaderboard(t *testing.T, id string, want ...trPlace) []trEntry {
	t.Helper()
	board, found := s.leaderboard(t, id)
	require.True(t, found, "reward %s has no leaderboard", id)
	type place struct{ Address, Amount string }
	var got, exp []place
	for _, e := range board {
		got = append(got, place{e.Address, e.Amount})
	}
	for _, w := range want {
		exp = append(exp, place{w.user.FormattedAddress(), strconv.FormatInt(w.amount, 10)})
	}
	require.Equal(t, exp, got, "leaderboard of %s", id)
	return board
}

func (s *tradingRewardsSuite) activateMsg(id string) map[string]any {
	return map[string]any{
		"@type":     "/bze.rewards.MsgActivateTradingReward",
		"creator":   s.gov,
		"reward_id": id,
	}
}

// requireProposalFails passes a proposal whose execution fails with reason;
// the pending and active rewards and the escrow stay as they were.
func (s *tradingRewardsSuite) requireProposalFails(t *testing.T, title, reason string, msgs ...any) {
	t.Helper()
	pending, active := s.rewardIDs(t, "pending"), s.rewardIDs(t, "active")
	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.creator, helpers.Proposal{Messages: msgs, Title: title, Summary: "e2e: " + title})
	require.ErrorContains(t, err, "PROPOSAL_STATUS_FAILED")
	require.ErrorContains(t, err, reason)
	require.Equal(t, pending, s.rewardIDs(t, "pending"), "pending rewards")
	require.Equal(t, active, s.rewardIDs(t, "active"), "active rewards")
	s.requireEscrow(t)
}

// activationHeight finds the block whose gov EndBlock activated every id,
// among the last blocks (the proposal just passed).
func (s *tradingRewardsSuite) activationHeight(t *testing.T, ids ...string) int64 {
	t.Helper()
	cur, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	for h := cur; h > cur-20 && h > 0; h-- {
		events := s.rewardEvents(t, h, "TradingRewardActivationEvent")
		if len(events) == 0 {
			continue
		}
		var got []string
		for _, e := range events {
			got = append(got, e.Attributes["reward_id"])
		}
		require.ElementsMatch(t, ids, got, "activations at height %d", h)
		return h
	}
	require.FailNow(t, "no TradingRewardActivationEvent", "in the 20 blocks up to %d", cur)
	return 0
}

// hourAt is the hour epoch number in the state of height.
func (s *tradingRewardsSuite) hourAt(t *testing.T, height int64) int64 {
	t.Helper()
	res, err := helpers.Query[struct {
		CurrentEpoch helpers.Int64 `json:"current_epoch"`
	}](s.ctx, s.bze, "epochs", "identifier", "hour", "--height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	return int64(res.CurrentEpoch)
}

// requireStarted asserts events hold exactly one TradingRewardStartEvent, for
// id with expireAt, and that the reward now carries that expire_at.
func (s *tradingRewardsSuite) requireStarted(t *testing.T, events []helpers.Event, id string, expireAt int64) {
	t.Helper()
	require.Len(t, events, 1, "TradingRewardStartEvent: %+v", events)
	require.Equal(t, id, events[0].Attributes["reward_id"])
	require.Equal(t, strconv.FormatInt(expireAt, 10), events[0].Attributes["expire_at"])
	require.EqualValues(t, expireAt, s.reward(t, id).ExpireAt, "expire_at of %s", id)
}

// epochStartHeight finds the first block of hour epoch n, which has already
// started: the block whose BeginBlock ran the AfterEpochEnd hooks for n-1.
// Hour epochs are monotonic in height, so it binary-searches the heights.
func (s *tradingRewardsSuite) epochStartHeight(t *testing.T, n int64) int64 {
	t.Helper()
	cur, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, s.hourAt(t, cur), n, "hour epoch %d has not started at height %d", n, cur)
	lo, hi := int64(1), cur // the answer is in (lo, hi]
	for hi-lo > 1 {
		mid := lo + (hi-lo)/2
		if s.hourAt(t, mid) >= n {
			hi = mid
		} else {
			lo = mid
		}
	}
	require.EqualValues(t, n, s.hourAt(t, hi), "first block of hour epoch %d", n)
	return hi
}

// epochEndHeight waits for hour epoch n to end and returns the height of the
// block whose BeginBlock ran the AfterEpochEnd hooks for n (the start block
// of epoch n+1).
func (s *tradingRewardsSuite) epochEndHeight(t *testing.T, n int64, wait time.Duration) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(s.ctx, wait)
	defer cancel()
	require.NoError(t, helpers.WaitForEpoch(ctx, s.bze, "hour", n+1))
	hour, err := helpers.Epoch(s.ctx, s.bze, "hour")
	require.NoError(t, err)
	require.EqualValues(t, n+1, hour.CurrentEpoch, "hour epoch %d already ended; the suite missed its start block", n+1)
	return int64(hour.CurrentEpochStartHeight)
}

func (s *tradingRewardsSuite) rewardEvents(t *testing.T, height int64, name string) []helpers.Event {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, height, drEventPrefix+name)
	require.NoError(t, err)
	return events
}

// requireRefused asserts a refusal with codespace, code and the chain's message.
func (s *tradingRewardsSuite) requireRefused(t *testing.T, res helpers.TxResult, err error, codespace string, code uint32, reason string) {
	t.Helper()
	got, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, got, reason)
	require.EqualValues(t, code, res.Code, got)
	require.Equal(t, codespace, res.Codespace, got)
}

// paidFee is the tx fee of a refused tx: none when CheckTx refused it.
func (s *tradingRewardsSuite) paidFee(t *testing.T, res helpers.TxResult) sdk.Coins {
	t.Helper()
	if res.Height == 0 {
		return nil
	}
	return sdk.NewCoins(sdk.NewCoin(chain.Denom, s.txFee(t, res)))
}

func (s *tradingRewardsSuite) prizeBalances(t *testing.T) map[string]sdkmath.Int {
	t.Helper()
	out := map[string]sdkmath.Int{}
	for _, u := range []ibc.Wallet{s.t1, s.t2, s.t3} {
		out[u.FormattedAddress()] = s.balance(t, u.FormattedAddress(), s.prize)
	}
	return out
}

func (s *tradingRewardsSuite) moduleBalances(t *testing.T) sdk.Coins {
	t.Helper()
	out := sdk.NewCoins()
	for _, d := range []string{chain.Denom, s.prize} {
		out = out.Add(sdk.NewCoin(d, s.balance(t, s.rewards, d)))
	}
	return out
}

// requireEscrow asserts the rewards module holds its balance before the
// suite plus the escrow of every reward not paid yet.
func (s *tradingRewardsSuite) requireEscrow(t *testing.T) {
	t.Helper()
	var escrow int64
	for _, spec := range s.live {
		escrow += spec.escrow()
	}
	want := s.baseline.Add(sdk.NewInt64Coin(s.prize, escrow))
	require.Equal(t, want.String(), s.moduleBalances(t).String(), "rewards module balances")
}

// tradingRewardFee reads create_trading_reward_fee from the rewards params
// (camelCase or snake_case, like the denom rewards suite).
func (s *tradingRewardsSuite) tradingRewardFee(t *testing.T) sdk.Coin {
	t.Helper()
	res, err := helpers.Query[struct {
		Params map[string]json.RawMessage `json:"params"`
	}](s.ctx, s.bze, "rewards", "params")
	require.NoError(t, err)
	raw, ok := res.Params["createTradingRewardFee"]
	if !ok {
		raw, ok = res.Params["create_trading_reward_fee"]
	}
	require.True(t, ok, "create_trading_reward_fee missing from %v", res.Params)
	var c struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	}
	require.NoError(t, json.Unmarshal(raw, &c))
	require.Equal(t, chain.Denom, c.Denom, "the suite pays the create fee in %s", chain.Denom)
	fee := sdk.NewCoin(c.Denom, parseInt(c.Amount))
	require.True(t, fee.IsPositive(), "create_trading_reward_fee %s", fee)
	return fee
}

func addresses(users ...ibc.Wallet) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		out = append(out, u.FormattedAddress())
	}
	return out
}
