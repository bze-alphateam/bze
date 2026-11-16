package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/burner/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		bankKeeper    types.BankKeeper
		accountKeeper types.AccountKeeper
		epochKeeper   types.EpochKeeper
		tradeKeeper   types.TradeKeeper

		// guardianKeeper tells whether the protocol automation is paused. nil means never paused.
		guardianKeeper types.GuardianKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	epochKeeper types.EpochKeeper,
	tradeKeeper types.TradeKeeper,
	guardianKeeper types.GuardianKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,

		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
		epochKeeper:   epochKeeper,
		tradeKeeper:   tradeKeeper,

		guardianKeeper: guardianKeeper,
	}
}

// IsAutomationPaused reports whether the guardian paused the protocol automation: the periodic burn then leaves the
// burner's coins where they are. A keeper without a guardian keeper is never paused.
func (k Keeper) IsAutomationPaused(ctx sdk.Context) bool {
	if k.guardianKeeper == nil {
		return false
	}

	return k.guardianKeeper.IsAutomationPaused(ctx)
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
