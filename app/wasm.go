package app

import (
	"fmt"

	wasm "github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	appmodule "cosmossdk.io/core/appmodule"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee"
	"github.com/spf13/cast"
)

// availableCapabilities lists all CosmWasm capabilities supported by this chain.
var availableCapabilities = []string{
	"iterator",
	"staking",
	"stargate",
	"cosmwasm_1_1",
	"cosmwasm_1_2",
	"cosmwasm_1_3",
	"cosmwasm_1_4",
	"cosmwasm_2_0",
	"cosmwasm_2_1",
	"cosmwasm_2_2",
}

// registerWasmModule registers the CosmWasm module and its keeper.
// This follows the same manual-wiring pattern as registerIBCModules in ibc.go,
// since wasmd does not support depinject.
func (app *App) registerWasmModule(appOpts servertypes.AppOptions) error {
	// Register store key
	if err := app.RegisterStores(
		storetypes.NewKVStoreKey(wasmtypes.StoreKey),
	); err != nil {
		return err
	}

	// Scope capability for wasm
	scopedWasmKeeper := app.CapabilityKeeper.ScopeToModule(wasmtypes.ModuleName)

	// Read wasm node config from app options
	nodeConfig, err := wasm.ReadNodeConfig(appOpts)
	if err != nil {
		return err
	}

	// Store config for ante handler setup
	app.wasmNodeConfig = nodeConfig

	// Resolve the actual node home from the runtime --home flag so the wasmvm
	// on-disk cache lands under the node's real data dir (wasmd joins
	// homeDir + "wasm"). Falling back to DefaultNodeHome keeps instantiation
	// paths that don't set --home (some test harnesses) working with an
	// absolute path instead of a relative "wasm" dir.
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	if homePath == "" {
		homePath = DefaultNodeHome
	}

	// Create wasm keeper
	app.WasmKeeper = wasmkeeper.NewKeeper(
		app.appCodec,
		runtime.NewKVStoreService(app.GetKey(wasmtypes.StoreKey)),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		distrkeeper.NewQuerier(app.DistrKeeper),
		app.IBCFeeKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		scopedWasmKeeper,
		app.TransferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		homePath,
		nodeConfig,
		// VMConfig holds consensus-critical Wasm static-validation limits
		// (WasmLimits). Zero value = use wasmvm's internal defaults, which
		// is what most chains do. Setting custom limits here is a hard fork
		// — all nodes must agree — so leave default unless we have a
		// specific reason to tighten binary validation.
		wasmtypes.VMConfig{},
		availableCapabilities,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		GetWasmOpts(app)...,
	)

	app.ScopedWasmKeeper = scopedWasmKeeper

	// Add wasm IBC port to the IBC router (before it's sealed).
	// Wrap with the ICS-29 fee middleware to match the transfer/ICA stacks
	// (see ibc.go) and the wasmd v0.54.x reference app. Without the wrapper
	// wasm IBC channels can't negotiate the ics29 fee version and a
	// fee-enabled handshake from a counterparty would fail.
	wasmIBCModule := wasm.NewIBCHandler(app.WasmKeeper, app.IBCKeeper.ChannelKeeper, app.IBCFeeKeeper)
	wasmStack := ibcfee.NewIBCMiddleware(wasmIBCModule, app.IBCFeeKeeper)
	app.ibcRouter.AddRoute(wasmtypes.ModuleName, wasmStack)

	// Register the legacy params subspace for wasm (mirrors the wasmd reference
	// app). It is only consumed by wasmd's Migrate1to2 — which never runs on this
	// chain since fresh installs start at consensus version 4 — but GetSubspace
	// silently returns a zero-value subspace when nothing is registered, which
	// would panic if that migration were ever exercised.
	app.ParamsKeeper.Subspace(wasmtypes.ModuleName)

	// Register wasm app module wrapped with a fee-charging MsgServer.
	// The wrapper is the only universal capture point for the cw deploy
	// fee — it covers direct user txs, authz, contract submsgs, ICA host
	// dispatch, and gov proposals because all paths funnel through the
	// wasm MsgServer registered on the MsgServiceRouter.
	wasmSubspace := app.GetSubspace(wasmtypes.ModuleName)
	innerWasmModule := wasm.NewAppModule(
		app.appCodec,
		&app.WasmKeeper,
		app.StakingKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.MsgServiceRouter(),
		wasmSubspace,
	)
	wrappedWasmModule := NewFeeWrappedWasmAppModule(
		innerWasmModule,
		&app.WasmKeeper,
		wasmSubspace,
		app.TxfeecollectorKeeper,
		app.BankKeeper,
		app.WasmKeeper.GetAuthority(),
	)
	if err := app.RegisterModules(wrappedWasmModule); err != nil {
		return err
	}

	// Register the wasm snapshot extension so state-sync snapshots include the
	// contract code stored in wasmvm's on-disk file store (it is NOT part of the
	// IAVL store). Without this, a node that joins via state sync receives the
	// code checksums but not the code itself and fails on the first contract
	// execution.
	if manager := app.SnapshotManager(); manager != nil {
		if err := manager.RegisterExtensions(
			wasmkeeper.NewWasmSnapshotter(app.CommitMultiStore(), &app.WasmKeeper),
		); err != nil {
			return fmt.Errorf("failed to register wasm snapshot extension: %w", err)
		}
	}

	return nil
}

// RegisterWasm registers wasm module on the client side for autocli/CLI support.
// Since wasmd does not support dependency injection, we need to manually register
// the module on the client side (same pattern as RegisterIBC in ibc.go).
func RegisterWasm(registry cdctypes.InterfaceRegistry) map[string]appmodule.AppModule {
	modules := map[string]appmodule.AppModule{
		wasmtypes.ModuleName: wasm.AppModule{},
	}

	for name, m := range modules {
		module.CoreAppModuleBasicAdaptor(name, m).RegisterInterfaces(registry)
	}

	return modules
}

// GetWasmOpts returns wasm keeper options.
// All SDK messages routed through MsgServiceRouter are available to contracts by default.
// Security is enforced by the SDKMessageHandler which validates that the contract
// address is the signer of every dispatched message — authority-only messages
// (e.g., MsgUpdateParams) are rejected because their required signer is the
// governance module, not the contract.
func GetWasmOpts(_ *App) []wasmkeeper.Option {
	return []wasmkeeper.Option{}
}
