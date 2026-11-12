package keeper_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/guardian/testutil"
	"github.com/bze-alphateam/bze/x/guardian/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	tradebinv2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

const (
	haltNative = "ubze"
	haltTokenA = "factory/bze1abc/utoka"
	haltTokenB = "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"
)

// haltEnv is the guardian env with a mocked tradebin keeper wired through
// SetTradebinKeeper. The mock's controller is strict: a call without an
// expectation fails the test, and so does an expectation left unmet.
type haltEnv struct {
	*env
	tb *testutil.MockTradebinKeeper
}

func newHaltEnv(t *testing.T) *haltEnv {
	t.Helper()
	e := newEnv(t)
	tb := testutil.NewMockTradebinKeeper(gomock.NewController(t))
	e.k.SetTradebinKeeper(tb)
	return &haltEnv{env: e, tb: tb}
}

func (e *haltEnv) expectParams() {
	e.tb.EXPECT().GetParams(gomock.Any()).Return(tradebinv2types.Params{NativeDenom: haltNative}).Times(1)
}

func (e *haltEnv) expectHalted(denom string, halted bool) {
	e.tb.EXPECT().IsDenomHalted(gomock.Any(), denom).Return(halted).Times(1)
}

func (e *haltEnv) expectSet(denom string) {
	e.tb.EXPECT().SetHaltedDenom(gomock.Any(), denom).Times(1)
}

func haltMsg(signer string, denoms ...string) *types.MsgHaltTradingDenoms {
	return &types.MsgHaltTradingDenoms{Signer: signer, Denoms: denoms}
}

// typedEvents decodes every typed event of the context, in emission order.
func typedEvents(t *testing.T, ctx sdk.Context) []proto.Message {
	t.Helper()
	var out []proto.Message
	for _, ev := range ctx.EventManager().Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		require.NoError(t, err)
		out = append(out, msg)
	}
	return out
}

// The guardian and the governance authority halt with the calls of
// tradebin's MsgHaltDenoms handler: the native denom read once from the
// params, then per denom IsDenomHalted and SetHaltedDenom, each halt with
// tradebin's own DenomHaltedEvent, then one guardian event.
func TestHaltTradingDenoms_GuardianAndAuthorityHalt(t *testing.T) {
	for name, signer := range map[string]string{"guardian": guardianAddr, "authority": authority} {
		t.Run(name, func(t *testing.T) {
			e := newHaltEnv(t)
			gomock.InOrder(
				e.tb.EXPECT().GetParams(gomock.Any()).Return(tradebinv2types.Params{NativeDenom: haltNative}),
				e.tb.EXPECT().IsDenomHalted(gomock.Any(), haltTokenA).Return(false),
				e.tb.EXPECT().SetHaltedDenom(gomock.Any(), haltTokenA),
				e.tb.EXPECT().IsDenomHalted(gomock.Any(), haltTokenB).Return(false),
				e.tb.EXPECT().SetHaltedDenom(gomock.Any(), haltTokenB),
			)

			res, err := e.msgServer.HaltTradingDenoms(e.ctx, haltMsg(signer, haltTokenA, haltTokenB))
			require.NoError(t, err)
			require.NotNil(t, res)

			require.Equal(t, []proto.Message{
				&tradebintypes.DenomHaltedEvent{Denom: haltTokenA},
				&tradebintypes.DenomHaltedEvent{Denom: haltTokenB},
				&types.TradingDenomsHaltedEvent{Signer: signer, Denoms: []string{haltTokenA, haltTokenB}},
			}, typedEvents(t, e.ctx))
		})
	}
}

// A denom that is already halted is left as it is: no write and no tradebin
// event; the guardian event lists only the denoms the message halted.
func TestHaltTradingDenoms_AlreadyHaltedIsLeftAsIs(t *testing.T) {
	e := newHaltEnv(t)
	e.expectParams()
	e.expectHalted(haltTokenA, true)
	e.expectHalted(haltTokenB, false)
	e.expectSet(haltTokenB)

	_, err := e.msgServer.HaltTradingDenoms(e.ctx, haltMsg(guardianAddr, haltTokenA, haltTokenB))
	require.NoError(t, err)

	require.Equal(t, []proto.Message{
		&tradebintypes.DenomHaltedEvent{Denom: haltTokenB},
		&types.TradingDenomsHaltedEvent{Signer: guardianAddr, Denoms: []string{haltTokenB}},
	}, typedEvents(t, e.ctx))
}

// Halting only denoms that are halted already succeeds without a write, like
// tradebin's handler, so a repeat never fails.
func TestHaltTradingDenoms_AllAlreadyHaltedSucceeds(t *testing.T) {
	e := newHaltEnv(t)
	e.expectParams()
	e.expectHalted(haltTokenA, true)

	_, err := e.msgServer.HaltTradingDenoms(e.ctx, haltMsg(guardianAddr, haltTokenA))
	require.NoError(t, err)

	require.Equal(t, []proto.Message{
		&types.TradingDenomsHaltedEvent{Signer: guardianAddr, Denoms: []string{}},
	}, typedEvents(t, e.ctx))
}

// Any other signer is refused before tradebin is touched.
func TestHaltTradingDenoms_StrangerIsRefused(t *testing.T) {
	e := newHaltEnv(t)

	_, err := e.msgServer.HaltTradingDenoms(e.ctx, haltMsg(strangerAddr, haltTokenA))
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.Empty(t, e.ctx.EventManager().Events())
}

// With no guardian in the params only the governance authority can halt.
func TestHaltTradingDenoms_NoGuardianOnlyAuthority(t *testing.T) {
	e := newHaltEnv(t)
	require.NoError(t, e.k.SetParams(e.ctx, types.NewParams("")))

	_, err := e.msgServer.HaltTradingDenoms(e.ctx, haltMsg(guardianAddr, haltTokenA))
	require.ErrorIs(t, err, types.ErrUnauthorized)

	e.expectParams()
	e.expectHalted(haltTokenA, false)
	e.expectSet(haltTokenA)
	_, err = e.msgServer.HaltTradingDenoms(e.ctx, haltMsg(authority, haltTokenA))
	require.NoError(t, err)
}

// The native denom is refused wherever it is in the list, before anything is
// written: halting it would halt every market and pool.
func TestHaltTradingDenoms_NativeDenomIsRefused(t *testing.T) {
	for name, denoms := range map[string][]string{
		"alone":  {haltNative},
		"first":  {haltNative, haltTokenA},
		"second": {haltTokenA, haltNative},
	} {
		t.Run(name, func(t *testing.T) {
			e := newHaltEnv(t)
			e.expectParams()
			// IsDenomHalted / SetHaltedDenom have no expectation: a call fails the test

			_, err := e.msgServer.HaltTradingDenoms(e.ctx, haltMsg(guardianAddr, denoms...))
			require.ErrorIs(t, err, types.ErrInvalidDenom)
			require.ErrorContains(t, err, "the native denom "+haltNative+" cannot be halted")
			require.Empty(t, e.ctx.EventManager().Events())
		})
	}
}

// An invalid list from the guardian is refused before tradebin is looked
// at: the handler runs ValidateBasic itself, for messages executed by x/gov
// or x/authz.
func TestHaltTradingDenoms_InvalidListIsRefused(t *testing.T) {
	for name, c := range map[string]struct {
		denoms []string
		reason string
	}{
		"empty":     {[]string{}, "no denoms provided"},
		"invalid":   {[]string{"u token"}, `invalid denom "u token"`},
		"duplicate": {[]string{haltTokenA, haltTokenB, haltTokenA}, "duplicate denom " + haltTokenA},
		"too many":  {manyDenoms(tradebintypes.MaxDenomsPerHaltMsg + 1), "at most 1000 denoms per message, got 1001"},
	} {
		t.Run(name, func(t *testing.T) {
			e := newHaltEnv(t)

			_, err := e.msgServer.HaltTradingDenoms(e.ctx, haltMsg(guardianAddr, c.denoms...))
			require.ErrorIs(t, err, types.ErrInvalidDenom)
			require.ErrorContains(t, err, c.reason)
			require.Empty(t, e.ctx.EventManager().Events())
		})
	}
}

// The signer is checked before the message body, as in tradebin's
// MsgHaltDenoms handler: a stranger, or an address that does not decode, is
// unauthorized even when the denoms are invalid too.
func TestHaltTradingDenoms_SignerIsCheckedFirst(t *testing.T) {
	for name, signer := range map[string]string{"stranger": strangerAddr, "not an address": "nope", "empty": ""} {
		for list, denoms := range map[string][]string{
			"valid":     {haltTokenA},
			"empty":     {},
			"invalid":   {"u token"},
			"duplicate": {haltTokenA, haltTokenA},
			"native":    {haltNative},
		} {
			t.Run(name+"/"+list, func(t *testing.T) {
				e := newHaltEnv(t)

				_, err := e.msgServer.HaltTradingDenoms(e.ctx, haltMsg(signer, denoms...))
				require.ErrorIs(t, err, types.ErrUnauthorized)
				require.NotErrorIs(t, err, types.ErrInvalidDenom)
				require.Empty(t, e.ctx.EventManager().Events())
			})
		}
	}
}

// Without a tradebin keeper the message fails instead of panicking.
func TestHaltTradingDenoms_TradebinKeeperNotSet(t *testing.T) {
	e := newEnv(t)

	_, err := e.msgServer.HaltTradingDenoms(e.ctx, haltMsg(guardianAddr, haltTokenA))
	require.ErrorIs(t, err, types.ErrTradebinKeeperNotSet)
}

func manyDenoms(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "factory/bze1abc/u" + string(rune('a'+i%26)) + string(rune('a'+(i/26)%26)) + string(rune('a'+(i/676)%26))
	}
	return out
}
