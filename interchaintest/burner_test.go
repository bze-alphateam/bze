package interchaintest_test

import (
	"context"
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
// LP shares are locked in the black hole, IBC vouchers with a deep ubze pool
// are added to that pool as liquidity and vouchers without one are locked.
// The network starts with periodic_burning_weeks = 1000, so no week tick burns
// before the suite enables the periodic burn by governance (step 3): steps 1
// and 2 fill the burner without racing the 210 s week epochs.
//
// The test pools keep their whole swap fee for the providers (none to the
// burner), so the coins a burn pass leaves in the burner are exactly what
// tradebin refunds plus the fees txfeecollector moves in after the burner's
// EndBlock.
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

	brFundBze    = 5_000_000
	brFundBrn    = 3_000_000
	brFundLp     = 1_000_000
	brFundStake  = 100_000_000
	brFundOther  = 10_000_000
	brUserRaffle = 100_000 // rw sent to p1, rl to p2

	brLockedWeeks = 1000
	brTxGas       = 500_000

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
// governance, raffles (start, a win, a loss, refusals, the expiry burn) and
// MoveIbcLockedCoins. Every burn is asserted exactly against the balance it
// destroys and the all-burned-coins record. Subtests run in order and share
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
}

func setupBurner(t *testing.T, ctx context.Context, net chain.IbcNetwork, simdMnemonic string) *burnerSuite {
	bze := net.Bze
	s := &burnerSuite{ammSuite: &ammSuite{ctx: ctx, bze: bze}, net: net, endAt: map[string]int64{}, burns: map[int64]sdk.Coins{}}

	s.admin = interchaintest.GetAndFundTestUsers(t, ctx, "br-admin", sdkmath.NewInt(brAdminFunds), bze)[0]
	users := interchaintest.GetAndFundTestUsers(t, ctx, "br", sdkmath.NewInt(brUserFunds), bze, bze, bze)
	s.p1, s.p2, s.p3 = users[0], users[1], users[2]
	simdUser, err := interchaintest.GetAndFundTestUserWithMnemonic(ctx, "br-simd", simdMnemonic, sdkmath.NewInt(brSimdStake), net.Simd)
	require.NoError(t, err)
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze, net.Simd))

	s.gov, err = helpers.GovAuthority(ctx, bze)
	require.NoError(t, err)
	for name, addr := range map[string]*string{
		"tradebin":              &s.tradebin,
		"burner":                &s.burner,
		"burner_black_hole":     &s.blackHole,
		"burner_raffle":         &s.raffleAcc,
		"txfeecollector_burner": &s.burnerFees,
		"txfeecollector_cp":     &s.cpFees,
	} {
		*addr, err = helpers.ModuleAddress(ctx, bze, name)
		require.NoError(t, err)
	}
	require.Equal(t, int64(brLockedWeeks), s.burningWeeks(t), "genesis periodic_burning_weeks")

	factory := "factory/" + s.admin.FormattedAddress() + "/"
	s.brn, s.rw, s.rl = factory+"brn", factory+"rw", factory+"rl"
	for _, sub := range []string{"brn", "rw", "rl"} {
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

// 2. A voucher with a deep ubze pool goes to the burner (it can be exchanged
// for ubze); a voucher without a pool is locked in the black hole.
func (s *burnerSuite) ibcRouting(t *testing.T) {
	stake, other := sdk.NewInt64Coin(s.stake, brFundStake), sdk.NewInt64Coin(s.other, brFundOther)
	s.requireFunded(t, s.admin, sdk.NewCoins(stake, other), sdk.NewCoins(stake), sdk.NewCoins(other))
}

// 3. MsgUpdateParams is gov-only and refuses a non-positive interval. With the
// interval set to 1 by a proposal, the next week tick burns the burner's whole
// ubze and factory balance (exactly its balance at the block before), adds the
// stake voucher to the stake pool as liquidity and records the burn. The LP
// shares and the ubze tradebin returns stay in the burner for the next pass.
func (s *burnerSuite) periodicBurn(t *testing.T) {
	t.Run("signed by a user", func(t *testing.T) {
		before := s.addrBalances(t, s.p3.FormattedAddress(), 0)
		res, err := helpers.ExecMsgs(s.ctx, s.bze, s.p3, []helpers.TxOption{helpers.WithGas(brTxGas)}, s.paramsMsg(s.p3.FormattedAddress(), 1))
		s.requireRefused(t, res, err, burnerCodespace, 1100, "invalid authority")
		require.Equal(t, before.Sub(s.paidFee(t, res)...), s.addrBalances(t, s.p3.FormattedAddress(), 0), "p3 balances")
	})
	for _, weeks := range []int64{0, -1} {
		t.Run("weeks "+strconv.FormatInt(weeks, 10), func(t *testing.T) {
			_, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
				Messages: []any{s.paramsMsg(s.gov, weeks)},
				Title:    "invalid burner params",
				Summary:  "e2e: a non-positive burning interval",
			})
			require.ErrorContains(t, err, "invalid burning period weeks")
			require.Equal(t, int64(brLockedWeeks), s.burningWeeks(t))
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

	// the stake voucher went into the stake pool: the voucher reserve grows by
	// all of it and the ubze reserve shrinks by what tradebin returned
	events := s.blockEvents(t, height)
	returned := s.transfers(t, events, s.tradebin, s.burner)
	added := helpers.EventsWith(s.blockTypedEvents(t, height, tradebinEventPrefix+"LiquidityAddedEvent"), map[string]string{"creator": "burner", "pool_id": s.stakePool})
	require.Len(t, added, 1, "liquidity added by the burner")
	require.Equal(t, sdkmath.NewInt(brFundStake), s.transfers(t, events, s.burner, s.tradebin).AmountOf(s.stake), "stake voucher to tradebin")
	require.Equal(t, parseInt(added[0].Attributes["mintedAmount"]), returned.AmountOf(s.stakeLp), "minted LP returned to the burner")
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(chain.Denom, returned.AmountOf(chain.Denom)), sdk.NewCoin(s.stakeLp, returned.AmountOf(s.stakeLp))), returned, "tradebin returns LP and ubze only")
	before, after := s.poolAt(t, s.stakePool, height-1), s.poolAt(t, s.stakePool, height)
	require.Equal(t, before.reserveBase().AddRaw(brFundStake), after.reserveBase(), "stake reserve")
	require.Equal(t, before.reserveQuote().Sub(returned.AmountOf(chain.Denom)), after.reserveQuote(), "ubze reserve")

	// left in the burner: tradebin's returns and the fees moved in after the burn
	want := returned.Add(s.transfers(t, events, s.burnerFees, s.burner)...).Add(s.transfers(t, events, s.cpFees, s.burner)...)
	require.Equal(t, want, s.addrBalances(t, s.burner, height), "burner after the burn")
}

// 4. With the interval set to 2, only ticks ending an even week burn: of the
// next two ticks one leaves the burner untouched and the other locks the LP
// shares the previous pass left and burns the ubze. The interval is then
// parked at 1000 weeks again.
func (s *burnerSuite) interval(t *testing.T) {
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
		require.True(t, prev.AmountOf(s.stakeLp).IsPositive(), "LP shares left by the previous pass")
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(chain.Denom, prev.AmountOf(chain.Denom)), sdk.NewCoin(s.stakeLp, prev.AmountOf(s.stakeLp))), prev, "burner before the pass")
		burnable := sdk.NewCoins(sdk.NewCoin(chain.Denom, prev.AmountOf(chain.Denom)))
		s.requireBurned(t, height, s.burner, burnable)
		s.requireCoinsBurnedEvent(t, height, burnable)
		s.burns[height] = burnable
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(s.stakeLp, prev.AmountOf(s.stakeLp))), s.transfers(t, events, s.burner, s.blackHole), "LP shares locked")
		want := s.transfers(t, events, s.burnerFees, s.burner).Add(s.transfers(t, events, s.cpFees, s.burner)...)
		require.Equal(t, want, s.addrBalances(t, s.burner, height), "burner after the pass")
	}
	require.Equal(t, 1, burned, "one of two consecutive ticks burns")

	// park the periodic burn again, so the later steps' burn records are
	// the raffles' alone
	s.setBurningWeeks(t, brLockedWeeks)
}

// 5. StartRaffle captures the pot in burner_raffle and records the raffle,
// ending 24 × duration hour epochs after its block. Refusals leave everything
// but the tx fee untouched.
func (s *burnerSuite) startRaffle(t *testing.T) {
	for denom, r := range map[string]brRaffle{s.rw: brWin, s.rl: brLoss} {
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
// pot and, for the won raffle, the ticket prices that never joined it.
func (s *burnerSuite) raffleExpiry(t *testing.T) {
	gap := map[string]int64{s.rw: 2 * brWin.ticket, s.rl: 0}
	for _, denom := range []string{s.rw, s.rl} {
		t.Run(denom[strings.LastIndex(denom, "/")+1:], func(t *testing.T) {
			height := s.epochStartHeight(t, "hour", s.endAt[denom]+1, brExpiryWait)
			prev, found := s.raffle(t, denom, height-1)
			require.True(t, found, "raffle of %s before its end", denom)
			balance := s.addrBalances(t, s.raffleAcc, height-1).AmountOf(denom)
			require.Equal(t, parseInt(prev.Pot).AddRaw(gap[denom]), balance, "burner_raffle holds the recorded pot and the gap")

			burned := sdk.NewCoins(sdk.NewCoin(denom, balance))
			s.requireBurned(t, height, s.raffleAcc, burned)
			s.burns[height] = s.burns[height].Add(burned...)
			finished := helpers.EventsWith(s.blockTypedEvents(t, height, burnerEventPrefix+"RaffleFinishedEvent"), map[string]string{"denom": denom})
			require.Len(t, finished, 1, "RaffleFinishedEvent")
			require.Equal(t, s.supplyAt(t, denom, height-1).Sub(balance), s.supplyAt(t, denom, height), "supply of %s", denom)

			_, found = s.raffle(t, denom, height)
			require.False(t, found, "raffle of %s removed", denom)
			require.Empty(t, s.raffleWinners(t, denom), "winners of %s removed", denom)
			require.True(t, s.balance(t, s.raffleAcc, denom).IsZero(), "burner_raffle %s balance", denom)
		})
	}
}

// 10. MoveIbcLockedCoins needs a deep enough ubze pool: with no pool and with
// a shallow one it is refused. Once the pool is deep, the whole locked
// balance goes into the pool as liquidity, the LP shares stay locked and the
// ubze tradebin returns goes to the burner.
func (s *burnerSuite) moveLocked(t *testing.T) {
	locked := s.balance(t, s.blackHole, s.other)
	require.Equal(t, sdkmath.NewInt(brFundOther), locked, "utoken voucher locked in step 2")
	refuse := func(t *testing.T, denom, codespace string, code uint32, reason string) {
		t.Helper()
		before, hole := s.addrBalances(t, s.p1.FormattedAddress(), 0), s.addrBalances(t, s.blackHole, 0)
		res, err := helpers.ExecTxWith(s.ctx, s.bze, s.p1, []helpers.TxOption{helpers.WithGas(brTxGas)}, "burner", "move-ibc-locked-coins", denom)
		s.requireRefused(t, res, err, codespace, code, reason)
		require.Equal(t, before.Sub(s.paidFee(t, res)...), s.addrBalances(t, s.p1.FormattedAddress(), 0), "p1 balances")
		require.Equal(t, hole, s.addrBalances(t, s.blackHole, 0), "black hole balances")
	}

	t.Run("no pool", func(t *testing.T) {
		refuse(t, s.other, burnerCodespace, 1102, "liquidity availability")
	})
	t.Run("shallow pool", func(t *testing.T) {
		s.createPool(t, s.other, brOtherPoolVoucher, brOtherPoolBze)
		refuse(t, s.other, burnerCodespace, 1102, "liquidity availability")
	})

	t.Run("move", func(t *testing.T) {
		s.mustExec(t, s.admin, "tradebin", "add-liquidity", s.otherPool, strconv.Itoa(2*brOtherPoolVoucher), strconv.Itoa(2*brOtherPoolBze), "1")
		pool := s.pool(t, s.otherPool)
		hole, burner := s.addrBalances(t, s.blackHole, 0), s.balance(t, s.burner, chain.Denom)
		before := s.addrBalances(t, s.p1.FormattedAddress(), 0)

		res := s.mustExecWith(t, s.p1, []helpers.TxOption{helpers.WithGas(brTxGas)}, "burner", "move-ibc-locked-coins", s.other)
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(s.other, locked)), s.transfers(t, res.Events, s.blackHole, s.tradebin), "locked voucher to tradebin")
		returned := s.transfers(t, res.Events, s.tradebin, s.blackHole)
		refund := returned.AmountOf(chain.Denom)
		added := helpers.EventsWith(helpers.TxEvents(res, tradebinEventPrefix+"LiquidityAddedEvent"), map[string]string{"creator": "burner_black_hole", "pool_id": s.otherPool})
		require.Len(t, added, 1, "liquidity added by the black hole")
		minted := parseInt(added[0].Attributes["mintedAmount"])
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(pool.LpDenom, minted), sdk.NewCoin(chain.Denom, refund)), returned, "tradebin returns LP and ubze only")
		require.Equal(t, sdk.NewCoins(sdk.NewCoin(chain.Denom, refund)), s.transfers(t, res.Events, s.blackHole, s.burner), "returned ubze to the burner")

		after := s.pool(t, s.otherPool)
		require.Equal(t, pool.reserveBase().Add(locked), after.reserveBase(), "utoken reserve")
		require.Equal(t, pool.reserveQuote().Sub(refund), after.reserveQuote(), "ubze reserve")
		wantHole := hole.Sub(sdk.NewCoin(s.other, locked)).Add(sdk.NewCoin(pool.LpDenom, minted))
		require.Equal(t, wantHole, s.addrBalances(t, s.blackHole, 0), "black hole balances")
		require.True(t, s.balance(t, s.burner, chain.Denom).GTE(burner.Add(refund)), "the burner holds the returned ubze")
		require.Equal(t, before.Sub(sdk.NewCoin(chain.Denom, s.txFee(t, res))), s.addrBalances(t, s.p1.FormattedAddress(), 0), "p1 pays only the fee")
	})

	for _, c := range []struct {
		name, denom, codespace, reason string
		code                           uint32
	}{
		{"nothing locked", s.other, burnerCodespace, "no coins to move", 1102},
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

// 11. The queries return what the steps produced: the interval parked again
// at the end of step 4, no running raffle or winner left, and one all-burned-coins record per burn
// height with exactly the coins burned there.
func (s *burnerSuite) queries(t *testing.T) {
	require.Equal(t, int64(brLockedWeeks), s.burningWeeks(t), "periodic_burning_weeks")
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

func (s *burnerSuite) paramsMsg(authority string, weeks int64) map[string]any {
	return map[string]any{
		"@type":     "/bze.burner.MsgUpdateParams",
		"authority": authority,
		"params":    map[string]any{"periodic_burning_weeks": strconv.FormatInt(weeks, 10)},
	}
}

// setBurningWeeks passes a proposal setting periodic_burning_weeks and
// returns the height of the block whose gov EndBlock executed it: a week tick
// at that height still ran with the old interval.
func (s *burnerSuite) setBurningWeeks(t *testing.T, weeks int64) int64 {
	t.Helper()
	from, err := s.bze.Height(s.ctx)
	require.NoError(t, err)
	id, err := helpers.SubmitAndPassProposal(s.ctx, s.bze, s.admin, helpers.Proposal{
		Messages: []any{s.paramsMsg(s.gov, weeks)},
		Title:    "burner interval",
		Summary:  "e2e: set periodic_burning_weeks to " + strconv.FormatInt(weeks, 10),
	})
	require.NoError(t, err)
	require.Equal(t, weeks, s.burningWeeks(t))

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

func (s *burnerSuite) burningWeeks(t *testing.T) int64 {
	t.Helper()
	res, err := helpers.Query[struct {
		Params struct {
			PeriodicBurningWeeks helpers.Int64 `json:"periodic_burning_weeks"`
		} `json:"params"`
	}](s.ctx, s.bze, "burner", "params")
	require.NoError(t, err)
	return int64(res.Params.PeriodicBurningWeeks)
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
