package interchaintest_test

import (
	"context"
	"encoding/json"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
	"github.com/bze-alphateam/bze/interchaintest/helpers"
)

const (
	ghGuardianFunds = 100_000_000_000 // ubze for the guardian's tx fees

	ghCodespace        = "guardian"
	ghUnauthorizedCode = 1101 // ErrUnauthorized
	ghInvalidDenomCode = 1108 // ErrInvalidDenom

	ghEventPrefix = "bze.guardian.v1."
)

// TestGuardianHaltTradingDenoms halts a factory denom on the DEX with the
// guardian's MsgHaltTradingDenoms instead of a governance proposal, and
// proves the halt is the one governance's MsgHaltDenoms sets: every trading
// path that touches the denom is refused, cancelling an order, removing
// liquidity and bank sends keep working and the ante handler no longer takes
// it as a fee denom. Any other account is refused, the native denom cannot be
// halted and the guardian cannot lift the halt: a real governance proposal
// with tradebin's MsgUnhaltDenoms does, and trading works again.
//
// The scenario and its checks are the halted denoms suite's (halt_test.go);
// only the halt itself and the guardian's refusals are new. Subtests run in
// order and share state.
func TestGuardianHaltTradingDenoms(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in short mode")
	}
	ctx := context.Background()

	guardianMnemonic, guardianAddr := newAccount(t, chain.Bech32Prefix)
	// the guardian is a test wallet set in genesis, so the suite does not
	// depend on the address the upgrade handler sets on mainnet
	net := chain.Start(t, ctx, chain.WithGenesis(cosmos.NewGenesisKV("app_state.guardian.params.guardian", guardianAddr)))
	s := setupGuardianHalt(t, ctx, net.Chain, guardianMnemonic, guardianAddr)

	t.Run("1 before the halt", s.beforeHalt)
	t.Run("2 a user cannot halt", s.strangerRefused)
	t.Run("3 guardian halts", s.guardianHalts)
	t.Run("4 native denom can not be halted", s.guardianHaltNative)
	t.Run("5 trading refused", s.tradingRefused)
	t.Run("6 exits still work", s.exits)
	t.Run("7 fee denom", s.feeDenom)
	t.Run("8 guardian cannot unhalt", s.guardianCannotUnhalt)
	t.Run("9 unhalt by governance", s.unhalt)
}

type guardianHaltSuite struct {
	// the halted denoms suite: tokens, pools, market, resting order and checks
	*haltSuite

	guardian ibc.Wallet
}

func setupGuardianHalt(t *testing.T, ctx context.Context, bze *cosmos.CosmosChain, guardianMnemonic, guardianAddr string) *guardianHaltSuite {
	t.Helper()
	s := &guardianHaltSuite{haltSuite: setupHalt(t, ctx, bze)}

	var err error
	s.guardian, err = interchaintest.GetAndFundTestUserWithMnemonic(ctx, "guardian", guardianMnemonic, sdkmath.NewInt(ghGuardianFunds), bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, s.guardian.FormattedAddress())
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, bze))

	guardian, err := helpers.QueryGuardian(ctx, bze)
	require.NoError(t, err)
	require.Equal(t, guardianAddr, guardian, "the genesis guardian")
	return s
}

// haltTx sends `tx guardian halt-trading-denoms` from user with a fixed gas,
// so a refusal comes from the msg server, in a block.
func (s *guardianHaltSuite) haltTx(t *testing.T, user ibc.Wallet, denoms ...string) helpers.TxResult {
	t.Helper()
	res, err := helpers.ExecTxWith(s.ctx, s.bze, user, []helpers.TxOption{helpers.WithGas(haltTxGas)},
		append([]string{"guardian", "halt-trading-denoms"}, denoms...)...)
	require.NoError(t, err)
	return res
}

// 2. The guardian message signed by a user is refused by the msg server and
// halts nothing.
func (s *guardianHaltSuite) strangerRefused(t *testing.T) {
	res := s.haltTx(t, s.trader, s.hlt)
	require.NotZero(t, res.Height, "refused by CheckTx instead of the msg server: %s", res.RawLog)
	require.Equal(t, ghCodespace, res.Codespace, res.RawLog)
	require.EqualValues(t, ghUnauthorizedCode, res.Code, res.RawLog)
	require.False(t, s.denomHalted(t, s.hlt))
	s.requireHaltedDenoms(t)
}

// 3. The guardian halts hlt in one tx: tradebin's DenomHaltedEvent, as with
// a governance halt, and the guardian's TradingDenomsHaltedEvent. A repeat
// succeeds, halts nothing and emits no tradebin event.
func (s *guardianHaltSuite) guardianHalts(t *testing.T) {
	res := s.haltTx(t, s.guardian, s.hlt)
	require.Zero(t, res.Code, res.RawLog)

	halted := helpers.TxEvents(res, tradebinEventPrefix+"DenomHaltedEvent")
	require.Len(t, halted, 1, "%+v", halted)
	require.Equal(t, s.hlt, halted[0].Attributes["denom"])
	s.requireGuardianEvent(t, res, s.hlt)
	require.True(t, s.denomHalted(t, s.hlt))
	s.requireHaltedDenoms(t, s.hlt)

	t.Run("repeat", func(t *testing.T) {
		res := s.haltTx(t, s.guardian, s.hlt)
		require.Zero(t, res.Code, res.RawLog)
		require.Empty(t, helpers.TxEvents(res, tradebinEventPrefix+"DenomHaltedEvent"), "already halted: no tradebin event")
		s.requireGuardianEvent(t, res)
		s.requireHaltedDenoms(t, s.hlt)
	})
}

// 4. Halting the native denom would halt every market and pool: refused,
// alone or next to another denom, and nothing is halted.
func (s *guardianHaltSuite) guardianHaltNative(t *testing.T) {
	for name, denoms := range map[string][]string{
		"alone":           {chain.Denom},
		"next to another": {s.oth, chain.Denom},
	} {
		t.Run(name, func(t *testing.T) {
			res := s.haltTx(t, s.guardian, denoms...)
			require.NotZero(t, res.Height, "refused by CheckTx instead of the msg server: %s", res.RawLog)
			require.Equal(t, ghCodespace, res.Codespace, res.RawLog)
			require.EqualValues(t, ghInvalidDenomCode, res.Code, res.RawLog)
			require.Contains(t, res.RawLog, "the native denom "+chain.Denom+" cannot be halted")
			require.False(t, s.denomHalted(t, chain.Denom))
			require.False(t, s.denomHalted(t, s.oth))
			s.requireHaltedDenoms(t, s.hlt)
		})
	}
}

// 8. Only governance lifts a halt: tradebin's MsgUnhaltDenoms signed by the
// guardian is refused and hlt stays halted.
func (s *guardianHaltSuite) guardianCannotUnhalt(t *testing.T) {
	res, err := helpers.ExecMsgs(s.ctx, s.bze, s.guardian, []helpers.TxOption{helpers.WithGas(haltTxGas)},
		s.haltMsg("MsgUnhaltDenoms", s.guardian.FormattedAddress(), s.hlt))
	reason, err := helpers.RejectionReason(res, err)
	require.NoError(t, err)
	require.Contains(t, reason, "invalid authority")
	require.True(t, s.denomHalted(t, s.hlt), "hlt still halted")
}

// requireGuardianEvent asserts the tx emitted one TradingDenomsHaltedEvent
// signed by the guardian, listing exactly the denoms it halted.
func (s *guardianHaltSuite) requireGuardianEvent(t *testing.T, res helpers.TxResult, denoms ...string) {
	t.Helper()
	events := helpers.TxEvents(res, ghEventPrefix+"TradingDenomsHaltedEvent")
	require.Len(t, events, 1, "%+v", events)
	require.Equal(t, s.guardian.FormattedAddress(), events[0].Attributes["signer"])

	var got []string
	if raw := events[0].Attributes["denoms"]; raw != "" {
		require.NoError(t, json.Unmarshal([]byte(raw), &got), raw)
	}
	require.ElementsMatch(t, denoms, got)
}
