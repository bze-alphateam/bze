package app

import (
	wasm "github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	appmodule "cosmossdk.io/core/appmodule"
	storetypes "cosmossdk.io/store/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
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
		DefaultNodeHome,
		nodeConfig,
		wasmtypes.VMConfig{},
		availableCapabilities,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		GetWasmOpts(app)...,
	)

	app.ScopedWasmKeeper = scopedWasmKeeper

	// Add wasm IBC port to the IBC router (before it's sealed)
	wasmIBCModule := wasm.NewIBCHandler(app.WasmKeeper, app.IBCKeeper.ChannelKeeper, app.IBCFeeKeeper)
	app.ibcRouter.AddRoute(wasmtypes.ModuleName, wasmIBCModule)

	// Register wasm app module
	if err := app.RegisterModules(
		wasm.NewAppModule(
			app.appCodec,
			&app.WasmKeeper,
			app.StakingKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.MsgServiceRouter(),
			app.GetSubspace(wasmtypes.ModuleName),
		),
	); err != nil {
		return err
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
