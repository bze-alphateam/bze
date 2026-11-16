package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/burner/keeper"
	"github.com/bze-alphateam/bze/x/burner/testutil"
	"github.com/bze-alphateam/bze/x/burner/types"
)

// burnerRaffleFailingMeter is an infinite gas meter that, once armed, panics on the next gas consumption, standing
// in for any failure in the draw after the payout. It panics once only, and not with an out-of-gas error.
type burnerRaffleFailingMeter struct {
	storetypes.GasMeter
	armed bool
	fired bool
}

func (m *burnerRaffleFailingMeter) ConsumeGas(amount storetypes.Gas, descriptor string) {
	if m.armed {
		m.armed = false
		m.fired = true
		panic("failure after the payout")
	}
	m.GasMeter.ConsumeGas(amount, descriptor)
}

// burnerRaffleFixture is a burner keeper next to a ledger store the recording bank writes its payouts to, in the
// same multistore: a payout written through a context that is dropped is dropped with it, like a real bank send.
type burnerRaffleFixture struct {
	k        keeper.Keeper
	ctx      sdk.Context
	ledger   *storetypes.KVStoreKey
	meter    *burnerRaffleFailingMeter
	sends    int
	failNext bool
}

func newBurnerRaffleFixture(t *testing.T) *burnerRaffleFixture {
	t.Helper()
	burnerKey := storetypes.NewKVStoreKey(types.StoreKey)
	ledgerKey := storetypes.NewKVStoreKey("raffle_payout_ledger")

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(burnerKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(ledgerKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctrl := gomock.NewController(t)
	bank := testutil.NewMockBankKeeper(ctrl)
	guardian := testutil.NewMockGuardianKeeper(ctrl)
	guardian.EXPECT().IsAutomationPaused(gomock.Any()).Return(false).AnyTimes()

	f := &burnerRaffleFixture{
		ledger: ledgerKey,
		meter:  &burnerRaffleFailingMeter{GasMeter: storetypes.NewInfiniteGasMeter()},
	}
	// the recording bank writes the payout to the ledger through the context it is given, then arms the meter
	// when the test asks for a failure after the send
	bank.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.RaffleModuleName, gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ string, to sdk.AccAddress, amt sdk.Coins) error {
			f.sends++
			sdk.UnwrapSDKContext(ctx).KVStore(ledgerKey).Set(to, []byte(amt.String()))
			if f.failNext {
				f.meter.armed = true
			}

			return nil
		}).AnyTimes()

	f.k = keeper.NewKeeper(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		runtime.NewKVStoreService(burnerKey),
		log.NewNopLogger(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		bank,
		testutil.NewMockAccountKeeper(ctrl),
		testutil.NewMockEpochKeeper(ctrl),
		testutil.NewMockTradeKeeper(ctrl),
		guardian,
	)
	f.ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger()).WithGasMeter(f.meter)
	require.NoError(t, f.k.SetParams(f.ctx, types.DefaultParams()))

	return f
}

// paid is what the ledger holds for addr, empty when nothing was paid.
func (f *burnerRaffleFixture) paid(addr sdk.AccAddress) string {
	return string(f.ctx.KVStore(f.ledger).Get(addr))
}

// seedWinningTicket stores a raffle every ticket wins (pot 100000, 10% prize) and one ticket for winner.
func (f *burnerRaffleFixture) seedWinningTicket(winner sdk.AccAddress) {
	f.k.SetRaffle(f.ctx, types.Raffle{
		Denom:       "utoken",
		Pot:         "100000",
		Chances:     1_000_000,
		Ratio:       "0.1",
		TicketPrice: "10",
		TotalWon:    "0",
	})
	f.k.SetRaffleParticipant(f.ctx, types.RaffleParticipant{Denom: "utoken", Participant: winner.String(), ExecuteAt: 10})
}

// Without a failure the recorded payout and the raffle totals are committed together.
func TestWithdrawLucky_PayoutCommittedWithTheTotals(t *testing.T) {
	f := newBurnerRaffleFixture(t)
	winner := sdk.AccAddress("raffle_winner_______")
	f.seedWinningTicket(winner)

	f.k.WithdrawLuckyRaffleParticipants(f.ctx, 10)

	require.Equal(t, 1, f.sends)
	require.Equal(t, sdk.NewCoins(sdk.NewCoin("utoken", math.NewInt(10_000))).String(), f.paid(winner))
	raffle, found := f.k.GetRaffle(f.ctx, "utoken")
	require.True(t, found)
	require.Equal(t, "90000", raffle.Pot)
	require.Equal(t, "10000", raffle.TotalWon)
	require.Equal(t, uint64(1), raffle.Winners)
	require.Len(t, f.k.GetRaffleWinners(f.ctx, "utoken"), 1)
	require.Empty(t, f.k.GetAllRaffleParticipants(f.ctx))
}

// A failure in the draw after the winner's send succeeded voids the whole draw: the payout is rolled back with the
// raffle totals and the winner record, and the ticket is still removed.
func TestWithdrawLucky_FailureAfterPayoutRollsBackThePayout(t *testing.T) {
	f := newBurnerRaffleFixture(t)
	winner := sdk.AccAddress("raffle_winner_______")
	f.seedWinningTicket(winner)
	f.failNext = true

	require.NotPanics(t, func() { f.k.WithdrawLuckyRaffleParticipants(f.ctx, 10) })

	require.Equal(t, 1, f.sends, "the send ran")
	require.True(t, f.meter.fired, "the draw failed after the send")
	require.Empty(t, f.paid(winner), "the payout is rolled back")

	raffle, found := f.k.GetRaffle(f.ctx, "utoken")
	require.True(t, found)
	require.Equal(t, "100000", raffle.Pot)
	require.Equal(t, "0", raffle.TotalWon)
	require.Equal(t, uint64(0), raffle.Winners)
	require.Empty(t, f.k.GetRaffleWinners(f.ctx, "utoken"))
	require.Empty(t, f.k.GetAllRaffleParticipants(f.ctx), "the ticket is removed")
}
