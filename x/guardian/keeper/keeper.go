package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	logger       log.Logger

	// the address capable of executing a MsgUpdateParams message and of
	// signing the guardian messages. Typically, this should be the x/gov
	// module account.
	authority string

	bankKeeper types.BankKeeper

	// rateLimitKeeper is wired after the depinject build, see SetRateLimitKeeper.
	rateLimitKeeper types.RateLimitKeeper

	// circuitKeeper and msgRouter are wired after the depinject build, see
	// SetCircuitKeeper and SetMsgRouter.
	circuitKeeper types.CircuitKeeper
	msgRouter     types.MsgRouter

	// tradebinKeeper is wired after the depinject build, see SetTradebinKeeper.
	tradebinKeeper types.TradebinKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	bankKeeper types.BankKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,
		bankKeeper:   bankKeeper,
	}
}

// SetRateLimitKeeper wires the rate-limiting keeper. The ibc-apps module is not
// a depinject module: the app builds its keeper after the depinject container,
// so it cannot be an input of ProvideModule.
func (k *Keeper) SetRateLimitKeeper(rateLimitKeeper types.RateLimitKeeper) {
	k.rateLimitKeeper = rateLimitKeeper
}

// SetCircuitKeeper wires the x/circuit disable list. The app adapts the
// circuit keeper, whose disable list is a store collection and not a method.
func (k *Keeper) SetCircuitKeeper(circuitKeeper types.CircuitKeeper) {
	k.circuitKeeper = circuitKeeper
}

// SetMsgRouter wires the app's message router, which only exists once the app
// is built.
func (k *Keeper) SetMsgRouter(msgRouter types.MsgRouter) {
	k.msgRouter = msgRouter
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
