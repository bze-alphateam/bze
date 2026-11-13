package v830_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channelv10 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v10"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/bze-alphateam/bze/app"
	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	daotypes "github.com/bze-alphateam/bze/x/dao/types"
	guardiantypes "github.com/bze-alphateam/bze/x/guardian/types"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	v5 "github.com/bze-alphateam/bze/x/tradebin/migrations/v5"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
)

const (
	v830FullChannel = "channel-1" // not on the initial rate limit list
	v830FullBase    = "ufullbase"
	v830FullQuote   = "ufullquote"
)

// v830FullVersionMap is the module version map of a v8.2.0 node: the IBC core, transfer, tradebin, rewards and
// tokenfactory modules one version behind or more, x/dao, x/guardian and the rate-limiting module not there yet,
// every other module at its current version.
func v830FullVersionMap(bApp *app.App) map[string]uint64 {
	vm := bApp.ModuleManager.GetVersionMap()
	vm[ibcexported.ModuleName] = 6
	vm[transfertypes.ModuleName] = 5
	vm[tradebintypes.ModuleName] = 4
	vm[rewardstypes.ModuleName] = 5
	vm[tokenfactorytypes.ModuleName] = 3
	delete(vm, daotypes.ModuleName)
	delete(vm, guardiantypes.ModuleName)
	delete(vm, ratelimittypes.ModuleName)

	return vm
}

// v830FullSetFlushingChannel stores a transfer channel end in the ibc-go v8 encoding, FLUSHING an upgrade that
// would have added the fee middleware.
func v830FullSetFlushingChannel(t *testing.T, bApp *app.App, ctx sdk.Context, channelID string) {
	t.Helper()
	store := ctx.KVStore(bApp.GetKey(ibcexported.StoreKey))

	channel := channelv10.Channel{
		State:           channelv10.FLUSHING,
		Ordering:        channelv10.UNORDERED,
		Counterparty:    channelv10.Counterparty{PortId: transfertypes.PortID, ChannelId: "channel-8"},
		ConnectionHops:  []string{"connection-0"},
		Version:         transfertypes.V1,
		UpgradeSequence: 1,
	}
	bz, err := channel.Marshal()
	require.NoError(t, err)
	store.Set(host.ChannelKey(transfertypes.PortID, channelID), bz)

	upgrade := channelv10.Upgrade{
		Fields: channelv10.UpgradeFields{
			Ordering:       channelv10.UNORDERED,
			ConnectionHops: []string{"connection-0"},
			Version:        feeVersion,
		},
		NextSequenceSend: 1,
	}
	bz, err = upgrade.Marshal()
	require.NoError(t, err)
	store.Set(channelv10.ChannelUpgradeKey(transfertypes.PortID, channelID), bz)
}

// v830FullQueueBuy stores a buy the v8.2.0 binary queued for EndBlock, escrowed the way it did: the order coins
// sent to the module, the message stored as its bze.tradebin.QueueMessage encoding under the queue prefix.
func v830FullQueueBuy(t *testing.T, bApp *app.App, ctx sdk.Context, market tradebintypes.Market, owner sdk.AccAddress, amount, price string) {
	t.Helper()
	marketId := market.Base + "/" + market.Quote
	id := fmt.Sprintf("%024d", 1)

	amt, ok := math.NewIntFromString(amount)
	require.True(t, ok)
	orderCoins, err := bApp.TradebinKeeper.GetOrderCoinsWithDust(ctx, tradebintypes.OrderCoinsArguments{
		OrderType: tradebintypes.OrderTypeBuy, OrderPrice: price, OrderAmount: amt, Market: &market, UserAddress: owner.String(),
	})
	require.NoError(t, err)
	require.NoError(t, bApp.BankKeeper.SendCoinsFromAccountToModule(ctx, owner, tradebintypes.ModuleName, sdk.NewCoins(orderCoins.Coin)))

	var bz []byte
	for _, f := range []struct {
		num protowire.Number
		v   string
	}{{1, id}, {2, marketId}, {3, tradebintypes.OrderTypeBuy}, {4, amount}, {5, price}, {8, tradebintypes.OrderTypeBuy}, {9, owner.String()}} {
		bz = protowire.AppendTag(bz, f.num, protowire.BytesType)
		bz = protowire.AppendString(bz, f.v)
	}
	bz = protowire.AppendTag(bz, 6, protowire.VarintType)
	bz = protowire.AppendVarint(bz, 1_699_999_000)

	store := ctx.KVStore(bApp.GetKey(tradebintypes.StoreKey))
	store.Set([]byte(v5.QueueMessagePrefix+marketId+"/"+id+"/"), bz)
	store.Set([]byte(v5.QueueMessageCounterPrefix), []byte("2"))
}

func v830FullMintTo(t *testing.T, bApp *app.App, ctx sdk.Context, addr sdk.AccAddress, coins sdk.Coins) {
	t.Helper()
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	// a plain send: the retired fee collector address is blocked as a recipient of module sends now
	require.NoError(t, bApp.BankKeeper.SendCoins(ctx, authtypes.NewModuleAddress(minttypes.ModuleName), addr, coins))
}

// The real v8.3.0 handler runs once on the real app, from the full v8.2.0 version map, over state that needs every
// one of its steps: a denom trace the transfer store migration cannot parse, a channel FLUSHING an upgrade, a
// queued order book message, a staking reward with a non-canonical prize amount and coins left at the retired
// txfeecollector_burner address. It completes and each step leaves its outcome.
func TestHandler_FullUpgradeFromV820State(t *testing.T) {
	converted := convertedTraces[0]
	valid := validTraces[0]
	bApp, ctx, _ := newTraceApp(t, converted.trace, valid)
	require.NoError(t, bApp.BankKeeper.SetParams(ctx, banktypes.DefaultParams()))

	// a channel FLUSHING an upgrade
	v830FullSetFlushingChannel(t, bApp, ctx, v830FullChannel)

	// a queued buy: 10 base at 2 escrowed 20 quote
	require.NoError(t, bApp.TradebinKeeper.SetParams(ctx, v2types.DefaultParams()))
	market := tradebintypes.Market{Base: v830FullBase, Quote: v830FullQuote, Creator: sdk.AccAddress("full_market_creator_").String()}
	bApp.TradebinKeeper.SetMarket(ctx, market)
	buyer := sdk.AccAddress("full_queued_buyer___")
	escrow := sdk.NewCoins(sdk.NewInt64Coin(v830FullQuote, 20))
	v830FullMintTo(t, bApp, ctx, buyer, escrow)
	v830FullQueueBuy(t, bApp, ctx, market, buyer, "10", "2")
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, buyer).IsZero())

	// a staking reward with a padded prize amount
	bApp.RewardsKeeper.SetStakingReward(ctx, stakingReward("full-upgrade", "0100"))

	// coins left at the retired fee collector, which still holds its module account
	retired := authtypes.NewModuleAddress("txfeecollector_burner")
	burner := authtypes.NewModuleAddress(burnertypes.ModuleName)
	bApp.AccountKeeper.SetAccount(ctx, bApp.AccountKeeper.NewAccount(ctx, authtypes.NewEmptyModuleAccount("txfeecollector_burner")))
	left := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_234), sdk.NewInt64Coin("factory/bze1creator/token", 56))
	v830FullMintTo(t, bApp, ctx, retired, left)
	burnerBefore := bApp.BankKeeper.GetAllBalances(ctx, burner)

	handler := v830.CreateUpgradeHandler(bApp.Configurator(), bApp.ModuleManager, bApp.AccountKeeper, bApp.GuardianKeeper,
		bApp.RatelimitKeeper, bApp.BankKeeper, bApp.TradebinKeeper, bApp.BankKeeper, bApp.RewardsKeeper,
		runtime.NewKVStoreService(bApp.GetKey(transfertypes.StoreKey)), bApp.TransferKeeper,
		runtime.NewKVStoreService(bApp.GetKey(ibcexported.StoreKey)))
	var (
		newVM map[string]uint64
		err   error
	)
	require.NotPanics(t, func() {
		newVM, err = handler(ctx, upgradetypes.Plan{Name: v830.UpgradeName}, v830FullVersionMap(bApp))
	})
	require.NoError(t, err)
	require.Equal(t, map[string]uint64(bApp.ModuleManager.GetVersionMap()), newVM, "every module at its current version")

	// denom traces: the unparsable one converted, the other migrated, no trace record left
	requireDenom(t, bApp, ctx, converted.trace, converted.want)
	requireDenom(t, bApp, ctx, valid, transfertypes.ExtractDenomFromPath(valid.fullPath()))
	requireNoTracesLeft(t, bApp, ctx)
	requireValidExport(t, bApp, ctx)

	// the FLUSHING channel is OPEN on its current version, without its upgrade
	channel, found := bApp.IBCKeeper.ChannelKeeper.GetChannel(ctx, transfertypes.PortID, v830FullChannel)
	require.True(t, found)
	require.Equal(t, channeltypes.OPEN, channel.State)
	require.Equal(t, transfertypes.V1, channel.Version)
	require.False(t, ctx.KVStore(bApp.GetKey(ibcexported.StoreKey)).Has(channelv10.ChannelUpgradeKey(transfertypes.PortID, v830FullChannel)))

	// the queued buy is refunded and the queue is gone
	require.Equal(t, escrow.String(), bApp.BankKeeper.GetAllBalances(ctx, buyer).String())
	it := storetypes.KVStorePrefixIterator(ctx.KVStore(bApp.GetKey(tradebintypes.StoreKey)), []byte(v5.QueueMessagePrefix))
	require.False(t, it.Valid(), "queued messages left after the upgrade")
	require.NoError(t, it.Close())

	// the prize amount is canonical, the rest of the reward as stored
	sr, found := bApp.RewardsKeeper.GetStakingReward(ctx, "full-upgrade")
	require.True(t, found)
	require.Equal(t, stakingReward("full-upgrade", "64"), sr)

	// the retired fee collector is swept to the burner
	require.True(t, bApp.BankKeeper.GetAllBalances(ctx, retired).IsZero())
	require.Equal(t, burnerBefore.Add(left...).String(), bApp.BankKeeper.GetAllBalances(ctx, burner).String())

	// the new modules are initialised and the guardian is set
	require.Equal(t, guardiantypes.NewParams(v830.GuardianAddress), bApp.GuardianKeeper.GetParams(ctx))
	require.NotZero(t, bApp.RatelimitKeeper.GetHourEpoch(ctx).Duration)
}
