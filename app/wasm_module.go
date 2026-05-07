package app

import (
	"fmt"

	wasm "github.com/CosmWasm/wasmd/x/wasm"
	"github.com/CosmWasm/wasmd/x/wasm/exported"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	txfeekeeper "github.com/bze-alphateam/bze/x/txfeecollector/keeper"
	txfeetypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// FeeWrappedWasmAppModule embeds wasmd's AppModule and overrides
// RegisterServices to install a fee-charging wrapper around the default
// wasm MsgServer. Everything else (InitGenesis, BeginBlock, EndBlock,
// CLI, query gateway, simulation) is inherited unchanged via embedding.
//
// Maintenance note: the body of RegisterServices below mirrors wasmd's
// own module.go::RegisterServices (msg server + query server + N
// migrations). Verify it stays in sync when bumping wasmd — if wasmd
// adds a new migration, this method must register it too.
type FeeWrappedWasmAppModule struct {
	wasm.AppModule

	keeper         *wasmkeeper.Keeper
	legacySubspace exported.Subspace

	txfeeKeeper  *txfeekeeper.Keeper
	bankKeeper   txfeetypes.BankKeeper
	govAuthority string
}

func NewFeeWrappedWasmAppModule(
	inner wasm.AppModule,
	keeper *wasmkeeper.Keeper,
	legacySubspace exported.Subspace,
	txfeeKeeper *txfeekeeper.Keeper,
	bankKeeper txfeetypes.BankKeeper,
	govAuthority string,
) FeeWrappedWasmAppModule {
	return FeeWrappedWasmAppModule{
		AppModule:      inner,
		keeper:         keeper,
		legacySubspace: legacySubspace,
		txfeeKeeper:    txfeeKeeper,
		bankKeeper:     bankKeeper,
		govAuthority:   govAuthority,
	}
}

func (am FeeWrappedWasmAppModule) RegisterServices(cfg module.Configurator) {
	innerMsgServer := wasmkeeper.NewMsgServerImpl(am.keeper)
	wrapped := NewFeeChargingWasmMsgServer(innerMsgServer, am.txfeeKeeper, am.bankKeeper, am.govAuthority)
	wasmtypes.RegisterMsgServer(cfg.MsgServer(), wrapped)

	wasmtypes.RegisterQueryServer(cfg.QueryServer(), wasmkeeper.Querier(am.keeper))

	m := wasmkeeper.NewMigrator(*am.keeper, am.legacySubspace)
	if err := cfg.RegisterMigration(wasmtypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to register wasm migration 1->2: %v", err))
	}
	if err := cfg.RegisterMigration(wasmtypes.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to register wasm migration 2->3: %v", err))
	}
	if err := cfg.RegisterMigration(wasmtypes.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to register wasm migration 3->4: %v", err))
	}
}
