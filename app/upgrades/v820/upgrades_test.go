package v820_test

import (
	"context"
	"errors"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	v820 "github.com/bze-alphateam/bze/app/upgrades/v820"
	testkeeper "github.com/bze-alphateam/bze/testutil/keeper"
	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

const (
	// expectedAdmin is the address the black hole's LP shares must end up on. It is
	// spelled out here on purpose: a typo in the wind-down table would send protocol
	// owned liquidity to a wallet nobody controls.
	expectedAdmin = "bze1jx4x3kn8mlz2s03zdpf2a38x9gl66llvjqdd55"
	// expectedLpDenom is the USDC.n/BZE pool's LP denomination.
	expectedLpDenom = "ulp_ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4_ubze"
	// expectedTestnetLpDenom is the LP denomination bzetestnet-3's black hole holds:
	// the testnet has no USDC.n pool, so the rehearsal moves these shares instead.
	expectedTestnetLpDenom = "ulp_ibc/9DA252F9F9C86132CC282EA431DFB7DE7729501F6DC9A3E0F50EC8C6EE380CC7_ubze"

	// otherLpDenom stands for the black hole's other holdings (VDL/BZE and PHOTON/BZE
	// LP shares, locked coins); they must survive the upgrade untouched.
	otherLpDenom = "ulp_ibc/0000000000000000000000000000000000000000000000000000000000000000_ubze"
)

// bankKeeper is an in-memory stand-in for the bank keeper: balances by address and
// denom, plus the module-account address resolution the handler needs.
type bankKeeper struct {
	balances map[string]sdk.Coins
	sendErr  error
	sends    int
}

// The handler decodes the admin address with the chain's bech32 prefix, which the app
// package's init() installs in the running binary. This test binary does not import the
// app package, so it installs the same prefix itself.
func init() {
	sdk.GetConfig().SetBech32PrefixForAccount("bze", "bzepub")
}

func newBankKeeper() *bankKeeper {
	return &bankKeeper{balances: map[string]sdk.Coins{}}
}

func (b *bankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, b.balances[addr.String()].AmountOf(denom))
}

func (b *bankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	b.sends++
	if b.sendErr != nil {
		return b.sendErr
	}

	sender := authtypes.NewModuleAddress(senderModule).String()
	current := b.balances[sender]
	remaining, negative := current.SafeSub(amt...)
	if negative {
		return errors.New("insufficient funds")
	}

	b.balances[sender] = remaining
	b.balances[recipientAddr.String()] = b.balances[recipientAddr.String()].Add(amt...)

	return nil
}

func (b *bankKeeper) fund(addr sdk.AccAddress, coins ...sdk.Coin) {
	b.balances[addr.String()] = b.balances[addr.String()].Add(sdk.NewCoins(coins...)...)
}

func (b *bankKeeper) balanceOf(addr sdk.AccAddress, denom string) sdkmath.Int {
	return b.balances[addr.String()].AmountOf(denom)
}

// accountKeeper resolves module names the way the real account keeper does.
type accountKeeper struct{}

func (accountKeeper) GetModuleAddress(moduleName string) sdk.AccAddress {
	return authtypes.NewModuleAddress(moduleName)
}

func blackHoleAddress() sdk.AccAddress {
	return authtypes.NewModuleAddress(burnertypes.BlackHoleModuleName)
}

func adminAddress(t *testing.T) sdk.AccAddress {
	t.Helper()
	addr, err := sdk.AccAddressFromBech32(expectedAdmin)
	require.NoError(t, err)

	return addr
}

// setup builds the real txfeecollector keeper over an in-memory store plus the fake
// bank keeper, on a context carrying the given chain id.
func setup(t *testing.T, chainID string) (sdk.Context, *bankKeeper, txfeecollectorKeeper) {
	t.Helper()

	k, ctx := testkeeper.TxfeecollectorKeeper(t)

	return ctx.WithChainID(chainID), newBankKeeper(), k
}

// txfeecollectorKeeper is the concrete keeper type returned by the test helper; naming
// it keeps the setup signature readable.
type txfeecollectorKeeper = interface {
	GetParams(ctx context.Context) txfeecollectortypes.Params
	SetParams(ctx context.Context, params txfeecollectortypes.Params) error
}

// The wind-down table must carry exactly the values agreed for mainnet. This is the
// guard against a wrong address or denom slipping in unnoticed.
func TestWindDownTable_Mainnet(t *testing.T) {
	windDown, found := v820.GetWindDown("beezee-1")
	require.True(t, found)

	require.Equal(t, expectedAdmin, windDown.AdminAddress)
	require.Equal(t, expectedLpDenom, windDown.LpDenom)
	require.Equal(t, []txfeecollectortypes.BlockedIbcTransfer{
		{ChannelId: "channel-3", BaseDenom: "uusdc"},
	}, windDown.BlockedInbound)

	// the values must survive param validation, otherwise the upgrade would fail
	params := txfeecollectortypes.DefaultParams()
	params.BlockedIbcInbound = windDown.BlockedInbound
	require.NoError(t, params.Validate())

	_, err := sdk.AccAddressFromBech32(windDown.AdminAddress)
	require.NoError(t, err)
}

// bzetestnet-3 is the only testnet still running, and it runs the wind-down against
// assets it actually has so both steps can be observed there.
func TestWindDownTable_Testnet(t *testing.T) {
	windDown, found := v820.GetWindDown("bzetestnet-3")
	require.True(t, found)

	require.Equal(t, expectedAdmin, windDown.AdminAddress)
	require.Equal(t, expectedTestnetLpDenom, windDown.LpDenom)
	require.Equal(t, []txfeecollectortypes.BlockedIbcTransfer{
		{ChannelId: "channel-0", BaseDenom: "ulmn"},
	}, windDown.BlockedInbound)

	params := txfeecollectortypes.DefaultParams()
	params.BlockedIbcInbound = windDown.BlockedInbound
	require.NoError(t, params.Validate())
}

// The retired testnets are gone from the table: they run neither part.
func TestWindDownTable_RetiredTestnets(t *testing.T) {
	for _, chainID := range []string{"bzetestnet-1", "bzetestnet-2"} {
		_, found := v820.GetWindDown(chainID)
		require.False(t, found, chainID)
	}
}

func TestWindDownTable_UnknownChain(t *testing.T) {
	_, found := v820.GetWindDown("localnet-1")
	require.False(t, found)
}

// The full mainnet outcome: the param blocks inbound Noble uusdc, the black hole's
// USDC.n/BZE LP shares are gone, the admin holds exactly that amount, and the black
// hole's other balances are untouched.
func TestApplyWindDown_Mainnet(t *testing.T) {
	ctx, bank, k := setup(t, "beezee-1")

	lpShares := sdkmath.NewInt(49778794490303243)
	otherShares := sdkmath.NewInt(123456789)
	bank.fund(blackHoleAddress(),
		sdk.NewCoin(expectedLpDenom, lpShares),
		sdk.NewCoin(otherLpDenom, otherShares),
		sdk.NewInt64Coin("ubze", 1_000000),
	)

	require.NoError(t, v820.ApplyWindDown(ctx, bank, accountKeeper{}, k))

	// Part A: the param blocks Noble uusdc on channel-3 and nothing else
	params := k.GetParams(ctx)
	require.Equal(t, []txfeecollectortypes.BlockedIbcTransfer{
		{ChannelId: "channel-3", BaseDenom: "uusdc"},
	}, params.BlockedIbcInbound)
	require.True(t, params.IsInboundBlocked("channel-3", "uusdc"))
	require.False(t, params.IsInboundBlocked("channel-13", "uusdc"))
	// the other params are left alone
	require.Equal(t, txfeecollectortypes.DefaultParams().ValidatorMinGasFee, params.ValidatorMinGasFee)
	require.Equal(t, txfeecollectortypes.DefaultParams().MaxBalanceIterations, params.MaxBalanceIterations)

	// Part B: the LP shares moved in full, nothing else did
	require.True(t, bank.balanceOf(blackHoleAddress(), expectedLpDenom).IsZero())
	require.Equal(t, lpShares, bank.balanceOf(adminAddress(t), expectedLpDenom))
	require.Equal(t, otherShares, bank.balanceOf(blackHoleAddress(), otherLpDenom))
	require.Equal(t, sdkmath.NewInt(1_000000), bank.balanceOf(blackHoleAddress(), "ubze"))
	require.True(t, bank.balanceOf(adminAddress(t), otherLpDenom).IsZero())
	require.True(t, bank.balanceOf(adminAddress(t), "ubze").IsZero())

	// exactly one transfer was made, so the whole balance moved in a single send
	require.Equal(t, 1, bank.sends)
}

// The testnet rehearsal: bzetestnet-3 blocks its own channel and moves its own LP
// shares, and nothing keyed to mainnet applies there.
func TestApplyWindDown_Testnet(t *testing.T) {
	ctx, bank, k := setup(t, "bzetestnet-3")

	lpShares := sdkmath.NewInt(2877672670753153)
	bank.fund(blackHoleAddress(),
		sdk.NewCoin(expectedTestnetLpDenom, lpShares),
		sdk.NewCoin(expectedLpDenom, sdkmath.NewInt(42)),
	)

	require.NoError(t, v820.ApplyWindDown(ctx, bank, accountKeeper{}, k))

	params := k.GetParams(ctx)
	require.True(t, params.IsInboundBlocked("channel-0", "ulmn"))
	require.False(t, params.IsInboundBlocked("channel-3", "uusdc"))

	require.True(t, bank.balanceOf(blackHoleAddress(), expectedTestnetLpDenom).IsZero())
	require.Equal(t, lpShares, bank.balanceOf(adminAddress(t), expectedTestnetLpDenom))
	// the mainnet denom is not part of the testnet wind-down
	require.Equal(t, sdkmath.NewInt(42), bank.balanceOf(blackHoleAddress(), expectedLpDenom))
}

// A chain that holds none of the LP denom - a pool already emptied, or a second run of
// the same handler - still gets the inbound block and never fails.
func TestApplyWindDown_NoLpBalanceIsANoOp(t *testing.T) {
	ctx, bank, k := setup(t, "beezee-1")

	bank.fund(blackHoleAddress(), sdk.NewCoin(otherLpDenom, sdkmath.NewInt(5)))

	require.NoError(t, v820.ApplyWindDown(ctx, bank, accountKeeper{}, k))

	require.True(t, k.GetParams(ctx).IsInboundBlocked("channel-3", "uusdc"))
	require.Equal(t, 0, bank.sends)
	require.Equal(t, sdkmath.NewInt(5), bank.balanceOf(blackHoleAddress(), otherLpDenom))
}

// Running the handler twice must not move anything a second time.
func TestApplyWindDown_IsIdempotent(t *testing.T) {
	ctx, bank, k := setup(t, "beezee-1")

	lpShares := sdkmath.NewInt(1000)
	bank.fund(blackHoleAddress(), sdk.NewCoin(expectedLpDenom, lpShares))

	require.NoError(t, v820.ApplyWindDown(ctx, bank, accountKeeper{}, k))
	require.NoError(t, v820.ApplyWindDown(ctx, bank, accountKeeper{}, k))

	require.Equal(t, 1, bank.sends)
	require.Equal(t, lpShares, bank.balanceOf(adminAddress(t), expectedLpDenom))
	require.True(t, k.GetParams(ctx).IsInboundBlocked("channel-3", "uusdc"))
}

// A chain id the table does not know runs neither part.
func TestApplyWindDown_UnknownChainDoesNothing(t *testing.T) {
	ctx, bank, k := setup(t, "localnet-1")

	bank.fund(blackHoleAddress(), sdk.NewCoin(expectedLpDenom, sdkmath.NewInt(1000)))

	require.NoError(t, v820.ApplyWindDown(ctx, bank, accountKeeper{}, k))

	require.Empty(t, k.GetParams(ctx).BlockedIbcInbound)
	require.Equal(t, 0, bank.sends)
	require.Equal(t, sdkmath.NewInt(1000), bank.balanceOf(blackHoleAddress(), expectedLpDenom))
}

// A failing bank transfer must not brick the upgrade: the shares stay where they were
// and the inbound block - already written - stands.
func TestApplyWindDown_TransferFailureDoesNotFailTheUpgrade(t *testing.T) {
	ctx, bank, k := setup(t, "beezee-1")

	bank.fund(blackHoleAddress(), sdk.NewCoin(expectedLpDenom, sdkmath.NewInt(1000)))
	bank.sendErr = errors.New("bank is unhappy")

	require.NoError(t, v820.ApplyWindDown(ctx, bank, accountKeeper{}, k))

	require.True(t, k.GetParams(ctx).IsInboundBlocked("channel-3", "uusdc"))
	require.Equal(t, sdkmath.NewInt(1000), bank.balanceOf(blackHoleAddress(), expectedLpDenom))
	require.True(t, bank.balanceOf(adminAddress(t), expectedLpDenom).IsZero())
}
