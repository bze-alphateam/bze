package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// Keeper is the daodao module's state keeper. It owns the module's KV store
// and depends on x/auth (AccountKeeper) for DAO BaseAccount registration,
// x/bank (BankKeeper) for fee routing and DAO balance reads,
// x/distribution (DistrKeeper) for routing creation fees to the community
// pool when configured, and x/rewards (RewardsKeeper) for proxied voting
// power on REWARD_STAKED DAOs.
type Keeper struct {
	// cdc is the full Codec (not just BinaryCodec) because Epic 5's
	// proposal-bundle signer check uses GetMsgV1Signers, which lives on
	// the Codec interface. The actual passed value is a *ProtoCodec which
	// implements both BinaryCodec and Codec, so this is backward compatible
	// with every existing call site that uses Marshal/Unmarshal.
	cdc          codec.Codec
	storeService store.KVStoreService
	logger       log.Logger

	// authority is the address allowed to execute MsgUpdateParams. Typically
	// the x/gov module account.
	authority string

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	distrKeeper   types.DistrKeeper
	rewardsKeeper types.RewardsKeeper

	// msgRouter dispatches a proposal's msgs[] bundle at execution time
	// (Epic 5). Wired in app/app.go AFTER baseapp is constructed via
	// SetMsgRouter, because the keeper is built in depinject BEFORE
	// baseapp.MsgServiceRouter exists. Nil-safe: MsgExecuteProposal
	// surfaces a clear error if it's not wired (which only happens in
	// tests that don't need execution).
	msgRouter types.MsgRouter
}

// NewKeeper constructs a daodao Keeper. Panics if `authority` is not a valid
// bech32 address — this is a programmer error in module wiring.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distrKeeper types.DistrKeeper,
	rewardsKeeper types.RewardsKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		authority:     authority,
		logger:        logger,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
		distrKeeper:   distrKeeper,
		rewardsKeeper: rewardsKeeper,
	}
}

// SetMsgRouter wires the baseapp MsgServiceRouter (or a fake in tests) so
// the keeper can dispatch proposal msgs[]. Must be called once at app
// init, after baseapp is built. Repeated calls overwrite the previous
// router — used by tests that swap a fake in / out between cases.
func (k *Keeper) SetMsgRouter(r types.MsgRouter) {
	k.msgRouter = r
}

// GetAuthority returns the module's authority address (chain gov).
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-namespaced logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
