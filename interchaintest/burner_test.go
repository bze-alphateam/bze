package interchaintest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

// The burner suite routes coins by kind: ubze and factory denoms are burned,
// LP shares and IBC vouchers without a deep ubze pool go straight to the black
// hole, and a voucher with a deep pool goes to the burner, whose periodic burn
// and raffle cleanup lock it in the black hole too. MoveIbcLockedCoins is the
// only way a locked voucher reaches its pool: at most ibc_move_max_reserve_ratio
// of the pool's voucher reserve per move, once per ibc_move_cooldown_blocks.
// The network starts with periodic_burning_weeks = 1000, so no week tick burns
// before the suite enables the periodic burn by governance (step 3): steps 1
// and 2 fill the burner without racing the 210 s week epochs.
//
// The test pools keep their whole swap fee for the providers (none to the
// burner), so the coins a burn pass leaves in the burner are exactly the fees
// txfeecollector moves in after the burner's EndBlock.
const (
	brAdminFunds = 1_000_000_000_000 // 1,000,000 BZE: three denoms, two pools, liquidity, proposals
	brUserFunds  = 10_000_000_000    // 10,000 BZE each
	brSimdStake  = 50_000_000_000
	brMint       = 100_000_000_000 // brn, rw and rl minted by admin

	brStakeIn = 20_000_000_000 // stake sent from simd to admin
	brOtherIn = 500_000_000    // utoken sent from simd to admin

	brPoolFeeDest = `{"treasury":"0","burner":"0","providers":"1"}`
	// stake/ubze: deep, well above min_native_liquidity_for_module_swap
	brStakePoolVoucher = 10_000_000_000
	brStakePoolBze     = 200_000_000_000
	// utoken/ubze: created below the minimum native liquidity, then deepened
	// to three times its size
	brOtherPoolVoucher = 100_000_000
	brOtherPoolBze     = 50_000_000_000

	brPriceDriftCode = 4018 // tradebin ErrPoolPriceDrift
	// blocks step 10 waits at most for the moved voucher pool's price to come
	// back within moduleMaxPriceDrift (about 56 blocks for a 21% move)
	brCalmBlocks = 150

	brFundBze    = 5_000_000
	brFundBrn    = 3_000_000
	brFundLp     = 1_000_000
	brFundStake  = 100_000_000
	brFundOther  = 10_000_000
	brUserRaffle = 100_000 // rw sent to p1, rl to p2

	brLockedWeeks = 1000
	brTxGas       = 500_000

	// the burner's IBC move params: the genesis defaults, and the values
	// step 10 sets by governance. brNewCooldown is longer than the blocks
	// between the first move and the end of that proposal, so the cooldown
	// is still running when it passes.
	brMoveRatio    = "0.03"
	brMoveCooldown = 1000
	brNewRatio     = "0.001"
	brNewCooldown  = 150
	brMaxCooldown  = 10_000_000 // MaxIbcMoveCooldownBlocks

	brCooldownCode   = 1103 // burner ErrIbcMoveCooldown
	brNoCapacityCode = 1104 // burner ErrIbcMoveNoCapacity

	burnerEventPrefix = "bze.burner."
	burnerCodespace   = "burner"
	sdkCodespace      = "sdk"

	brTickWait   = 5 * time.Minute
	brExpiryWait = 4 * time.Minute
)

// brRaffle is a raffle the suite starts. win always wins (the random number
// is below 1,000,000), loss practically never does (1 in 1,000,000).
type brRaffle struct {
	pot, duration, chances int64
	ratio                  string
	ticket                 int64
}

var (
	brWin  = brRaffle{pot: 1_000_000, duration: 1, chances: 1_000_000, ratio: "0.5", ticket: 1_000}
	brLoss = brRaffle{pot: 2_000_000, duration: 1, chances: 1, ratio: "0.1", ticket: 2_000}
)

// TestBurner runs the burner module end to end on a bze network connected to
// an ibc-go simd chain: FundBurner routing by coin kind (IBC vouchers
// included), the periodic burn on the week tick and its interval param set by
// governance, raffles (start, a win, a loss, refusals, the expiry burn, the
// lock of an IBC pot), MoveIbcLockedCoins with its cap, cooldown and params,
// and the move locks in the genesis export. Every burn is asserted exactly
// against the balance it destroys and the all-burned-coins record, every lock
// against the coins the black hole receives. Subtests run in order and share
// state; the step numbers match the scenario of the suite's design.
func TestBurner(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()

	mnemonic, simdAddr := newSimdAccount(t)
	net := chain.StartIbc(t, ctx, []ibc.WalletAmount{{
		Address: simdAddr,
		Denom:   ibcOtherDenom,
		Amount:  sdkmath.NewInt(ibcOtherFunds),
	}}, chain.WithGenesis(cosmos.NewGenesisKV("app_state.burner.params.periodic_burning_weeks", strconv.Itoa(brLockedWeeks))))
	s := setupBurner(t, ctx, net, mnemonic)

	t.Run("1 fund burner routing", s.fundRouting)
	t.Run("2 ibc vouchers", s.ibcRouting)
	t.Run("3 periodic burn on the week tick", s.periodicBurn)
	t.Run("4 burn interval", s.interval)
	t.Run("5 start raffle", s.startRaffle)
	t.Run("6 join raffle win", s.joinWin)
	t.Run("7 join raffle loss", s.joinLoss)
	t.Run("8 join raffle refusals", s.joinRefusals)
	t.Run("9 raffle expiry", s.raffleExpiry)
	t.Run("10 move ibc locked coins", s.moveLocked)
	t.Run("11 queries", s.queries)
	// last: it stops the network
	t.Run("12 genesis export", s.exportLocks)
}

type burnerSuite struct {
	// the AMM suite's query and tx helpers (balances, events, pools, params)
	*ammSuite
	net chain.IbcNetwork

	admin, p1, p2, p3 ibc.Wallet
	gov, raffleAcc    string

	brn, rw, rl          string // factory denoms of admin
	stake, other         string // simd vouchers
	stakePool, otherPool string
	stakeLp              string

	endAt map[string]int64    // raffle denom → EndAt hour epoch
	burns map[int64]sdk.Coins // height → coins the burner recorded as burned
	// moves is the height of the last successful MoveIbcLockedCoins per denom
	moves map[string]int64
}

func setupBurner(t *testing.T, ctx context.Context, net chain.IbcNetwork, simdMnemonic string) *burnerSuite {
	bze := net.Bze
	s := &burnerSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}, net: net, endAt: map[string]int64{}, burns: map[int64]sdk.Coins{}, moves: map[string]int64{}}

	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "br-admin", sdkmath.NewInt(brAdminFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "br", sdkmath.NewInt(brUserFunds), bze, bze, bze)
	s.p1, s.p2, s.p3 = users[0], users[1], users[2]
	simdUser, err := interchaintest.GetAndFundTestUserWithMnemonic(ctx, "br-simd", simdMnemonic, sdkmath.NewInt(brSimdStake), net.Simd)
	require.NoError(t, err)
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze, net.Simd))

	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)
	for name, addr := range map[string]*string{
		"tradebin":          &s.tradebin,
		"burner":            &s.burner,
		"burner_black_hole": &s.blackHole,
		"burner_raffle":     &s.raffleAcc,
		"txfeecollector_cp": &s.cpFees,
	} {
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	require.Equal(t, brParams{weeks: brLockedWeeks, ratio: brMoveRatio, cooldown: brMoveCooldown}.String(), s.burnerParams(t).String(),
		"genesis burner params: the weeks override and the IBC move defaults")

	factory := "factory/" + s.admin.FormattedAddress() + "/"
	s.brn, s.rw, s.rl = factory+"brn", factory+"rwd", factory+"rlk"
	for _, sub := range []string{"brn", "rwd", "rlk"} {
		s.mustExec(t, s.admin, "tokenfactory", "create-denom", sub)
		s.mustExec(t, s.admin, "tokenfactory", "mint", coinStr(brMint, factory+sub))
	}
	s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), s.p1.FormattedAddress(), coinStr(brUserRaffle, s.rw))
	s.mustExec(t, s.admin, "bank", "send", s.admin.FormattedAddress(), s.p2.FormattedAddress(), coinStr(brUserRaffle, s.rl))

	// vouchers of simd's stake and utoken
	s.stake = helpers.IbcDenom(net.BzeChannel, chain.SimdDenom)
	s.other = helpers.IbcDenom(net.BzeChannel, ibcOtherDenom)
	for _, in := range []sdk.Coin{sdk.NewInt64Coin(chain.SimdDenom, brStakeIn), sdk.NewInt64Coin(ibcOtherDenom, brOtherIn)} {
		send, err := helpers.SendIbc(ctx, net.Simd, simdUser, net.SimdChannel, s.admin.FormattedAddress(), in)
		require.NoError(t, err)
		_, err = helpers.WaitForRecv(ctx, bze, net.BzeChannel, send.Sequence)
		require.NoError(t, err)
	}
	require.Equal(t, sdkmath.NewInt(brStakeIn), s.balance(t, s.admin.FormattedAddress(), s.stake))
	require.Equal(t, sdkmath.NewInt(brOtherIn), s.balance(t, s.admin.FormattedAddress(), s.other))

	// pool ids order the denoms alphabetically: "ibc/…" sorts before "ubze"
	s.stakePool, s.otherPool = s.stake+"_"+chain.Denom, s.other+"_"+chain.Denom
	s.createPool(t, s.stake, brStakePoolVoucher, brStakePoolBze)
	s.stakeLp = s.pool(t, s.stakePool).LpDenom
	// admin's LP shares, funded to the burner in step 1
	s.mustExec(t, s.admin, "tradebin", "add-liquidity", s.stakePool,
		strconv.Itoa(brStakePoolVoucher/10), strconv.Itoa(brStakePoolBze/10), "1")
	require.True(t, s.balance(t, s.admin.FormattedAddress(), s.stakeLp).GTE(sdkmath.NewInt(2*brFundLp)), "admin LP shares")
	return s
}

// 1. One FundBurner tx with ubze, a factory denom and an LP share sends the
// first two to the burner and the LP share to the black hole; so does one tx
// per kind. Malformed, zero and unfunded amounts are refused.
func (s *burnerSuite) fundRouting(t *testing.T) {
	bze, brn, lp := sdk.NewInt64Coin(chain.Denom, brFundBze), sdk.NewInt64Coin(s.brn, brFundBrn), sdk.NewInt64Coin(s.stakeLp, brFundLp)
	for _, c := range []struct {
		name               string
		amount             sdk.Coins
		toBurner, toLocker sdk.Coins
	}{
		{"several coins", sdk.NewCoins(bze, brn, lp), sdk.NewCoins(bze, brn), sdk.NewCoins(lp)},
		{"ubze", sdk.NewCoins(bze), sdk.NewCoins(bze), sdk.NewCoins()},
		{"factory denom", sdk.NewCoins(brn), sdk.NewCoins(brn), sdk.NewCoins()},
		{"LP share", sdk.NewCoins(lp), sdk.NewCoins(), sdk.NewCoins(lp)},
	} {
		t.Run(c.name, func(t *testing.T) {
			s.requireFunded(t, s.admin, c.amount, c.toBurner, c.toLocker)
		})
	}

	for _, c := range []struct {
		name, amount, reason string
		user                 ibc.Wallet
		codespace            string
		code                 uint32
	}{
		{"malformed amount", "abc", "invalid amount", s.p3, sdkCodespace, 10},
		{"zero amount", "0" + chain.Denom, "amounts must be positive", s.p3, sdkCodespace, 10},
		// p3 holds no brn
		{"insufficient funds", coinStr(1, s.brn), "insufficient funds", s.p3, sdkCodespace, 5},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := s.addrBalances(t, c.user.FormattedAddress(), 0)
			res, err := helpers.ExecTxWith(s.ctx, s.bze, c.user, []helpers.TxOption{helpers.WithGas(brTxGas)}, "burner", "fund-burner", c.amount)
			s.requireRefused(t, res, err, c.codespace, c.code, c.reason)
			require.Equal(t, before.Sub(s.paidFee(t, res)...), s.addrBalances(t, c.user.FormattedAddress(), 0), "balances of %s", c.user.KeyName())
		})
	}
}

// 2. A voucher with a deep ubze pool goes to the burner (the periodic burn
// locks it, step 3); a voucher without a pool is locked in the black hole.
func (s *burnerSuite) ibcRouting(t *testing.T) {
	stake, other := sdk.NewInt64Coin(s.stake, brFundStake), sdk.NewInt64Coin(s.other, brFundOther)
	s.requireFunded(t, s.admin, sdk.NewCoins(stake, other), sdk.NewCoins(stake), sdk.NewCoins(other))
}

// 3. MsgUpdateParams is gov-only and refuses a non-positive interval. With the
// interval set to 1 by a proposal, the next week tick burns the burner's whole
// ubze and factory balance (exactly its balance at the block before), locks
// the stake voucher in the black hole (no liquidity is added, the stake pool
// does not change) and records the burn. Only the fees moved in after the
// burn are left in the burner.
func (s *burnerSuite) periodicBurn(t *testing.T) {
	t.Run("signed by a user", func(t *testing.T) {
		before := s.addrBalances(t, s.p3.FormattedAddress(), 0)
		p := s.burnerParams(t)
		p.weeks = 1
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.p3, []helpers.TxOption{helpers.WithGas(brTxGas)}, s.paramsMsg(s.p3.FormattedAddress(), p))
		s.requireRefused(t, res, err, burnerCodespace, 1100, "invalid authority")
		require.Equal(t, before.Sub(s.paidFee(t, res)...), s.addrBalances(t, s.p3.FormattedAddress(), 0), "p3 balances")
	})
	for _, weeks := range []int64{0, -1} {
		t.Run("weeks "+strconv.FormatInt(weeks, 10), func(t *testing.T) {
			p := s.burnerParams(t)
			p.weeks = weeks
			s.requireParamsRefused(t, p, "a non-positive burning interval", "invalid burning period weeks")
		})
	}

	executed := s.setBurningWeeks(t, 1)
	height, _ := s.nextWeekTick(t, executed)
	prev := s.addrBalances(t, s.burner, height-1)
	require.Equal(t, sdkmath.NewInt(brFundStake), prev.AmountOf(s.stake), "stake voucher in the burner")
	require.Equal(t, sdkmath.NewInt(2*brFundBrn), prev.AmountOf(s.brn), "brn in the burner")
	require.True(t, prev.AmountOf(chain.Denom).GTE(sdkmath.NewInt(2*brFundBze)), "ubze in the burner: funds and fees")
	require.True(t, prev.AmountOf(s.stakeLp).IsZero(), "no LP shares in the burner")

	burnable := sdk.NewCoins(sdk.NewCoin(chain.Denom, prev.AmountOf(chain.Denom)), sdk.NewCoin(s.brn, prev.AmountOf(s.brn)))
	s.requireBurned(t, height, s.burner, burnable)
	s.requireCoinsBurnedEvent(t, height, burnable)
	s.burns[height] = burnable
	require.Equal(t, s.supplyAt(t, s.brn, height-1).Sub(prev.AmountOf(s.brn)), s.supplyAt(t, s.brn, height), "brn supply")
	require.Equal(t, s.supplyAt(t, s.stake, height-1), s.supplyAt(t, s.stake, height), "the stake voucher is not burned")

	// the stake voucher is locked: all of it goes to the black hole, none to
	// tradebin, and the stake pool is untouched
	events := s.blockEvents(t, height)
	stake := sdk.NewCoins(sdk.NewInt64Coin(s.stake, brFundStake))
	require.Equal(t, stake, s.transfers(t, events, s.burner, s.blackHole), "the stake voucher, and only it, is locked")
	require.Empty(t, s.transfers(t, events, s.burner, s.tradebin), "nothing goes from the burner to tradebin")
	require.Empty(t, helpers.EventsWith(s.blockTypedEvents(t, height, tradebinEventPrefix+"LiquidityAddedEvent"), map[string]string{"creator": "burner"}),
		"the burner adds no liquidity")
	require.Equal(t, s.poolAt(t, s.stakePool, height-1), s.poolAt(t, s.stakePool, height), "stake pool unchanged")
	require.Equal(t, s.addrBalances(t, s.blackHole, height-1).Add(stake...), s.addrBalances(t, s.blackHole, height), "black hole gains the stake voucher")

	// left in the burner: the fees moved in after the burn
	want := s.transfers(t, events, s.cpFees, s.burner)
	require.Equal(t, want, s.addrBalances(t, s.burner, height), "burner after the burn")
}

// 4. With the interval set to 2, only ticks ending an even week burn: of the
// next two ticks one leaves the burner untouched (a stake voucher funded
// before them stays there) and the other burns the ubze and locks that
// voucher. The interval is then parked at 1000 weeks again.
func (s *burnerSuite) interval(t *testing.T) {
	// funded right after step 3's tick, so both ticks below see it: the next
	// tick of the 1-week interval is a whole week epoch away
	stake := sdk.NewInt64Coin(s.stake, brFundStake)
	s.requireFunded(t, s.admin, sdk.NewCoins(stake), sdk.NewCoins(stake), sdk.NewCoins())
	executed := s.setBurningWeeks(t, 2)
	burned := 0
	for tick := 0; tick < 2; tick++ {
		height, week := s.nextWeekTick(t, executed)
		executed = height
		prev := s.addrBalances(t, s.burner, height-1)
		events := s.blockEvents(t, height)
		if week%2 != 0 {
			t.Logf("week %d ended at height %d: no burn", week, height)
			s.requireBurned(t, height, s.burner, sdk.NewCoins())
			require.Empty(t, s.blockTypedEvents(t, height, burnerEventPrefix+"CoinsBurnedEvent"), "no CoinsBurnedEvent")
			require.Empty(t, s.transfersFrom(t, events, s.burner), "nothing leaves the burner")
			continue
		}
		t.Logf("week %d ended at height %d: burn", week, height)
		burned++
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(chain.Denom, prev.AmountOf(chain.Denom)), stake), prev, "burner before the pass: ubze and the stake voucher")
		burnable := sdk.NewCoins(sdk.NewCoin(chain.Denom, prev.AmountOf(chain.Denom)))
		s.requireBurned(t, height, s.burner, burnable)
		s.requireCoinsBurnedEvent(t, height, burnable)
		s.burns[height] = burnable
		require.Equal(t, sdk.NewCoins(stake), s.transfers(t, events, s.burner, s.blackHole), "the stake voucher is locked")
		require.Empty(t, s.transfers(t, events, s.burner, s.tradebin), "nothing goes from the burner to tradebin")
		require.Equal(t, s.poolAt(t, s.stakePool, height-1), s.poolAt(t, s.stakePool, height), "stake pool unchanged")
		want := s.transfers(t, events, s.cpFees, s.burner)
		require.Equal(t, want, s.addrBalances(t, s.burner, height), "burner after the pass")
	}
	require.Equal(t, 1, burned, "one of two consecutive ticks burns")

	// park the periodic burn again, so the later steps' burn records are
	// the raffles' alone
	s.setBurningWeeks(t, brLockedWeeks)
}

// 5. StartRaffle captures the pot in burner_raffle and records the raffle,
// ending 24 × duration hour epochs after its block; an IBC voucher can be a
// raffle's denom too. Refusals leave everything but the tx fee untouched.
func (s *burnerSuite) startRaffle(t *testing.T) {
	for denom, r := range map[string]brRaffle{s.rw: brWin, s.rl: brLoss, s.stake: brLoss} {
		before := s.addrBalances(t, s.admin.FormattedAddress(), 0)
		res := s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(brTxGas)}, s.startArgs(r, denom)...)
		pot := sdk.NewInt64Coin(denom, r.pot)
		require.Equal(t, sdk.NewCoins(pot), s.transfers(t, res.Events, s.admin.FormattedAddress(), s.raffleAcc), "pot captured")
		require.Equal(t, before.Sub(pot, sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, s.admin.FormattedAddress(), 0), "admin balances")

		s.endAt[denom] = s.hourAt(t, res.Height) + r.duration*24
		got, found := s.raffle(t, denom, 0)
		require.True(t, found, "raffle of %s", denom)
		require.Equal(t, brRaffleInfo{
			Pot: strconv.FormatInt(r.pot, 10), Duration: helpers.Int64(r.duration), Chances: helpers.Int64(r.chances),
			Ratio: r.ratio, EndAt: helpers.Int64(s.endAt[denom]), TicketPrice: strconv.FormatInt(r.ticket, 10),
			Denom: denom, TotalWon: "0",
		}, got)
	}

	ghost := "factory/" + s.admin.FormattedAddress() + "/ghost"
	valid := func(mod func(*brRaffle)) brRaffle {
		r := brWin
		mod(&r)
		return r
	}
	for _, c := range []struct {
		name      string
		user      ibc.Wallet
		raffle    brRaffle
		ratio     string
		denom     string
		codespace string
		code      uint32
		reason    string
	}{
		{"denom without supply", s.admin, brWin, "", ghost, sdkCodespace, 18, "does not exist"},
		{"already running", s.admin, brWin, "", s.rw, sdkCodespace, 18, "raffle already running"},
		// p3 holds no brn
		{"balance", s.p3, brWin, "", s.brn, sdkCodespace, 18, "not enough balance"},
		{"pot below minimum", s.admin, valid(func(r *brRaffle) { r.pot = 99_999 }), "", s.brn, sdkCodespace, 10, "pot must be at least"},
		{"duration 0", s.admin, valid(func(r *brRaffle) { r.duration = 0 }), "", s.brn, sdkCodespace, 18, "duration should be positive"},
		{"duration 181", s.admin, valid(func(r *brRaffle) { r.duration = 181 }), "", s.brn, sdkCodespace, 18, "duration have a value between"},
		{"ratio 0.009", s.admin, brWin, "0.009", s.brn, sdkCodespace, 18, "ratio must have a value between"},
		{"ratio 1.01", s.admin, brWin, "1.01", s.brn, sdkCodespace, 18, "ratio must have a value between"},
		{"chances 0", s.admin, valid(func(r *brRaffle) { r.chances = 0 }), "", s.brn, sdkCodespace, 18, "chances should have a value between"},
		{"chances 1000001", s.admin, valid(func(r *brRaffle) { r.chances = 1_000_001 }), "", s.brn, sdkCodespace, 18, "chances should have a value between"},
		{"ticket price 0", s.admin, valid(func(r *brRaffle) { r.ticket = 0 }), "", s.brn, sdkCodespace, 10, "ticket price is not positive"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.ratio != "" {
				c.raffle.ratio = c.ratio
			}
			before, raffles, module := s.addrBalances(t, c.user.FormattedAddress(), 0), s.raffles(t), s.addrBalances(t, s.raffleAcc, 0)
			res, err := helpers.ExecTxWith(s.ctx, s.bze, c.user, []helpers.TxOption{helpers.WithGas(brTxGas)}, s.startArgs(c.raffle, c.denom)...)
			s.requireRefused(t, res, err, c.codespace, c.code, c.reason)
			require.Equal(t, before.Sub(s.paidFee(t, res)...), s.addrBalances(t, c.user.FormattedAddress(), 0), "balances of %s", c.user.KeyName())
			require.Equal(t, raffles, s.raffles(t), "raffles")
			require.Equal(t, module, s.addrBalances(t, s.raffleAcc, 0), "burner_raffle balances")
		})
	}
}

// 6. Two tickets on the always-winning raffle execute one per block, two and
// three blocks after the join: each pays ratio × the recorded pot (truncated)
// from burner_raffle, lowers the recorded pot by the prize and is listed as a
// winner. The ticket prices stay in burner_raffle without joining the pot.
func (s *burnerSuite) joinWin(t *testing.T) {
	before := s.addrBalances(t, s.p1.FormattedAddress(), 0)
	res := s.mustExec(t, s.p1, "burner", "join-raffle", s.rw, "2")
	tickets := sdk.NewInt64Coin(s.rw, 2*brWin.ticket)
	require.Equal(t, sdk.NewCoins(tickets), s.transfers(t, res.Events, s.p1.FormattedAddress(), s.raffleAcc), "ticket prices")
	s.waitForHeight(t, res.Height+4)

	pot := sdkmath.NewInt(brWin.pot)
	won := sdkmath.ZeroInt()
	var winners []brWinner
	for i := int64(0); i < 2; i++ {
		height := res.Height + 2 + i
		prize := sdkmath.LegacyNewDecFromInt(pot).Mul(sdkmath.LegacyMustNewDecFromStr(brWin.ratio)).TruncateInt()
		events := helpers.EventsWith(s.blockTypedEvents(t, height, burnerEventPrefix+"RaffleWinnerEvent"), map[string]string{"denom": s.rw})
		require.Len(t, events, 1, "one win at height %d", height)
		require.Equal(t, map[string]string{"denom": s.rw, "winner": s.p1.FormattedAddress(), "amount": prize.String()}, withoutMode(events[0].Attributes))
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(s.rw, prize)), s.transfers(t, s.blockEvents(t, height), s.raffleAcc, s.p1.FormattedAddress()), "prize at height %d", height)
		pot, won = pot.Sub(prize), won.Add(prize)
		winners = append(winners, brWinner{Index: strconv.FormatInt(i+1, 10), Denom: s.rw, Amount: prize.String(), Winner: s.p1.FormattedAddress()})
	}
	for _, h := range []int64{res.Height + 1, res.Height + 4} {
		require.Empty(t, s.blockTypedEvents(t, h, burnerEventPrefix+"RaffleWinnerEvent"), "no ticket executes at height %d", h)
	}

	got, found := s.raffle(t, s.rw, 0)
	require.True(t, found)
	require.Equal(t, pot.String(), got.Pot, "recorded pot")
	require.EqualValues(t, 2, got.Winners, "winners")
	require.Equal(t, won.String(), got.TotalWon, "total won")
	require.ElementsMatch(t, winners, s.raffleWinners(t, s.rw), "raffle winners")
	require.Equal(t, sdkmath.NewInt(brWin.pot).Add(tickets.Amount).Sub(won), s.balance(t, s.raffleAcc, s.rw), "burner_raffle rw balance")
	require.Equal(t, before.Add(sdk.NewCoin(s.rw, won)).Sub(tickets, sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, s.p1.FormattedAddress(), 0), "p1 balances")
}

// 7. Three tickets on the practically-never-winning raffle lose one per block;
// each adds its price to the recorded pot and nothing leaves burner_raffle.
func (s *burnerSuite) joinLoss(t *testing.T) {
	before := s.addrBalances(t, s.p2.FormattedAddress(), 0)
	res := s.mustExec(t, s.p2, "burner", "join-raffle", s.rl, "3")
	tickets := sdk.NewInt64Coin(s.rl, 3*brLoss.ticket)
	require.Equal(t, sdk.NewCoins(tickets), s.transfers(t, res.Events, s.p2.FormattedAddress(), s.raffleAcc), "ticket prices")
	s.waitForHeight(t, res.Height+5)

	for i := int64(0); i < 3; i++ {
		height := res.Height + 2 + i
		lost := helpers.EventsWith(s.blockTypedEvents(t, height, burnerEventPrefix+"RaffleLostEvent"), map[string]string{"denom": s.rl, "participant": s.p2.FormattedAddress()})
		require.Len(t, lost, 1, "one lost ticket at height %d", height)
		require.Empty(t, s.transfersFrom(t, s.blockEvents(t, height), s.raffleAcc), "nothing leaves burner_raffle at height %d", height)
	}
	require.Empty(t, s.blockTypedEvents(t, res.Height+5, burnerEventPrefix+"RaffleLostEvent"), "three tickets only")

	got, found := s.raffle(t, s.rl, 0)
	require.True(t, found)
	require.Equal(t, sdkmath.NewInt(brLoss.pot).Add(tickets.Amount).String(), got.Pot, "recorded pot")
	require.Zero(t, got.Winners)
	require.Empty(t, s.raffleWinners(t, s.rl))
	require.Equal(t, sdkmath.NewInt(brLoss.pot).Add(tickets.Amount), s.balance(t, s.raffleAcc, s.rl), "burner_raffle rl balance")
	require.Equal(t, before.Sub(tickets, sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, s.p2.FormattedAddress(), 0), "p2 balances")
}

// 8. Joins are refused for a denom without a raffle, outside 1..50 tickets,
// without the ticket price and within 2 hour epochs of the raffle's end.
func (s *burnerSuite) joinRefusals(t *testing.T) {
	type refusal struct {
		name, denom, tickets string
		user                 ibc.Wallet
		reason               string
	}
	check := func(t *testing.T, c refusal) {
		before, raffles := s.addrBalances(t, c.user.FormattedAddress(), 0), s.raffles(t)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, c.user, []helpers.TxOption{helpers.WithGas(brTxGas)}, "burner", "join-raffle", c.denom, c.tickets)
		s.requireRefused(t, res, err, sdkCodespace, 18, c.reason)
		require.Equal(t, before.Sub(s.paidFee(t, res)...), s.addrBalances(t, c.user.FormattedAddress(), 0), "balances of %s", c.user.KeyName())
		require.Equal(t, raffles, s.raffles(t), "raffles")
	}
	for _, c := range []refusal{
		{"no raffle for the denom", s.brn, "1", s.p1, "raffle not found"},
		{"0 tickets", s.rw, "0", s.p1, "missing tickets"},
		{"51 tickets", s.rw, "51", s.p1, "can not buy more than 50 tickets"},
		// p3 holds no rw
		{"balance", s.rw, "1", s.p3, "not enough balance"},
	} {
		t.Run(c.name, func(t *testing.T) { check(t, c) })
	}

	t.Run("too close to the end", func(t *testing.T) {
		end := s.endAt[s.rw]
		ctx, cancel := context.WithTimeout(s.ctx, brExpiryWait)
		defer cancel()
		require.NoError(t, helpers.WaitForEpoch(ctx, s.bze, "hour", end-2))
		check(t, refusal{"too close", s.rw, "1", s.p1, "too close to expiration"})
		hour, err := helpers.CurrentEpoch(s.ctx, s.bze, "hour")
		require.NoError(t, err)
		require.LessOrEqual(t, hour, end, "the refusal must come before the raffle ends")
	})
}

// 9. The hook of a raffle's EndAt hour epoch removes the raffle and its
// winners and burns burner_raffle's whole balance of the denom: the recorded
// pot and, for the won raffle, the ticket prices that never joined it. Both
// raffles end at the same height when they started in the same hour epoch,
// so the burns are checked per height once both raffles are done.
func (s *burnerSuite) raffleExpiry(t *testing.T) {
	gap := map[string]int64{s.rw: 2 * brWin.ticket, s.rl: 0}
	expired := map[int64]sdk.Coins{}
	for _, denom := range []string{s.rw, s.rl} {
		t.Run(denom[strings.LastIndex(denom, "/")+1:], func(t *testing.T) {
			height := s.epochStartHeight(t, "hour", s.endAt[denom]+1, brExpiryWait)
			prev, found := s.raffle(t, denom, height-1)
			require.True(t, found, "raffle of %s before its end", denom)
			balance := s.addrBalances(t, s.raffleAcc, height-1).AmountOf(denom)
			require.Equal(t, parseInt(prev.Pot).AddRaw(gap[denom]), balance, "burner_raffle holds the recorded pot and the gap")

			expired[height] = expired[height].Add(sdk.NewCoin(denom, balance))
			finished := helpers.EventsWith(s.blockTypedEvents(t, height, burnerEventPrefix+"RaffleFinishedEvent"), map[string]string{"denom": denom})
			require.Len(t, finished, 1, "RaffleFinishedEvent")
			require.Equal(t, s.supplyAt(t, denom, height-1).Sub(balance), s.supplyAt(t, denom, height), "supply of %s", denom)

			_, found = s.raffle(t, denom, height)
			require.False(t, found, "raffle of %s removed", denom)
			require.Empty(t, s.raffleWinners(t, denom), "winners of %s removed", denom)
			require.True(t, s.balance(t, s.raffleAcc, denom).IsZero(), "burner_raffle %s balance", denom)
		})
	}
	for height, burned := range expired {
		s.requireBurned(t, height, s.raffleAcc, burned)
		s.burns[height] = s.burns[height].Add(burned...)
	}

	// the stake voucher raffle's pot can't be burned: its cleanup locks it in
	// the black hole instead of adding it to the deep stake pool. A pot that
	// is only locked burns nothing, so the cleanup records no burn and, as
	// before, emits no RaffleFinishedEvent
	t.Run("ibc voucher pot locked", func(t *testing.T) {
		height := s.epochStartHeight(t, "hour", s.endAt[s.stake]+1, brExpiryWait)
		_, found := s.raffle(t, s.stake, height-1)
		require.True(t, found, "raffle of the stake voucher before its end")
		pot := s.addrBalances(t, s.raffleAcc, height-1).AmountOf(s.stake)
		require.Equal(t, sdkmath.NewInt(brLoss.pot), pot, "burner_raffle holds the pot, nobody joined")

		events := s.blockEvents(t, height)
		locked := sdk.NewCoins(sdk.NewCoin(s.stake, pot))
		require.Equal(t, locked, s.transfers(t, events, s.raffleAcc, s.blackHole), "the pot, and only it, is locked")
		require.True(t, s.transfers(t, events, s.raffleAcc, s.tradebin).IsZero(), "nothing goes from burner_raffle to tradebin")
		require.Equal(t, s.addrBalances(t, s.blackHole, height-1).AmountOf(s.stake).Add(pot), s.addrBalances(t, s.blackHole, height).AmountOf(s.stake), "black hole stake voucher balance")
		// burner_raffle burns at that height only what the rw and rl raffles left
		s.requireBurned(t, height, s.raffleAcc, expired[height])
		require.Equal(t, s.supplyAt(t, s.stake, height-1), s.supplyAt(t, s.stake, height), "the stake voucher is not burned")
		require.Equal(t, s.poolAt(t, s.stakePool, height-1), s.poolAt(t, s.stakePool, height), "stake pool unchanged")
		require.Empty(t, helpers.EventsWith(s.blockTypedEvents(t, height, tradebinEventPrefix+"LiquidityAddedEvent"), map[string]string{"pool_id": s.stakePool}),
			"no liquidity added to the stake pool")
		require.Empty(t, helpers.EventsWith(s.blockTypedEvents(t, height, burnerEventPrefix+"RaffleFinishedEvent"), map[string]string{"denom": s.stake}),
			"no RaffleFinishedEvent for a locked pot")
		require.True(t, s.burns[height].AmountOf(s.stake).IsZero(), "the stake voucher is in no burn record")

		_, found = s.raffle(t, s.stake, height)
		require.False(t, found, "raffle of the stake voucher removed")
		require.True(t, s.balance(t, s.raffleAcc, s.stake).IsZero(), "burner_raffle stake voucher balance")
	})
}

// 10. MoveIbcLockedCoins of the utoken voucher (locked in step 2) is refused
// with no pool (no capacity), with a shallow pool (the swap guard) and while
// the deep pool's price is far from its moving average (tradebin's price
// drift error). None of the refusals starts the cooldown: once the price is
// calm the first move goes ahead at once. A move takes at most
// floor(ibc_move_max_reserve_ratio × the pool's voucher reserve) and leaves
// the rest locked; the same denom is then refused until
// ibc_move_cooldown_blocks have passed, while another denom (the stake
// voucher the periodic burns and the raffle cleanup locked) moves in the same
// window. Both IBC move params are set by governance, which refuses values out
// of their bounds and a message without them, and the next move follows the
// new ones.
func (s *burnerSuite) moveLocked(t *testing.T) {
	locked := s.balance(t, s.blackHole, s.other)
	require.Equal(t, sdkmath.NewInt(brFundOther), locked, "utoken voucher locked in step 2")
	stakeLocked := s.balance(t, s.blackHole, s.stake)
	require.Equal(t, sdkmath.NewInt(2*brFundStake+brLoss.pot), stakeLocked, "stake voucher locked by the burns of steps 3 and 4 and the raffle cleanup of step 9")
	refuse := func(t *testing.T, denom, codespace string, code uint32, reason string) {
		t.Helper()
		before, hole := s.addrBalances(t, s.p1.FormattedAddress(), 0), s.addrBalances(t, s.blackHole, 0)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.p1, []helpers.TxOption{helpers.WithGas(brTxGas)}, "burner", "move-ibc-locked-coins", denom)
		s.requireRefused(t, res, err, codespace, code, reason)
		require.Equal(t, before.Sub(s.paidFee(t, res)...), s.addrBalances(t, s.p1.FormattedAddress(), 0), "p1 balances")
		require.Equal(t, hole, s.addrBalances(t, s.blackHole, 0), "black hole balances")
	}
	cooldownReason := func(denom string, cooldown int64) string {
		return fmt.Sprintf("last move at height %d, next one allowed at height %d", s.moves[denom], s.moves[denom]+cooldown)
	}

	t.Run("no pool", func(t *testing.T) {
		refuse(t, s.other, burnerCodespace, brNoCapacityCode, "no liquidity pool "+s.otherPool)
	})
	t.Run("shallow pool", func(t *testing.T) {
		// the cap (3% of 100,000,000) is positive, but the pool's 50,000 BZE
		// are not above min_native_liquidity_for_module_swap
		s.createPool(t, s.other, brOtherPoolVoucher, brOtherPoolBze)
		refuse(t, s.other, burnerCodespace, 1102, "liquidity availability")
	})

	t.Run("pool price moved", func(t *testing.T) {
		s.mustExec(t, s.admin, "tradebin", "add-liquidity", s.otherPool, strconv.Itoa(2*brOtherPoolVoucher), strconv.Itoa(2*brOtherPoolBze), "1")
		// a swap of a tenth of the ubze reserve moves the price by about 21%,
		// far beyond moduleMaxPriceDrift: the burner adds no liquidity at
		// that price
		in := s.pool(t, s.otherPool).reserveQuote().QuoRaw(10)
		s.mustExec(t, s.admin, "tradebin", "multi-swap", s.otherPool, sdk.NewCoin(chain.Denom, in).String(), coinStr(1, s.other))
		refuse(t, s.other, ammCodespace, brPriceDriftCode, "pool price is too far from its moving average")
		require.NoError(t, helpers.WaitForPoolPriceCalm(s.ctx, s.bze, brCalmBlocks, s.otherPool))
	})

	// no refusal above wrote a lock: the move goes ahead right after them
	var remaining sdkmath.Int
	t.Run("move capped at the reserve ratio", func(t *testing.T) {
		remaining = s.requireMoved(t, s.other, s.otherPool, locked, brMoveRatio, true)
	})
	t.Run("same denom within the cooldown", func(t *testing.T) {
		refuse(t, s.other, burnerCodespace, brCooldownCode, cooldownReason(s.other, brMoveCooldown))
	})
	t.Run("another denom within the cooldown", func(t *testing.T) {
		// the stake pool's voucher reserve is deep enough for the whole locked
		// balance: the move takes all of it
		s.requireMoved(t, s.stake, s.stakePool, stakeLocked, brMoveRatio, false)
	})
	t.Run("nothing left to move", func(t *testing.T) {
		// the balance is checked before the cooldown
		refuse(t, s.stake, burnerCodespace, 1102, "no coins to move")
	})

	t.Run("params refusals", func(t *testing.T) {
		for _, c := range []struct {
			name, reason string
			mod          func(*brParams)
		}{
			{"ratio 0", "ibc move max reserve ratio must be in (0, 1]", func(p *brParams) { p.ratio = "0" }},
			{"ratio above 1", "ibc move max reserve ratio must be in (0, 1]", func(p *brParams) { p.ratio = "1.5" }},
			{"cooldown 0", "ibc move cooldown blocks must be at least 1", func(p *brParams) { p.cooldown = 0 }},
			{"cooldown above the maximum", fmt.Sprintf("ibc move cooldown blocks must be at most %d", brMaxCooldown), func(p *brParams) { p.cooldown = brMaxCooldown + 1 }},
		} {
			t.Run(c.name, func(t *testing.T) {
				p := s.burnerParams(t)
				c.mod(&p)
				s.requireParamsRefused(t, p, c.name, c.reason)
			})
		}
		t.Run("weeks only", func(t *testing.T) {
			// the v8.2.0 shape of the message: the IBC move params are missing
			before := s.burnerParams(t)
			msg := map[string]any{
				"@type":     "/bze.burner.MsgUpdateParams",
				"authority": s.gov,
				"params":    map[string]any{"periodic_burning_weeks": strconv.FormatInt(before.weeks, 10)},
			}
			_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
				Messages: []any{msg}, Title: "invalid burner params", Summary: "e2e: weeks only",
			})
			require.Error(t, err)
			// a missing decimal decodes as nil or as zero, depending on the JSON path
			if !strings.Contains(err.Error(), "ibc move max reserve ratio is required") {
				require.ErrorContains(t, err, "ibc move max reserve ratio must be in (0, 1]")
			}
			require.Equal(t, before.String(), s.burnerParams(t).String(), "burner params unchanged")
		})
	})

	t.Run("params by governance", func(t *testing.T) {
		p := s.burnerParams(t)
		p.ratio, p.cooldown = brNewRatio, brNewCooldown
		s.setBurnerParams(t, p)
		h, err := s.bze.Height(s.ctx)
		require.NoError(t, err)
		require.Less(t, h, s.moves[s.other]+brNewCooldown, "the new cooldown must still run when the proposal passed")
	})
	t.Run("same denom within the new cooldown", func(t *testing.T) {
		refuse(t, s.other, burnerCodespace, brCooldownCode, cooldownReason(s.other, brNewCooldown))
	})
	t.Run("move after the new cooldown, capped at the new ratio", func(t *testing.T) {
		s.waitForHeight(t, s.moves[s.other]+brNewCooldown)
		// the first move's swap moved the pool price
		require.NoError(t, helpers.WaitForPoolPriceCalm(s.ctx, s.bze, brCalmBlocks, s.otherPool))
		s.requireMoved(t, s.other, s.otherPool, remaining, brNewRatio, true)
	})

	for _, c := range []struct {
		name, denom, codespace, reason string
		code                           uint32
	}{
		// pools created since v8.1.1 mint ulp/<hash> LP denoms
		{"LP share", s.stakeLp, sdkCodespace, "cannot move LP shares", 18},
		{"not an IBC denom", s.brn, sdkCodespace, "only ibc denoms are allowed", 18},
		{"unknown IBC denom", "ibc/" + strings.Repeat("0", 64), burnerCodespace, "denom does not exist", 1102},
	} {
		t.Run(c.name, func(t *testing.T) {
			refuse(t, c.denom, c.codespace, c.code, c.reason)
		})
	}
}

// requireMoved sends MoveIbcLockedCoins of denom, of which the black hole
// holds locked, and asserts the move: exactly min(locked, floor(ratio × the
// pool's voucher reserve)) goes from the black hole to tradebin and into the
// pool, the LP shares tradebin mints stay locked, the ubze it returns goes to
// the burner and the rest of the voucher stays locked. capped says whether
// the cap must bite (less than locked moves). It records the move height and
// returns the voucher still locked.
func (s *burnerSuite) requireMoved(t *testing.T, denom, poolID string, locked sdkmath.Int, ratio string, capped bool) sdkmath.Int {
	t.Helper()
	before := s.addrBalances(t, s.p1.FormattedAddress(), 0)
	res := s.mustExecWith(t, s.p1, []helpers.TxOption{helpers.WithGas(brTxGas)}, "burner", "move-ibc-locked-coins", denom)
	pool, hole := s.poolAt(t, poolID, res.Height-1), s.addrBalances(t, s.blackHole, res.Height-1)
	require.Equal(t, denom, pool.Base, "the voucher is the pool's base")
	require.Equal(t, locked, hole.AmountOf(denom), "locked %s before the move", denom)

	limit := sdkmath.LegacyMustNewDecFromStr(ratio).MulInt(pool.reserveBase()).TruncateInt()
	require.True(t, limit.IsPositive(), "the cap of %s is positive", denom)
	if capped {
		require.True(t, limit.LT(locked), "the cap %s bites on the %s locked", limit, locked)
	} else {
		require.True(t, locked.LTE(limit), "the %s locked fit under the cap %s", locked, limit)
	}
	moved := sdkmath.MinInt(locked, limit)

	require.Equal(t, sdk.NewCoins(sdk.NewCoin(denom, moved)), s.transfers(t, res.Events, s.blackHole, s.tradebin), "locked voucher to tradebin")
	returned := s.transfers(t, res.Events, s.tradebin, s.blackHole)
	refund := returned.AmountOf(chain.Denom)
	added := helpers.EventsWith(helpers.TxEvents(res, tradebinEventPrefix+"LiquidityAddedEvent"), map[string]string{"creator": "burner_black_hole", "pool_id": poolID})
	require.Len(t, added, 1, "liquidity added by the black hole")
	minted := parseInt(added[0].Attributes["mintedAmount"])
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(pool.LpDenom, minted), sdk.NewCoin(chain.Denom, refund)), returned, "tradebin returns LP and ubze only")
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(chain.Denom, refund)), s.transfers(t, res.Events, s.blackHole, s.burner), "returned ubze to the burner")
	// the periodic burn is parked, so the burner keeps it (fees may come in too)
	burnerBefore := s.addrBalances(t, s.burner, res.Height-1).AmountOf(chain.Denom)
	require.True(t, s.addrBalances(t, s.burner, res.Height).AmountOf(chain.Denom).GTE(burnerBefore.Add(refund)), "the burner holds the returned ubze")

	after := s.poolAt(t, poolID, res.Height)
	require.Equal(t, pool.reserveBase().Add(moved), after.reserveBase(), "voucher reserve")
	require.Equal(t, pool.reserveQuote().Sub(refund), after.reserveQuote(), "ubze reserve")
	wantHole := hole.Sub(sdk.NewCoin(denom, moved)).Add(sdk.NewCoin(pool.LpDenom, minted))
	require.Equal(t, wantHole, s.addrBalances(t, s.blackHole, res.Height), "black hole balances: the rest of the voucher stays locked")
	require.Equal(t, before.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, s.p1.FormattedAddress(), 0), "p1 pays only the fee")

	s.moves[denom] = res.Height
	return locked.Sub(moved)
}

// 11. The queries return what the steps produced: the interval parked again
// at the end of step 4, the IBC move params set in step 10, no running raffle or winner left, and one all-burned-coins record per burn
// height with exactly the coins burned there.
func (s *burnerSuite) queries(t *testing.T) {
	require.Equal(t, brParams{weeks: brLockedWeeks, ratio: brNewRatio, cooldown: brNewCooldown}.String(), s.burnerParams(t).String(),
		"burner params: the interval parked in step 4, the IBC move params of step 10")
	require.Empty(t, s.raffles(t), "raffles")
	require.Empty(t, s.raffleWinners(t, s.rw), "winners of rw")

	res, err := helpers.Query[struct {
		BurnedCoins []struct {
			Burned string `json:"burned"`
			Height string `json:"height"`
		} `json:"burnedCoins"`
	}](s.ctx, s.bze, "burner", "all-burned-coins")
	require.NoError(t, err)
	got := map[int64]sdk.Coins{}
	for _, b := range res.BurnedCoins {
		h, err := strconv.ParseInt(b.Height, 10, 64)
		require.NoError(t, err)
		coins, err := sdk.ParseCoinsNormalized(b.Burned)
		require.NoError(t, err)
		got[h] = coins
	}
	require.Equal(t, s.burns, got, "all-burned-coins")
}

// 12. The genesis export carries the burner params and one move lock per
// denom moved in step 10, at the height of its last successful move. It
// stops the network, so it runs last.
func (s *burnerSuite) exportLocks(t *testing.T) {
	require.Len(t, s.moves, 2, "step 10 moved the utoken and the stake vouchers")
	node := s.bze.Validators[0]
	require.NoError(t, s.bze.StopAllNodes(s.ctx))
	exported, err := node.ExportState(s.ctx, -1)
	require.NoError(t, err)

	var doc struct {
		AppState struct {
			Burner struct {
				Params struct {
					PeriodicBurningWeeks   helpers.Int64 `json:"periodic_burning_weeks"`
					IbcMoveMaxReserveRatio string        `json:"ibc_move_max_reserve_ratio"`
					IbcMoveCooldownBlocks  helpers.Int64 `json:"ibc_move_cooldown_blocks"`
				} `json:"params"`
				IbcMoveLocks []struct {
					Denom          string        `json:"denom"`
					LastMoveHeight helpers.Int64 `json:"last_move_height"`
				} `json:"ibc_move_locks"`
			} `json:"burner"`
		} `json:"app_state"`
	}
	require.NoError(t, json.Unmarshal([]byte(exported), &doc))
	g := doc.AppState.Burner
	ratio, err := helpers.ParseDec(g.Params.IbcMoveMaxReserveRatio)
	require.NoError(t, err)
	require.Equal(t, brParams{weeks: brLockedWeeks, ratio: brNewRatio, cooldown: brNewCooldown}.String(),
		brParams{weeks: int64(g.Params.PeriodicBurningWeeks), ratio: ratio.String(), cooldown: uint64(g.Params.IbcMoveCooldownBlocks)}.String(), //nolint:gosec // a validated block count
		"exported burner params")
	got := map[string]int64{}
	for _, l := range g.IbcMoveLocks {
		got[l.Denom] = int64(l.LastMoveHeight)
	}
	require.Len(t, g.IbcMoveLocks, len(got), "one lock per denom")
	require.Equal(t, s.moves, got, "exported ibc_move_locks")
}

// requireFunded sends amount to the burner from user and asserts the routing:
// toBurner to the burner, toLocker to the black hole, the FundBurnerEvent and
// user's balances.
func (s *burnerSuite) requireFunded(t *testing.T, user ibc.Wallet, amount, toBurner, toLocker sdk.Coins) {
	t.Helper()
	addr := user.FormattedAddress()
	before := s.addrBalances(t, addr, 0)
	res := s.mustExec(t, user, "burner", "fund-burner", amount.String())
	require.Equal(t, toBurner, s.transfers(t, res.Events, addr, s.burner), "to the burner")
	require.Equal(t, toLocker, s.transfers(t, res.Events, addr, s.blackHole), "to the black hole")
	events := helpers.TxEvents(res, burnerEventPrefix+"FundBurnerEvent")
	require.Len(t, events, 1, "FundBurnerEvent")
	require.Equal(t, map[string]string{"from": addr, "amount": amount.String()}, withoutMode(events[0].Attributes))
	require.Equal(t, before.Sub(amount...).Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, addr, 0), "balances of %s", user.KeyName())
}

// requireBurned asserts the coins burned by addr in the finalize-block events
// of height.
func (s *burnerSuite) requireBurned(t *testing.T, height int64, addr string, want sdk.Coins) {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, height, "burn")
	require.NoError(t, err)
	_, burned, err := helpers.SupplyChange(events)
	require.NoError(t, err)
	require.Equal(t, want.String(), burned[addr].String(), "burned by %s at height %d", addr, height)
}

func (s *burnerSuite) requireCoinsBurnedEvent(t *testing.T, height int64, want sdk.Coins) {
	t.Helper()
	events := s.blockTypedEvents(t, height, burnerEventPrefix+"CoinsBurnedEvent")
	require.Len(t, events, 1, "CoinsBurnedEvent at height %d", height)
	got, err := sdk.ParseCoinsNormalized(events[0].Attributes["burned"])
	require.NoError(t, err)
	require.Equal(t, want, got, "CoinsBurnedEvent")
}

func (s *burnerSuite) createPool(t *testing.T, voucher string, voucherAmount, bzeAmount int64) {
	t.Helper()
	s.mustExecWith(t, s.admin, []helpers.TxOption{helpers.WithGas(brTxGas)},
		"tradebin", "create-liquidity-pool", voucher, chain.Denom, ammPoolFee, brPoolFeeDest, "false",
		strconv.FormatInt(voucherAmount, 10), strconv.FormatInt(bzeAmount, 10))
	pool := s.pool(t, voucher+"_"+chain.Denom)
	require.Equal(t, voucher, pool.Base, "the voucher is the pool's base")
	require.Equal(t, sdkmath.NewInt(voucherAmount), pool.reserveBase())
	require.Equal(t, sdkmath.NewInt(bzeAmount), pool.reserveQuote())
}

func (s *burnerSuite) startArgs(r brRaffle, denom string) []string {
	return []string{"burner", "start-raffle", strconv.FormatInt(r.pot, 10), strconv.FormatInt(r.duration, 10),
		strconv.FormatInt(r.chances, 10), r.ratio, strconv.FormatInt(r.ticket, 10), denom}
}

// brParams are the burner params; ratio is a decimal ("0.03").
type brParams struct {
	weeks    int64
	ratio    string
	cooldown uint64
}

// String is the params with the ratio in its canonical 18-decimal form, so
// two params compare whatever form the ratio was written in.
func (p brParams) String() string {
	return fmt.Sprintf("periodic_burning_weeks %d, ibc_move_max_reserve_ratio %s, ibc_move_cooldown_blocks %d",
		p.weeks, sdkmath.LegacyMustNewDecFromStr(p.ratio).String(), p.cooldown)
}

// paramsMsg is a burner MsgUpdateParams carrying all three params, the ratio
// as a decimal (gogoproto JSON).
func (s *burnerSuite) paramsMsg(authority string, p brParams) map[string]any {
	return map[string]any{
		"@type":     "/bze.burner.MsgUpdateParams",
		"authority": authority,
		"params": map[string]any{
			"periodic_burning_weeks":     strconv.FormatInt(p.weeks, 10),
			"ibc_move_max_reserve_ratio": sdkmath.LegacyMustNewDecFromStr(p.ratio).String(),
			"ibc_move_cooldown_blocks":   strconv.FormatUint(p.cooldown, 10),
		},
	}
}

// setBurningWeeks passes a proposal setting periodic_burning_weeks, keeping
// the other params, and returns the height of the block whose gov EndBlock
// executed it: a week tick at that height still ran with the old interval.
func (s *burnerSuite) setBurningWeeks(t *testing.T, weeks int64) int64 {
	t.Helper()
	p := s.burnerParams(t)
	p.weeks = weeks
	return s.setBurnerParams(t, p)
}

// setBurnerParams passes a proposal setting the burner params to p, checks
// the chain holds them and returns the height that executed it.
func (s *burnerSuite) setBurnerParams(t *testing.T, p brParams) int64 {
	t.Helper()
	from, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	id, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: []any{s.paramsMsg(s.gov, p)},
		Title:    "burner params",
		Summary:  "e2e: " + p.String(),
	})
	require.NoError(t, err)
	require.Equal(t, p.String(), s.burnerParams(t).String(), "burner params after the proposal")

	to, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	attrs := map[string]string{"proposal_id": strconv.FormatUint(id, 10), "proposal_result": "proposal_passed"}
	for h := from; h <= to; h++ {
		if len(helpers.EventsWith(s.blockTypedEvents(t, h, "active_proposal"), attrs)) > 0 {
			return h
		}
	}
	require.FailNow(t, "no active_proposal event", "proposal %d between heights %d and %d", id, from, to)
	return 0
}

// requireParamsRefused requires a proposal setting the burner params to p to
// be refused with reason, and the params to stay as they were.
func (s *burnerSuite) requireParamsRefused(t *testing.T, p brParams, name, reason string) {
	t.Helper()
	before := s.burnerParams(t)
	_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: []any{s.paramsMsg(s.gov, p)},
		Title:    "invalid burner params",
		Summary:  "e2e: " + name,
	})
	require.ErrorContains(t, err, reason)
	require.Equal(t, before.String(), s.burnerParams(t).String(), "burner params unchanged")
}

// burnerParams queries the burner params. The CLI prints the ratio either as
// a decimal or as the 18-decimal integer of a LegacyDec.
func (s *burnerSuite) burnerParams(t *testing.T) brParams {
	t.Helper()
	res, err := helpers.Query[struct {
		Params struct {
			PeriodicBurningWeeks   helpers.Int64 `json:"periodic_burning_weeks"`
			IbcMoveMaxReserveRatio string        `json:"ibc_move_max_reserve_ratio"`
			IbcMoveCooldownBlocks  helpers.Int64 `json:"ibc_move_cooldown_blocks"`
		} `json:"params"`
	}](s.ctx, s.bze, "burner", "params")
	require.NoError(t, err)
	ratio, err := helpers.ParseDec(res.Params.IbcMoveMaxReserveRatio)
	require.NoError(t, err)
	return brParams{
		weeks:    int64(res.Params.PeriodicBurningWeeks),
		ratio:    ratio.String(),
		cooldown: uint64(res.Params.IbcMoveCooldownBlocks), //nolint:gosec // a validated block count
	}
}

// nextWeekTick returns the first week tick after height: the start block of a
// week epoch, whose BeginBlock ran the AfterEpochEnd hooks, and the number of
// the week that ended there.
func (s *burnerSuite) nextWeekTick(t *testing.T, after int64) (int64, int64) {
	t.Helper()
	week, err := helpers.Epoch(s.ctx, s.bze, "week")
	require.NoError(t, err)
	next := int64(week.CurrentEpoch)
	if int64(week.CurrentEpochStartHeight) <= after {
		next++
	}
	return s.epochStartHeight(t, "week", next, brTickWait), next - 1
}

// epochStartHeight waits for epoch n of identifier and returns its start
// block, the first height whose state is in epoch n.
func (s *burnerSuite) epochStartHeight(t *testing.T, identifier string, n int64, wait time.Duration) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(s.ctx, wait)
	defer cancel()
	require.NoError(t, helpers.WaitForEpoch(ctx, s.bze, identifier, n))
	info, err := helpers.Epoch(s.ctx, s.bze, identifier)
	require.NoError(t, err)
	h := int64(info.CurrentEpochStartHeight)
	for s.epochAt(t, identifier, h-1) >= n {
		h--
	}
	require.Equal(t, n, s.epochAt(t, identifier, h), "%s epoch at its start block %d", identifier, h)
	return h
}

func (s *burnerSuite) epochAt(t *testing.T, identifier string, height int64) int64 {
	t.Helper()
	res, err := helpers.Query[struct {
		CurrentEpoch helpers.Int64 `json:"current_epoch"`
	}](s.ctx, s.bze, "epochs", "identifier", identifier, "--height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	return int64(res.CurrentEpoch)
}

func (s *burnerSuite) hourAt(t *testing.T, height int64) int64 {
	t.Helper()
	return s.epochAt(t, "hour", height)
}

func (s *burnerSuite) waitForHeight(t *testing.T, h int64) {
	t.Helper()
	_, err := helpers.WaitForHeight(s.ctx, s.bze, h)
	require.NoError(t, err)
}

type brRaffleInfo struct {
	Pot         string        `json:"pot"`
	Duration    helpers.Int64 `json:"duration"`
	Chances     helpers.Int64 `json:"chances"`
	Ratio       string        `json:"ratio"`
	EndAt       helpers.Int64 `json:"end_at"`
	Winners     helpers.Int64 `json:"winners"`
	TicketPrice string        `json:"ticket_price"`
	Denom       string        `json:"denom"`
	TotalWon    string        `json:"total_won"`
}

type brWinner struct {
	Index  string `json:"index"`
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
	Winner string `json:"winner"`
}

// raffles returns the running raffles, at height (0 for the latest).
func (s *burnerSuite) rafflesAt(t *testing.T, height int64) []brRaffleInfo {
	t.Helper()
	args := []string{"burner", "raffles"}
	if height > 0 {
		args = append(args, "--height", strconv.FormatInt(height, 10))
	}
	res, err := helpers.Query[struct {
		List []brRaffleInfo `json:"list"`
	}](s.ctx, s.bze, args...)
	require.NoError(t, err)
	return res.List
}

func (s *burnerSuite) raffles(t *testing.T) []brRaffleInfo {
	t.Helper()
	return s.rafflesAt(t, 0)
}

func (s *burnerSuite) raffle(t *testing.T, denom string, height int64) (brRaffleInfo, bool) {
	t.Helper()
	for _, r := range s.rafflesAt(t, height) {
		if r.Denom == denom {
			return r, true
		}
	}
	return brRaffleInfo{}, false
}

func (s *burnerSuite) raffleWinners(t *testing.T, denom string) []brWinner {
	t.Helper()
	res, err := helpers.Query[struct {
		List []brWinner `json:"list"`
	}](s.ctx, s.bze, "burner", "raffle-winners", denom)
	require.NoError(t, err)
	return res.List
}

// addrBalances returns every balance of addr at height (0 for the latest).
func (s *burnerSuite) addrBalances(t *testing.T, addr string, height int64) sdk.Coins {
	t.Helper()
	args := []string{"bank", "balances", addr}
	if height > 0 {
		args = append(args, "--height", strconv.FormatInt(height, 10))
	}
	res, err := helpers.Query[struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}](s.ctx, s.bze, args...)
	require.NoError(t, err)
	out := sdk.NewCoins()
	for _, b := range res.Balances {
		out = out.Add(sdk.NewCoin(b.Denom, parseInt(b.Amount)))
	}
	return out
}

func (s *burnerSuite) supplyAt(t *testing.T, denom string, height int64) sdkmath.Int {
	t.Helper()
	res, err := helpers.Query[struct {
		Amount struct {
			Amount string `json:"amount"`
		} `json:"amount"`
	}](s.ctx, s.bze, "bank", "total-supply-of", denom, "--height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	return parseInt(res.Amount.Amount)
}

func (s *burnerSuite) poolAt(t *testing.T, id string, height int64) poolInfo {
	t.Helper()
	res, err := helpers.Query[struct {
		Pool poolInfo `json:"pool"`
	}](s.ctx, s.bze, "tradebin", "liquidity-pool", id, "--height", strconv.FormatInt(height, 10))
	require.NoError(t, err)
	require.Equal(t, id, res.Pool.ID, "pool %s not found at height %d", id, height)
	return res.Pool
}

func (s *burnerSuite) blockTypedEvents(t *testing.T, height int64, eventType string) []helpers.Event {
	t.Helper()
	events, err := helpers.BlockEvents(s.ctx, s.bze, height, eventType)
	require.NoError(t, err)
	return events
}

// transfers sums every coin moved from one address to another by the bank
// transfer events in events.
func (s *burnerSuite) transfers(t *testing.T, events []helpers.Event, from, to string) sdk.Coins {
	t.Helper()
	out := sdk.NewCoins()
	for _, e := range helpers.EventsWith(filterType(events, "transfer"), map[string]string{"sender": from, "recipient": to}) {
		coins, err := sdk.ParseCoinsNormalized(e.Attributes["amount"])
		require.NoError(t, err)
		out = out.Add(coins...)
	}
	return out
}

// transfersFrom returns the transfer events sent by from.
func (s *burnerSuite) transfersFrom(t *testing.T, events []helpers.Event, from string) []helpers.Event {
	t.Helper()
	return helpers.EventsWith(filterType(events, "transfer"), map[string]string{"sender": from})
}

// requireRefused asserts a refusal with codespace, code and the chain's message.
func (s *burnerSuite) requireRefused(t *testing.T, res helpers.TxResult, err error, codespace string, code uint32, reason string) {
	t.Helper()
	got, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, got, reason)
	require.EqualValues(t, code, res.Code, got)
	require.Equal(t, codespace, res.Codespace, got)
}

// paidFee is the tx fee of a refused tx: none when CheckTx refused it.
func (s *burnerSuite) paidFee(t *testing.T, res helpers.TxResult) sdk.Coins {
	t.Helper()
	if res.Height == 0 {
		return nil
	}
	return sdk.NewCoins(sdk.NewCoin(chain.Denom, s.txFee(t, res)))
}

// withoutMode drops the "mode" attribute the SDK adds to finalize-block
// events, so typed event attributes compare whole.
func withoutMode(attrs map[string]string) map[string]string {
	out := make(map[string]string, len(attrs))
	for k, v := range attrs {
		if k != "mode" && k != "msg_index" {
			out[k] = v
		}
	}
	return out
}
