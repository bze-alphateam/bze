package interchaintest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	upAdminFunds = 1_000_000_000_000 // 1,000,000 BZE: denoms, pools, a market, rewards, proposals
	upUserFunds  = 10_000_000_000    // 10,000 BZE each

	upValidators = 2
	// upTxGas is the fixed gas limit of every tx: --gas auto does not work on
	// v8.1.1 and older (see the README), and exits charge 1,000,000 extra gas.
	upTxGas = 2_000_000

	upMint = 1_000_000_000_000 // usd, abc and hlt minted by admin

	upPoolUsd    = 50_000_000_000
	upPoolBze    = 200_000_000_000
	upPoolAbc    = 40_000_000_000
	upPoolAbcUsd = 10_000_000_000
	upLpUsd      = 100_000_000 // alice's deposit, at the usd/ubze ratio of 1:4
	upLpBze      = 400_000_000

	upUserUsd = 1_000_000_000 // usd sent to alice, bob and carol
	upBobAbc  = 100_000_000

	// bob's resting orders on abc/usd; the pool prices abc at 0.25 usd
	upOrderAmount = 1_000
	upBuyPrice    = "1"
	upSellPrice   = "3"
	upCarolPrice  = "4"

	// the staking reward pays upSrPrize a day; with alice alone at
	// upSrAliceStake every day adds exactly 1 to its accumulator
	upSrPrize      = 1_000_000
	upSrDuration   = 30
	upSrLock       = 1 // day: 24 hour epochs, about 2 minutes
	upSrAliceStake = 1_000_000
	upSrBobStake   = 500_000 // joined and exited before the upgrade: a pending unlock

	// the denom reward pays upDrDaily a day to alice alone
	upDrDaily    = 1_000_000
	upDrDuration = 30
	upDrStake    = 1_000_000

	upPrzMint = upSrPrize*upSrDuration + upDrDaily*upDrDuration
	upStkMint = upSrAliceStake + upDrStake + upSrBobStake

	upBurnerFund = 1_000_000
	upSmokeMint  = 1_000
	upSmokeSwap  = 1_000_000
	upSmokeBurn  = 1_000

	// upHaltDelta is the number of blocks between submitting the upgrade
	// proposal and its plan height: the 20 s voting period plus a margin.
	upHaltDelta = 30
	// upHaltSettle is how long the halted chain must stay at the plan height.
	upHaltSettle = 5 * time.Second
	// upHaltTimeout bounds the wait for the plan height after the proposal passed.
	upHaltTimeout = 2 * time.Minute
	// upRestartTimeout bounds the wait for the candidate's first blocks.
	upRestartTimeout = 2 * time.Minute
)

// TestChainUpgrade runs a software upgrade on a two-validator network: it
// starts on the previous release (E2E_UPGRADE_FROM), seeds state in the custom
// modules, passes a software-upgrade proposal named E2E_UPGRADE_NAME, swaps the
// validators to the candidate image (E2E_UPGRADE_TO) at the plan height and
// asserts that nothing was lost and everything still works. The candidate's
// handler-specific assertions live in upgradeChecks. Steps 1-3 build on each
// other: the suite stops at the first of them that fails.
func TestChainUpgrade(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	cfg := chain.UpgradeFromEnvironment()
	check, ok := upgradeChecks[cfg.Name]
	require.True(t, ok, "no per-upgrade checks for %q: add an entry to upgradeChecks", cfg.Name)
	t.Logf("upgrade %s: bze:%s -> bze:%s", cfg.Name, cfg.From, cfg.To)

	ctx := context.Background()
	opts := []chain.Option{
		chain.WithImage(cfg.From),
		chain.WithValidators(upValidators),
		// the param only exists from v8.2.0 on; older genesis would not parse
		chain.WithoutGenesis(chain.DenomRewardLockKey),
	}
	var s *upgradeSuite
	if check.ibc {
		mnemonic, _ := newSimdAccount(t)
		inet := chain.StartIbc(t, ctx, nil, opts...)
		s = setupUpgrade(t, ctx, chain.Network{Chain: inet.Bze, Docker: inet.Docker}, cfg, check)
		s.linkSimd(t, inet, mnemonic)
	} else {
		s = setupUpgrade(t, ctx, chain.Start(t, ctx, opts...), cfg, check)
	}

	for _, step := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"1 seed state on the previous release", s.seed},
		{"2 the plan halts the chain at its height", s.proposeAndHalt},
		{"3 the candidate resumes the chain", s.swapAndResume},
	} {
		if !t.Run(step.name, step.run) {
			t.FailNow()
		}
	}
	t.Run("4 state is preserved across the upgrade block", s.statePreserved)
	t.Run("5 "+cfg.Name+" handler checks", s.handlerChecks)
	t.Run("6 the next day tick pays out", s.nextDayTick)
	t.Run("7 one tx per module", s.smoke)
	t.Run("8 the pending unlock is returned", s.unlockReturned)
}

// upgradeCheck holds the assertions of one upgrade handler. before runs on
// the previous release before the proposal and returns what after needs;
// after runs on the candidate once the chain resumed.
type upgradeCheck struct {
	before func(s *upgradeSuite, t *testing.T) any
	after  func(s *upgradeSuite, t *testing.T, before any)
	// moves are the balances the handler moves between s.accounts (by name),
	// the only balance changes step 4 accepts besides a block's own
	moves func(s *upgradeSuite, before any) []upgradeMove
	// ibc links the network to simd over IBC, for a handler that touches IBC
	ibc bool
}

// upgradeMove is a coin the handler moves from one account to another.
type upgradeMove struct {
	from, to string
	coin     sdk.Coin
}

// upgradeChecks is keyed by upgrade name. It holds the entry of the handler
// the code builds; a release line replaces the previous release's entry with
// its own, the way app/upgrades keeps only the current handler.
var upgradeChecks = map[string]upgradeCheck{
	// v8.3.0 runs linked to simd to prove IBC across the ibc-go v10 move
	"v8.3.0": {before: (*upgradeSuite).v830Before, after: (*upgradeSuite).v830After, ibc: true},
}

// upgradeFeatures are the features of the previous release the seed uses.
type upgradeFeatures struct {
	denomRewards, haltedDenoms, branding bool
}

type upgradeSuite struct {
	ctx    context.Context
	bze    *cosmos.CosmosChain
	docker *client.Client
	cfg    chain.Upgrade
	check  upgradeCheck

	admin, alice, bob, carol ibc.Wallet
	gov                      string
	// accounts are the addresses whose balances must survive the upgrade
	accounts map[string]string

	usd, abc, stk, prz, hlt  string
	usdPool, abcPool, market string

	features upgradeFeatures
	srID     string
	// unlockAt is the hour epoch bob's exited stake unlocks at
	unlockAt       int64
	buyID, sellID  string
	drActive       bool
	plan           int64
	fromVer, toVer string
	handlerBefore  any
	before, after  upgradeSnapshot
	// blackHoleAddr and burnerAddr are the burner modules' accounts
	blackHoleAddr, burnerAddr string
	// ibc is the simd link of a run with upgradeCheck.ibc, nil otherwise; it
	// reuses the IBC suite's transfer helpers
	ibc *ibcSuite
}

// linkSimd prepares the IBC side of a run linked to simd: a funded simd user,
// transfers to and from admin over the channel StartIbc opened.
func (s *upgradeSuite) linkSimd(t *testing.T, net chain.IbcNetwork, simdMnemonic string) {
	simdUser, err := interchaintest.GetAndFundTestUserWithMnemonic(s.ctx, "up-simd", simdMnemonic, sdkmath.NewInt(ibcSimdFunds), net.Simd)
	require.NoError(t, err)
	require.NoError(t, testutil.WaitForBlocks(s.ctx, 2, net.Simd))
	s.ibc = &ibcSuite{ctx: s.ctx, net: net, bze: net.Bze, simd: net.Simd, admin: s.admin, bzeUser: s.admin, simdUser: simdUser, gov: s.gov}
	s.ibc.main = s.ibc.newChannel(t, net.BzeChannel, net.SimdChannel)
}

func setupUpgrade(t *testing.T, ctx context.Context, net chain.Network, cfg chain.Upgrade, check upgradeCheck) *upgradeSuite {
	bze := net.Chain
	s := &upgradeSuite{ctx: ctx, bze: bze, docker: net.Docker, cfg: cfg, check: check}

	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "up-admin", sdkmath.NewInt(upAdminFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "up", sdkmath.NewInt(upUserFunds), bze, bze, bze)
	s.alice, s.bob, s.carol = users[0], users[1], users[2]
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	var err error
	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)
	s.accounts = map[string]string{
		"admin": s.admin.FormattedAddress(),
		"alice": s.alice.FormattedAddress(),
		"bob":   s.bob.FormattedAddress(),
		"carol": s.carol.FormattedAddress(),
	}
	for _, module := range []string{"rewards", "tradebin", "burner", "burner_black_hole"} {
		s.accounts[module+" module"], err = helpers.ModuleAddress(ctx, bze, module)
		require.NoError(t, err)
	}
	s.blackHoleAddr, s.burnerAddr = s.accounts["burner_black_hole module"], s.accounts["burner module"]

	factory := "factory/" + s.admin.FormattedAddress() + "/"
	s.usd, s.abc, s.stk, s.prz, s.hlt = factory+"usd", factory+"abc", factory+"stk", factory+"prz", factory+"hlt"
	// pool ids order the denoms alphabetically: "factory/…" sorts before "ubze"
	s.usdPool = s.usd + "_" + chain.Denom
	s.abcPool = s.abc + "_" + s.usd
	s.market = s.abc + "/" + s.usd

	s.features = upgradeFeatures{
		denomRewards: s.hasParam(t, "rewards", "createDenomRewardFee", "create_denom_reward_fee"),
		haltedDenoms: s.supported(t, "tradebin", "halted-denoms"),
		branding:     s.supported(t, "tokenfactory", "all-denom-branding"),
	}
	s.fromVer = s.appVersion(t)
	t.Logf("previous release %s (bze:%s), features %+v", s.fromVer, cfg.From, s.features)
	return s
}

// 1. Seed what the previous release supports: factory denoms with metadata
// (and branding), two pools with LP holders, a market with resting orders, a
// staking reward with pending rewards and a pending unlock, a denom reward, a
// halted denom and a burner balance.
func (s *upgradeSuite) seed(t *testing.T) {
	subs := []string{"usd", "abc", "stk", "prz"}
	if s.features.haltedDenoms {
		subs = append(subs, "hlt")
	}
	mint := map[string]int64{s.stk: upStkMint, s.prz: upPrzMint}
	for _, sub := range subs {
		denom := "factory/" + s.admin.FormattedAddress() + "/" + sub
		amount, ok := mint[denom]
		if !ok {
			amount = upMint
		}
		s.exec(t, s.admin, "tokenfactory", "create-denom", sub)
		s.exec(t, s.admin, "tokenfactory", "mint", coinStr(amount, denom))
	}
	s.execMsg(t, s.admin, map[string]any{
		"@type":   "/bze.tokenfactory.MsgSetDenomMetadata",
		"creator": s.admin.FormattedAddress(),
		"metadata": map[string]any{
			"description": "USD for the upgrade suite",
			"denom_units": []map[string]any{{"denom": s.usd, "exponent": 0}, {"denom": "USD", "exponent": 6}},
			"base":        s.usd,
			"display":     "USD",
			"name":        "USD",
			"symbol":      "USD",
		},
	})
	if s.features.branding {
		s.execMsg(t, s.admin, map[string]any{
			"@type": "/bze.tokenfactory.MsgSetDenomBranding", "creator": s.admin.FormattedAddress(), "denom": s.usd, "branding": tfBranding("inter"),
		})
	}

	admin := s.admin.FormattedAddress()
	for _, send := range []struct {
		to   ibc.Wallet
		coin string
	}{
		{s.alice, coinStr(upUserUsd, s.usd)},
		{s.alice, coinStr(upSrAliceStake+upDrStake, s.stk)},
		{s.bob, coinStr(upUserUsd, s.usd)},
		{s.bob, coinStr(upBobAbc, s.abc)},
		{s.bob, coinStr(upSrBobStake, s.stk)},
		{s.carol, coinStr(upUserUsd, s.usd)},
	} {
		s.exec(t, s.admin, "bank", "send", admin, send.to.FormattedAddress(), send.coin)
	}

	// AMM: two pools, alice holds LP shares of usd/ubze
	for _, p := range []struct {
		base, quote         string
		initBase, initQuote int64
	}{
		{s.usd, chain.Denom, upPoolUsd, upPoolBze},
		{s.abc, s.usd, upPoolAbc, upPoolAbcUsd},
	} {
		s.exec(t, s.admin, "tradebin", "create-liquidity-pool", p.base, p.quote, ammPoolFee, ammFeeDest, "false",
			sdkmath.NewInt(p.initBase).String(), sdkmath.NewInt(p.initQuote).String())
	}
	s.exec(t, s.alice, "tradebin", "add-liquidity", s.usdPool, sdkmath.NewInt(upLpUsd).String(), sdkmath.NewInt(upLpBze).String(), "1")
	require.True(t, s.balance(t, s.alice.FormattedAddress(), s.pool(t, 0, s.usdPool).LpDenom).IsPositive(), "alice holds usd/ubze LP")

	// order book: bob rests a buy below and a sell above the pool price
	s.exec(t, s.admin, "tradebin", "create-market", s.abc, s.usd)
	s.exec(t, s.bob, "tradebin", "create-order", "buy", strconv.Itoa(upOrderAmount), upBuyPrice, s.market)
	s.exec(t, s.bob, "tradebin", "create-order", "sell", strconv.Itoa(upOrderAmount), upSellPrice, s.market)
	for _, o := range s.userOrders(t, 0, s.bob) {
		switch o.OrderType {
		case "buy":
			s.buyID = o.ID
		case "sell":
			s.sellID = o.ID
		}
	}
	require.NotEmpty(t, s.buyID, "bob's buy order rests")
	require.NotEmpty(t, s.sellID, "bob's sell order rests")

	// staking reward: alice and bob stake
	s.exec(t, s.admin, "rewards", "create-staking-reward",
		strconv.Itoa(upSrPrize), s.prz, s.stk, strconv.Itoa(upSrDuration), "1", strconv.Itoa(upSrLock))
	for _, sr := range s.allStakingRewards(t) {
		if sr.StakingDenom == s.stk {
			s.srID = sr.RewardID
		}
	}
	require.NotEmpty(t, s.srID, "staking reward of %s", s.stk)
	s.exec(t, s.alice, "rewards", "join-staking", s.srID, strconv.Itoa(upSrAliceStake))
	s.exec(t, s.bob, "rewards", "join-staking", s.srID, strconv.Itoa(upSrBobStake))

	if s.features.denomRewards {
		s.createDenomReward(t)
	} else {
		t.Log("the previous release has no denom rewards: the denom reward is created after the upgrade")
	}
	if s.features.haltedDenoms {
		_, err := helpers.SubmitAndPassProposalWith(s.ctx, s.bze, s.admin, s.txOpts(), helpers.Proposal{
			Messages: []any{map[string]any{"@type": "/bze.tradebin.MsgHaltDenoms", "authority": s.gov, "denoms": []string{s.hlt}}},
			Title:    "Halt hlt",
			Summary:  "e2e: a halted denom across the upgrade",
		})
		require.NoError(t, err)
		require.Equal(t, []string{s.hlt}, s.haltedDenoms(t, 0))
	} else {
		t.Log("the previous release has no halted denoms")
	}
	s.exec(t, s.bob, "burner", "fund-burner", coinStr(upBurnerFund, chain.Denom))

	// a payout so that alice has pending rewards, then bob's exit queues his
	// stake for unlock: 24 hour epochs, well after the upgrade height
	s.waitDayTick(t)
	require.Positive(t, int64(s.stakingReward(t, 0).Payouts), "the staking reward paid out")
	res := s.exec(t, s.bob, "rewards", "exit-staking", s.srID)
	hour := s.epoch(t, res.Height, "hour")
	s.unlockAt = hour + upSrLock*24
	_, found := s.pendingUnlock(t, 0, s.unlockIndex())
	require.True(t, found, "bob's stake is queued for unlock at hour %d", s.unlockAt)
}

// 2. The upgrade proposal passes and the previous release halts at exactly
// the plan height; its last state (plan height - 1) is the "before" snapshot.
func (s *upgradeSuite) proposeAndHalt(t *testing.T) {
	s.handlerBefore = s.check.before(s, t)

	height, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	s.plan = height + upHaltDelta
	_, err = helpers.SubmitAndPassProposalWith(s.ctx, s.bze, s.admin, s.txOpts(), helpers.Proposal{
		Messages: []any{map[string]any{
			"@type":     "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
			"authority": s.gov,
			"plan":      map[string]any{"name": s.cfg.Name, "height": strconv.FormatInt(s.plan, 10), "info": "e2e"},
		}},
		Title:   "Upgrade " + s.cfg.Name,
		Summary: "e2e: software upgrade to " + s.cfg.Name,
	})
	require.NoError(t, err)
	passed, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	require.Less(t, passed, s.plan, "the proposal passed after its plan height; raise upHaltDelta")

	ctx, cancel := context.WithTimeout(s.ctx, upHaltTimeout)
	defer cancel()
	require.NoError(t, helpers.WaitForHalt(ctx, s.bze, s.plan, upHaltSettle))
	t.Logf("halted at the plan height %d", s.plan)

	s.before = s.snapshot(t, s.plan-1)
}

// 3. The validators restart on the candidate image, which runs the handler at
// the plan height and produces blocks again.
func (s *upgradeSuite) swapAndResume(t *testing.T) {
	require.NoError(t, s.bze.StopAllNodes(s.ctx))
	s.bze.UpgradeVersion(s.ctx, s.docker, chain.ImageRepository, s.cfg.To)
	require.NoError(t, s.bze.StartAllNodes(s.ctx))

	ctx, cancel := context.WithTimeout(s.ctx, upRestartTimeout)
	defer cancel()
	_, err := helpers.WaitForHeight(ctx, s.bze, s.plan+2)
	require.NoError(t, err, "the candidate does not produce blocks after the plan height")

	s.toVer = s.appVersion(t)
	t.Logf("version %s -> %s", s.fromVer, s.toVer)
	require.NotEqual(t, s.fromVer, s.toVer, "bzed version must change")

	applied, err := helpers.Query[struct {
		Height helpers.Int64 `json:"height"`
	}](s.ctx, s.bze, "upgrade", "applied", s.cfg.Name)
	require.NoError(t, err)
	require.Equal(t, s.plan, int64(applied.Height), "upgrade %s applied at the plan height", s.cfg.Name)
}

// 4. The upgrade block changes nothing but what any block changes: inflation
// (and burns), and the day payouts if a day epoch ticked in it.
func (s *upgradeSuite) statePreserved(t *testing.T) {
	s.after = s.snapshot(t, s.plan)
	b, a := s.before, s.after

	// supply: every change is minted or burned by a bank event of the block
	var events []helpers.Event
	for _, typ := range []string{"coinbase", "burn"} {
		e, err := helpers.BlockEvents(s.ctx, s.bze, s.plan, typ)
		require.NoError(t, err)
		events = append(events, e...)
	}
	minted, burned, err := helpers.SupplyChange(events)
	require.NoError(t, err)
	mintModule, err := helpers.ModuleAddress(s.ctx, s.bze, "mint")
	require.NoError(t, err)
	want := b.Supply
	for who, coins := range minted {
		require.Equal(t, mintModule, who, "only the mint module mints in the upgrade block, %s minted %s", who, coins)
		want = want.Add(coins...)
	}
	for who, coins := range burned {
		require.Equal(t, s.burnerAddr, who, "only the burner burns in the upgrade block, %s burned %s", who, coins)
		want = want.Sub(coins...)
	}
	require.Equal(t, want.String(), a.Supply.String(), "supply after = before + minted - burned")
	t.Logf("upgrade block: minted %v, burned %v", minted, burned)

	wantBalances := map[string]string{}
	for name, coins := range b.Balances {
		wantBalances[name] = coins.String()
	}
	if s.check.moves != nil {
		moved := map[string]sdk.Coins{}
		for name, coins := range b.Balances {
			moved[name] = coins
		}
		for _, m := range s.check.moves(s, s.handlerBefore) {
			moved[m.from] = moved[m.from].Sub(m.coin)
			moved[m.to] = moved[m.to].Add(m.coin)
			t.Logf("the handler moves %s from %s to %s", m.coin, m.from, m.to)
		}
		for name, coins := range moved {
			wantBalances[name] = coins.String()
		}
	}
	gotBalances := map[string]string{}
	for name, coins := range a.Balances {
		gotBalances[name] = coins.String()
	}
	require.Equal(t, wantBalances, gotBalances, "account and module balances: before, plus the handler's moves")
	require.Equal(t, b.Pools, a.Pools, "pools: reserves and LP denoms")
	require.Equal(t, b.LpSupply, a.LpSupply, "LP supply")
	require.Equal(t, b.Markets, a.Markets, "markets")
	require.Equal(t, b.Orders, a.Orders, "resting orders")
	require.Equal(t, b.Authorities, a.Authorities, "denom admins")
	require.Equal(t, b.Metadata, a.Metadata, "denom metadata")
	require.Equal(t, b.Branding, a.Branding, "denom branding")
	require.Equal(t, b.Halted, a.Halted, "halted denoms")
	// a handler may add tradebin params, but never changes the ones before it
	for name, raw := range b.TradebinParams {
		require.JSONEq(t, string(raw), string(a.TradebinParams[name]), "tradebin param %s", name)
	}

	ticks := a.Day - b.Day
	require.Contains(t, []int64{0, 1}, ticks, "day epochs ticked in the upgrade block")
	t.Logf("day ticks in the upgrade block: %d, hour %d -> %d", ticks, b.Hour, a.Hour)
	s.requireDayPayouts(t, b.Rewards, a.Rewards, ticks)
	require.Equal(t, b.Rewards.Unlocks, a.Rewards.Unlocks, "pending unlocks (bob's unlocks at hour %d)", s.unlockAt)
}

// 5. The handler-specific checks of upgradeChecks, then a denom reward for
// the next steps if neither the seed nor the handler checks made one.
func (s *upgradeSuite) handlerChecks(t *testing.T) {
	s.check.after(s, t, s.handlerBefore)
	if !s.drActive {
		s.createDenomReward(t)
	}
}

// 6. The first day tick after the upgrade pays the staking reward and the
// denom reward exactly as the previous release would have: the epoch hooks
// are still wired.
func (s *upgradeSuite) nextDayTick(t *testing.T) {
	require.True(t, s.drActive, "a denom reward exists")
	day := s.waitDayTick(t)
	tick := int64(day.CurrentEpochStartHeight)
	require.Greater(t, tick, s.plan)

	s.requireDayPayouts(t, s.rewardsState(t, tick-1), s.rewardsState(t, tick), 1)
	sr, err := helpers.BlockEvents(s.ctx, s.bze, tick, drEventPrefix+"StakingRewardDistributionEvent")
	require.NoError(t, err)
	require.Len(t, helpers.EventsWith(sr, map[string]string{"reward_id": s.srID, "amount": strconv.Itoa(upSrPrize)}), 1,
		"StakingRewardDistributionEvent at height %d", tick)
	dr, err := helpers.BlockEvents(s.ctx, s.bze, tick, drEventPrefix+"DenomRewardDistributionEvent")
	require.NoError(t, err)
	require.Len(t, helpers.EventsWith(dr, map[string]string{"denom": s.stk, "prize_denom": s.prz}), 1,
		"DenomRewardDistributionEvent at height %d", tick)
}

// 7. One tx per custom module succeeds on the candidate, each with its
// effect asserted.
func (s *upgradeSuite) smoke(t *testing.T) {
	admin, alice, bob, carol := s.admin.FormattedAddress(), s.alice.FormattedAddress(), s.bob.FormattedAddress(), s.carol.FormattedAddress()

	t.Run("tokenfactory mint", func(t *testing.T) {
		supply, held := s.supply(t, s.usd), s.balance(t, admin, s.usd)
		s.exec(t, s.admin, "tokenfactory", "mint", coinStr(upSmokeMint, s.usd))
		require.Equal(t, supply.AddRaw(upSmokeMint), s.supply(t, s.usd))
		require.Equal(t, held.AddRaw(upSmokeMint), s.balance(t, admin, s.usd))
	})

	t.Run("AMM swap", func(t *testing.T) {
		pool := s.pool(t, 0, s.usdPool)
		q := quoteSwap(sdkmath.NewInt(upSmokeSwap), pool.reserveQuote(), pool.reserveBase())
		before := s.balance(t, carol, s.usd)
		s.exec(t, s.carol, "tradebin", "multi-swap", s.usdPool, coinStr(upSmokeSwap, chain.Denom), coinStr(1, s.usd))
		require.Equal(t, before.Add(q.out), s.balance(t, carol, s.usd), "swap output")
		after := s.pool(t, 0, s.usdPool)
		require.Equal(t, pool.reserveBase().Sub(q.out), after.reserveBase())
		require.Equal(t, pool.reserveQuote().Add(q.inReserveAdd), after.reserveQuote())
	})

	t.Run("resting order executes", func(t *testing.T) {
		abcBefore := s.balance(t, carol, s.abc)
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.carol, s.txOpts(), map[string]any{
			"@type":      "/bze.tradebin.MsgFillOrders",
			"creator":    carol,
			"market_id":  s.market,
			"order_type": "sell",
			"orders":     []map[string]string{{"price": upSellPrice, "amount": strconv.Itoa(upOrderAmount)}},
		})
		require.NoError(t, err)
		require.Zero(t, res.Code, res.RawLog)
		executed, err := helpers.BlockEvents(s.ctx, s.bze, res.Height, tradebinEventPrefix+"OrderExecutedEvent")
		require.NoError(t, err)
		require.Len(t, helpers.EventsWith(executed, map[string]string{"market_id": s.market, "maker": bob, "taker": carol}), 1,
			"bob's sell order executes against carol")
		require.Equal(t, abcBefore.AddRaw(upOrderAmount), s.balance(t, carol, s.abc))
		var ids []string
		for _, o := range s.userOrders(t, 0, s.bob) {
			ids = append(ids, o.ID)
		}
		require.Equal(t, []string{s.buyID}, ids, "only bob's buy order still rests")
	})

	t.Run("order create", func(t *testing.T) {
		s.exec(t, s.carol, "tradebin", "create-order", "sell", strconv.Itoa(upOrderAmount), upCarolPrice, s.market)
		orders := s.userOrders(t, 0, s.carol)
		require.Len(t, orders, 1, "carol's sell order rests")
		require.Equal(t, "sell", orders[0].OrderType)
	})

	t.Run("staking reward claim", func(t *testing.T) {
		pending := s.srPending(t, 0, s.alice)
		require.True(t, pending.IsPositive(), "alice has staking rewards to claim")
		before := s.balance(t, alice, s.prz)
		res := s.exec(t, s.alice, "rewards", "claim-staking-rewards", s.srID)
		s.requireEvent(t, res, drEventPrefix+"StakingRewardClaimEvent", map[string]string{"reward_id": s.srID, "address": alice, "amount": pending.String()})
		require.Equal(t, before.Add(pending), s.balance(t, alice, s.prz))
	})

	t.Run("denom reward claim", func(t *testing.T) {
		dr := s.denomReward(t, 0)
		require.True(t, dr.AlicePending.IsAllPositive(), "alice has denom rewards to claim")
		before := s.balance(t, alice, s.prz)
		s.exec(t, s.alice, "rewards", "claim-denom-rewards", s.stk)
		require.Equal(t, before.Add(dr.AlicePending.AmountOf(s.prz)), s.balance(t, alice, s.prz))
		require.True(t, s.denomReward(t, 0).AlicePending.IsZero(), "nothing left to claim")
	})

	t.Run("burner fund", func(t *testing.T) {
		before := s.balance(t, s.burnerAddr, chain.Denom)
		res := s.exec(t, s.carol, "burner", "fund-burner", coinStr(upSmokeBurn, chain.Denom))
		got, err := helpers.Transferred(helpers.TxEvents(res, "transfer"), carol, s.burnerAddr, chain.Denom)
		require.NoError(t, err)
		require.Equal(t, sdkmath.NewInt(upSmokeBurn), got)
		require.True(t, s.balance(t, s.burnerAddr, chain.Denom).GTE(before.AddRaw(upSmokeBurn)), "the burner holds the funds")
	})

	if s.features.haltedDenoms {
		t.Run("halted denom still halted", func(t *testing.T) {
			res, err := helpers.Query[struct {
				Halted bool `json:"halted"`
			}](s.ctx, s.bze, "tradebin", "denom-halted", s.hlt)
			require.NoError(t, err)
			require.True(t, res.Halted)
		})
	}
}

// 8. Bob's stake, queued for unlock before the upgrade, is returned at its
// hour epoch by the candidate.
func (s *upgradeSuite) unlockReturned(t *testing.T) {
	ctx, cancel := context.WithTimeout(s.ctx, unlockTimeout)
	defer cancel()
	// AfterEpochEnd(h) runs when the hour epoch moves to h+1
	require.NoError(t, helpers.WaitForEpoch(ctx, s.bze, "hour", s.unlockAt+1))
	require.Equal(t, sdkmath.NewInt(upSrBobStake), s.balance(t, s.bob.FormattedAddress(), s.stk), "bob's stake is back")
	_, found := s.pendingUnlock(t, 0, s.unlockIndex())
	require.False(t, found, "the pending unlock is removed once paid")
}

// requireDayPayouts asserts that between two rewards states exactly ticks day
// payouts happened and nothing else changed: the staking reward's payouts and
// accumulator, and alice's pending denom rewards, move by ticks payouts.
func (s *upgradeSuite) requireDayPayouts(t *testing.T, b, a upRewardsState, ticks int64) {
	t.Helper()
	require.Equal(t, int64(b.SR.Payouts)+ticks, int64(a.SR.Payouts), "staking reward payouts")
	staked := a.SR.staked()
	require.Equal(t, b.SR.staked(), staked, "staked amount")
	perDay := new(big.Int).Quo(scaled(upSrPrize), staked.BigInt())
	wantS := new(big.Int).Add(decScaled(t, b.SR.DistributedStake), new(big.Int).Mul(perDay, big.NewInt(ticks)))
	require.Equal(t, wantS.String(), decScaled(t, a.SR.DistributedStake).String(), "staking reward accumulator")
	bSR, aSR := b.SR, a.SR
	bSR.Payouts, bSR.DistributedStake, aSR.Payouts, aSR.DistributedStake = 0, "", 0, ""
	require.Equal(t, bSR, aSR, "the rest of the staking reward")
	require.Equal(t, b.SRParticipants, a.SRParticipants, "staking reward participants")

	if b.DR == nil {
		require.Nil(t, a.DR, "no denom reward appeared")
		return
	}
	require.NotNil(t, a.DR, "the denom reward is gone")
	require.Equal(t, b.DR.Reward, a.DR.Reward, "denom reward")
	require.Equal(t, b.DR.AliceStake, a.DR.AliceStake, "alice's denom reward stake")
	require.Len(t, a.DR.Schedules, len(b.DR.Schedules))
	for i := range b.DR.Schedules {
		bs, as := b.DR.Schedules[i], a.DR.Schedules[i]
		require.Equal(t, int64(bs.Payouts)+ticks, int64(as.Payouts), "schedule %s payouts", bs.ScheduleID)
		bs.Payouts, as.Payouts = 0, 0
		require.Equal(t, bs, as, "the rest of schedule %s", bs.ScheduleID)
	}
	want := b.DR.AlicePending.Add(sdk.NewInt64Coin(s.prz, upDrDaily*ticks))
	require.Equal(t, want.String(), a.DR.AlicePending.String(), "alice's pending denom rewards")
}

// createDenomReward creates the denom reward of stk with a prz schedule and
// alice's stake, checking the create, prize and schedule fees.
func (s *upgradeSuite) createDenomReward(t *testing.T) {
	t.Helper()
	params := (&denomRewardsSuite{ctx: s.ctx, bze: s.bze}).queryParams(t)
	admin := s.admin.FormattedAddress()
	before := s.balance(t, admin, chain.Denom)

	created := s.exec(t, s.admin, "rewards", "create-denom-reward", s.stk)
	s.requireEvent(t, created, drEventPrefix+"DenomRewardCreateEvent", map[string]string{"denom": s.stk})
	scheduled := s.exec(t, s.admin, "rewards", "create-denom-reward-schedule", s.stk, s.prz, strconv.Itoa(upDrDaily), strconv.Itoa(upDrDuration))
	s.requireEvent(t, scheduled, drEventPrefix+"DenomRewardScheduleCreateEvent", map[string]string{"denom": s.stk, "prize_denom": s.prz})
	fees := params.CreateDenomRewardFee.Amount.Add(params.CreateDenomRewardPrizeFee.Amount).Add(params.AddDenomRewardScheduleFee.Amount)
	require.Equal(t, before.Sub(fees).Sub(s.txFee(t, created)).Sub(s.txFee(t, scheduled)), s.balance(t, admin, chain.Denom),
		"admin pays the create, prize and schedule fees and the tx fees")

	joined := s.exec(t, s.alice, "rewards", "join-denom-reward", s.stk, strconv.Itoa(upDrStake))
	s.requireEvent(t, joined, drEventPrefix+"DenomRewardJoinEvent", map[string]string{"denom": s.stk, "address": s.alice.FormattedAddress()})
	s.drActive = true
}

// v8.2.0: rewards v4 -> v5 sets the Denom Rewards params, txfeecollector
// v2 -> v3 adds blocked_ibc_inbound, and the Noble USDC wind-down runs as on
// mainnet: the network is beezee-1 and its channel-3 carries simd's uusdc, so
// the voucher and the pool's LP denom are the exact mainnet ones. The handler
// blocks uusdc on channel-3 and hands the black hole's USDC.n/BZE LP shares to
// the wind-down admin, touching nothing else. The Denom Rewards messages work.
// v8.3.0: TODO(BZE-152) — x/dao added with its params.
//
// BZE-154 (cosmos-sdk v0.53 + ibc-go v10): the capability and feeibc modules
// are gone (their version entries stay frozen: x/upgrade never deletes one),
// ibc-go v10 migrated core (6 -> 8) and transfer (5 -> 6), no other module
// version moved, and IBC keeps working over the
// channel opened before the upgrade, including the voucher it minted.
//
// BZE-155: the unused x/group and x/nft modules are gone: their stores are
// deleted, their version entries frozen, their CLI queries unknown, and the
// nft module account (left on chain, empty) still refuses funds.
type v830State struct {
	versions map[string]int64
	// voucher is the stake voucher bze minted over the main channel before
	// the upgrade
	voucher string
}

// upV830Deleted are the stores the v8.3.0 store loader deletes: x/capability
// and the ICS-29 fee middleware (both removed with ibc-go v10), x/group and
// x/nft (removed from the app).
var upV830Deleted = []string{"capability", "feeibc", "group", "nft"}

// upV830Frozen are the version entries of the removed modules: x/upgrade never
// deletes one, so they keep their v8.2.0 values.
var upV830Frozen = map[string]int64{"capability": 1, "feeibc": 2, "group": 2, "nft": 1}

const (
	upV830In  = 300_000 // stake simd -> bze over the main channel, before and after
	upV830Out = 100_000 // stake vouchers bze -> simd after the upgrade
)

func (s *upgradeSuite) v830Before(t *testing.T) any {
	st := v830State{versions: s.moduleVersions(t, 0)}
	for name, want := range map[string]int64{"ibc": 6, "transfer": 5, "interchainaccounts": 3} {
		require.Equal(t, want, st.versions[name], "%s module version before", name)
	}
	for name, want := range upV830Frozen {
		require.Equal(t, want, st.versions[name], "%s module version before", name)
	}
	require.True(t, s.supported(t, "ibc-fee", "packets"), "the fee module answers before the upgrade")
	require.True(t, s.supported(t, "group", "groups"), "x/group answers before the upgrade")
	require.True(t, s.supported(t, "nft", "classes"), "x/nft answers before the upgrade")
	s.requireNftAccountBlocked(t)
	for _, store := range upV830Deleted {
		res, err := helpers.StoreQuery(s.ctx, s.bze, store, 0)
		require.NoError(t, err)
		require.Zero(t, res.Code, "the %s store is mounted before the upgrade: %s", store, res.Log)
	}

	s.ibc.transferIn(t, chain.SimdDenom, upV830In).requireAccepted(t)
	st.voucher = helpers.IbcDenom(s.ibc.main.bze, chain.SimdDenom)
	return st
}

func (s *upgradeSuite) v830After(t *testing.T, before any) {
	b := before.(v830State)

	// module versions: the ibc-go v10 migrations ran, everything else as
	// before. x/upgrade never deletes a version entry, so the removed
	// capability, feeibc, group and nft modules keep their last versions; their
	// stores are deleted by the store loader and their CLI is gone (checked below)
	versions := s.moduleVersions(t, s.plan)
	t.Logf("module versions before %v, after %v", b.versions, versions)
	for name, want := range upV830Frozen {
		require.Equal(t, want, versions[name], "%s entry frozen, no migration", name)
	}
	require.Equal(t, int64(8), versions["ibc"], "ibc core migrated 6 -> 8")
	require.Equal(t, int64(6), versions["transfer"], "transfer migrated 5 -> 6")
	for name, v := range b.versions {
		switch name {
		case "ibc", "transfer":
			continue
		}
		require.Equal(t, v, versions[name], "%s module version unchanged", name)
	}
	require.False(t, s.supported(t, "ibc-fee", "packets"), "the fee module is gone from the CLI")
	require.False(t, s.supported(t, "group", "groups"), "x/group is gone from the CLI")
	require.False(t, s.supported(t, "nft", "classes"), "x/nft is gone from the CLI")
	s.requireNftAccountBlocked(t)

	// the store loader deleted the four stores: a raw query fails with the
	// multistore's own error, while a kept store (ibc) still answers
	for _, store := range upV830Deleted {
		res, err := helpers.StoreQuery(s.ctx, s.bze, store, 0)
		require.NoError(t, err)
		require.NotZero(t, res.Code, "the %s store is gone after the upgrade", store)
		require.Contains(t, res.Log, "no such store: "+store, "the %s store is gone after the upgrade", store)
	}
	res, err := helpers.StoreQuery(s.ctx, s.bze, "ibc", 0)
	require.NoError(t, err)
	require.Zero(t, res.Code, "the ibc store still answers: %s", res.Log)

	// the channel opened before the upgrade is still open on both ends
	s.ibc.requireChannelOpen(t, s.ibc.main)

	// the transfer migration turned the voucher's denom trace into a denom
	denom := query[struct {
		Denom struct {
			Base  string `json:"base"`
			Trace []struct {
				PortID    string `json:"port_id"`
				ChannelID string `json:"channel_id"`
			} `json:"trace"`
		} `json:"denom"`
	}](t, s, 0, "ibc-transfer", "denom", b.voucher).Denom
	require.Equal(t, chain.SimdDenom, denom.Base, "voucher base denom")
	require.Len(t, denom.Trace, 1, "voucher trace")
	require.Equal(t, chain.TransferPort, denom.Trace[0].PortID)
	require.Equal(t, s.ibc.main.bze, denom.Trace[0].ChannelID)

	// IBC keeps working on the same channel: stake arrives, vouchers leave
	s.ibc.transferIn(t, chain.SimdDenom, upV830In).requireAccepted(t)
	vouchers := s.ibc.balance(t, s.bze, s.ibc.bzeUser, b.voucher)
	out, err := helpers.SendIbc(s.ctx, s.bze, s.ibc.bzeUser, s.ibc.main.bze, s.ibc.simdUser.FormattedAddress(), sdk.NewInt64Coin(b.voucher, upV830Out))
	require.NoError(t, err)
	_, err = helpers.WaitForRecv(s.ctx, s.ibc.simd, s.ibc.main.simd, out.Sequence)
	require.NoError(t, err)
	_, err = helpers.WaitForAck(s.ctx, s.bze, out)
	require.NoError(t, err)
	require.Equal(t, vouchers.SubRaw(upV830Out), s.ibc.balance(t, s.bze, s.ibc.bzeUser, b.voucher), "stake vouchers left bze")

	// the ICA host still answers
	require.True(t, s.supported(t, "interchain-accounts", "host", "params"), "ICA host params")
}

// upgradeSnapshot is the state that must survive the upgrade, read at one height.
type upgradeSnapshot struct {
	Day, Hour   int64
	Supply      sdk.Coins
	Balances    map[string]sdk.Coins
	Pools       []poolInfo
	LpSupply    map[string]string
	Markets     []map[string]any
	Orders      map[string][]map[string]any
	Authorities map[string]map[string]any
	Metadata    map[string]any
	Branding    *brandingInfo
	Halted      []string
	Rewards     upRewardsState
	// TradebinParams are the tradebin params as the query prints them
	TradebinParams map[string]json.RawMessage
}

// upRewardsState is the rewards state the day payouts move.
type upRewardsState struct {
	SR             srInfo
	SRParticipants map[string][]srParticipant
	Unlocks        []pendingUnlockInfo
	DR             *upDenomReward
}

type upDenomReward struct {
	Reward       denomRewardInfo
	Schedules    []scheduleInfo
	AliceStake   string
	AlicePending sdk.Coins
}

func (s *upgradeSuite) snapshot(t *testing.T, height int64) upgradeSnapshot {
	t.Helper()
	snap := upgradeSnapshot{
		Day:         s.epoch(t, height, "day"),
		Hour:        s.epoch(t, height, "hour"),
		Supply:      s.totalSupply(t, height),
		Balances:    map[string]sdk.Coins{},
		Pools:       s.pools(t, height),
		LpSupply:    map[string]string{},
		Markets:     s.rawList(t, height, "market", "tradebin", "all-markets"),
		Orders:      map[string][]map[string]any{},
		Authorities: map[string]map[string]any{},
		Rewards:     s.rewardsState(t, height),

		TradebinParams: s.params(t, height, "tradebin"),
	}
	for name, addr := range s.accounts {
		snap.Balances[name] = s.balances(t, height, addr)
	}
	for _, p := range snap.Pools {
		snap.LpSupply[p.LpDenom] = snap.Supply.AmountOf(p.LpDenom).String()
	}
	for _, typ := range []string{"buy", "sell"} {
		snap.Orders["aggregated "+typ] = s.rawList(t, height, "list", "tradebin", "market-aggregated-orders", s.market, typ)
	}
	snap.Orders["bob"] = s.rawList(t, height, "list", "tradebin", "user-market-orders", s.bob.FormattedAddress(), s.market)
	denoms := []string{s.usd, s.abc, s.stk, s.prz}
	if s.features.haltedDenoms {
		denoms = append(denoms, s.hlt)
	}
	for _, d := range denoms {
		res := query[struct {
			DenomAuthority map[string]any `json:"denomAuthority"`
		}](t, s, height, "tokenfactory", "denom-authority", d)
		snap.Authorities[d] = res.DenomAuthority
	}
	snap.Metadata = query[struct {
		Metadata map[string]any `json:"metadata"`
	}](t, s, height, "bank", "denom-metadata", s.usd).Metadata
	if s.features.branding {
		snap.Branding = query[struct {
			Branding *brandingInfo `json:"branding"`
		}](t, s, height, "tokenfactory", "denom-branding", s.usd).Branding
		require.NotNil(t, snap.Branding, "usd branding")
	}
	if s.features.haltedDenoms {
		snap.Halted = s.haltedDenoms(t, height)
	}
	return snap
}

func (s *upgradeSuite) rewardsState(t *testing.T, height int64) upRewardsState {
	t.Helper()
	st := upRewardsState{
		SR:             s.stakingReward(t, height),
		SRParticipants: map[string][]srParticipant{},
		Unlocks: query[struct {
			List []pendingUnlockInfo `json:"list"`
		}](t, s, height, "rewards", "all-pending-unlock-participants").List,
	}
	for name, user := range map[string]ibc.Wallet{"alice": s.alice, "bob": s.bob} {
		st.SRParticipants[name] = query[struct {
			List []srParticipant `json:"list"`
		}](t, s, height, "rewards", "staking-reward-participant", user.FormattedAddress()).List
	}
	if s.drActive {
		dr := s.denomReward(t, height)
		st.DR = &dr
	}
	return st
}

func (s *upgradeSuite) denomReward(t *testing.T, height int64) upDenomReward {
	t.Helper()
	p := query[struct {
		Participant struct {
			Amount string `json:"amount"`
		} `json:"participant"`
		Pending sdk.Coins `json:"pending"`
	}](t, s, height, "rewards", "denom-reward-participant", s.alice.FormattedAddress(), s.stk)
	return upDenomReward{
		Reward: query[struct {
			DenomReward denomRewardInfo `json:"denom_reward"`
		}](t, s, height, "rewards", "denom-reward", s.stk).DenomReward,
		Schedules: query[struct {
			List []scheduleInfo `json:"list"`
		}](t, s, height, "rewards", "denom-reward-schedules", s.stk).List,
		AliceStake:   p.Participant.Amount,
		AlicePending: sdk.NewCoins(p.Pending...),
	}
}

// srPending is what the user can claim from the staking reward: stake × (S − S0).
func (s *upgradeSuite) srPending(t *testing.T, height int64, user ibc.Wallet) sdkmath.Int {
	t.Helper()
	sr := s.stakingReward(t, height)
	for _, p := range query[struct {
		List []srParticipant `json:"list"`
	}](t, s, height, "rewards", "staking-reward-participant", user.FormattedAddress()).List {
		if p.RewardID == s.srID {
			delta := new(big.Int).Sub(decScaled(t, sr.DistributedStake), decScaled(t, p.JoinedAt))
			return sdkmath.NewInt(mulFloor(parseInt(p.Amount).Int64(), delta))
		}
	}
	t.Fatalf("%s does not stake in reward %s", user.KeyName(), s.srID)
	return sdkmath.Int{}
}

func (s *upgradeSuite) stakingReward(t *testing.T, height int64) srInfo {
	t.Helper()
	return query[struct {
		StakingReward srInfo `json:"staking_reward"`
	}](t, s, height, "rewards", "get-staking-reward", s.srID).StakingReward
}

func (s *upgradeSuite) allStakingRewards(t *testing.T) []srInfo {
	t.Helper()
	return query[struct {
		List []srInfo `json:"list"`
	}](t, s, 0, "rewards", "all-staking-rewards").List
}

func (s *upgradeSuite) unlockIndex() string {
	return fmt.Sprintf("%d/%s/%s", s.unlockAt, s.srID, s.bob.FormattedAddress())
}

func (s *upgradeSuite) pendingUnlock(t *testing.T, height int64, index string) (pendingUnlockInfo, bool) {
	t.Helper()
	for _, e := range s.rewardsState(t, height).Unlocks {
		if e.Index == index {
			return e, true
		}
	}
	return pendingUnlockInfo{}, false
}

func (s *upgradeSuite) userOrders(t *testing.T, height int64, user ibc.Wallet) []haltOrder {
	t.Helper()
	return query[struct {
		List []haltOrder `json:"list"`
	}](t, s, height, "tradebin", "user-market-orders", user.FormattedAddress(), s.market).List
}

func (s *upgradeSuite) pools(t *testing.T, height int64) []poolInfo {
	t.Helper()
	return query[struct {
		List []poolInfo `json:"list"`
	}](t, s, height, "tradebin", "all-liquidity-pools").List
}

func (s *upgradeSuite) pool(t *testing.T, height int64, id string) poolInfo {
	t.Helper()
	for _, p := range s.pools(t, height) {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("pool %s not found", id)
	return poolInfo{}
}

func (s *upgradeSuite) haltedDenoms(t *testing.T, height int64) []string {
	t.Helper()
	return query[struct {
		Denoms []string `json:"denoms"`
	}](t, s, height, "tradebin", "halted-denoms").Denoms
}

func (s *upgradeSuite) totalSupply(t *testing.T, height int64) sdk.Coins {
	t.Helper()
	res := query[struct {
		Supply     sdk.Coins `json:"supply"`
		Pagination struct {
			NextKey *string `json:"next_key"`
		} `json:"pagination"`
	}](t, s, height, "bank", "total-supply")
	require.True(t, res.Pagination.NextKey == nil || *res.Pagination.NextKey == "", "the supply has more than one page")
	return sdk.NewCoins(res.Supply...)
}

func (s *upgradeSuite) balances(t *testing.T, height int64, addr string) sdk.Coins {
	t.Helper()
	return sdk.NewCoins(query[struct {
		Balances sdk.Coins `json:"balances"`
	}](t, s, height, "bank", "balances", addr).Balances...)
}

func (s *upgradeSuite) moduleVersions(t *testing.T, height int64) map[string]int64 {
	t.Helper()
	res := query[struct {
		ModuleVersions []struct {
			Name    string        `json:"name"`
			Version helpers.Int64 `json:"version"`
		} `json:"module_versions"`
	}](t, s, height, "upgrade", "module-versions")
	out := map[string]int64{}
	for _, m := range res.ModuleVersions {
		out[m.Name] = int64(m.Version)
	}
	require.NotEmpty(t, out, "module versions")
	return out
}

func (s *upgradeSuite) params(t *testing.T, height int64, module string) map[string]json.RawMessage {
	t.Helper()
	return query[struct {
		Params map[string]json.RawMessage `json:"params"`
	}](t, s, height, module, "params").Params
}

func (s *upgradeSuite) epoch(t *testing.T, height int64, identifier string) int64 {
	t.Helper()
	return int64(query[struct {
		CurrentEpoch helpers.Int64 `json:"current_epoch"`
	}](t, s, height, "epochs", "identifier", identifier).CurrentEpoch)
}

// rawList returns the list under key of a query response, as generic JSON
// values so that the previous release's and the candidate's answers compare.
func (s *upgradeSuite) rawList(t *testing.T, height int64, key string, args ...string) []map[string]any {
	t.Helper()
	res := query[map[string]json.RawMessage](t, s, height, args...)
	var list []map[string]any
	if raw, ok := res[key]; ok {
		require.NoError(t, json.Unmarshal(raw, &list), "%v: %s", args, raw)
	}
	return list
}

// query runs a query at height (0: the latest).
func query[T any](t *testing.T, s *upgradeSuite, height int64, args ...string) T {
	t.Helper()
	if height > 0 {
		args = append(args, "--height", strconv.FormatInt(height, 10))
	}
	res, err := helpers.Query[T](s.ctx, s.bze, args...)
	require.NoError(t, err)
	return res
}

// supported reports whether the running release has the query; an unknown
// command means it does not.
// requireNftAccountBlocked checks that the x/nft module account refuses funds,
// before the upgrade as a module account and after it as a blocked address
// kept for the removed module.
func (s *upgradeSuite) requireNftAccountBlocked(t *testing.T) {
	t.Helper()
	nftAddr, err := sdk.Bech32ifyAddressBytes(chain.Bech32Prefix, authtypes.NewModuleAddress("nft"))
	require.NoError(t, err)
	res, err := helpers.ExecTxWith(s.ctx, s.bze, s.admin, s.txOpts(), "bank", "send", s.admin.FormattedAddress(), nftAddr, coinStr(1, chain.Denom))
	require.NoError(t, err)
	require.NotZero(t, res.Code, "sending to the nft module account must fail")
	require.Contains(t, res.RawLog, nftAddr+" is not allowed to receive funds")
}

func (s *upgradeSuite) supported(t *testing.T, args ...string) bool {
	t.Helper()
	_, err := helpers.Query[json.RawMessage](s.ctx, s.bze, args...)
	if err != nil && (strings.Contains(err.Error(), "unknown command") || strings.Contains(err.Error(), "Available Commands")) {
		return false
	}
	require.NoError(t, err, "query %v", args)
	return true
}

func (s *upgradeSuite) hasParam(t *testing.T, module string, names ...string) bool {
	t.Helper()
	params := s.params(t, 0, module)
	for _, n := range names {
		if _, ok := params[n]; ok {
			return true
		}
	}
	return false
}

func (s *upgradeSuite) appVersion(t *testing.T) string {
	t.Helper()
	v, err := helpers.AppVersion(s.ctx, s.bze)
	require.NoError(t, err)
	require.NotEmpty(t, v)
	return v
}

func (s *upgradeSuite) waitDayTick(t *testing.T) helpers.EpochInfo {
	t.Helper()
	return (&denomRewardsSuite{ctx: s.ctx, bze: s.bze}).waitDayTick(t)
}

func (s *upgradeSuite) txOpts() []helpers.TxOption {
	return []helpers.TxOption{helpers.WithGas(upTxGas)}
}

func (s *upgradeSuite) exec(t *testing.T, user ibc.Wallet, args ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(s.ctx, s.bze, user, s.txOpts(), args...)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", args, res.RawLog)
	return res
}

func (s *upgradeSuite) execMsg(t *testing.T, user ibc.Wallet, msg any) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecMsgs(s.ctx, s.bze, user, s.txOpts(), msg)
	require.NoError(t, err)
	require.Zero(t, res.Code, "tx %v failed: %s", msg, res.RawLog)
	return res
}

func (s *upgradeSuite) requireEvent(t *testing.T, res helpers.TxResult, typ string, attrs map[string]string) {
	t.Helper()
	require.Len(t, helpers.EventsWith(helpers.TxEvents(res, typ), attrs), 1, "want one %s with %v in tx %s, got %+v", typ, attrs, res.TxHash, helpers.TxEvents(res, typ))
}

func (s *upgradeSuite) txFee(t *testing.T, res helpers.TxResult) sdkmath.Int {
	t.Helper()
	fee, err := helpers.TxFee(res, chain.Denom)
	require.NoError(t, err)
	return fee
}

func (s *upgradeSuite) balance(t *testing.T, addr, denom string) sdkmath.Int {
	t.Helper()
	b, err := helpers.Balance(s.ctx, s.bze, addr, denom)
	require.NoError(t, err)
	return b
}

func (s *upgradeSuite) supply(t *testing.T, denom string) sdkmath.Int {
	t.Helper()
	return s.totalSupply(t, 0).AmountOf(denom)
}
