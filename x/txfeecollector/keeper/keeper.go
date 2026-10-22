package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/txfeecollector/types"
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
		tradeKeeper   types.TradeKeeper
		distrKeeper   types.DistrKeeper

		// denomCpProvider resolves the community pool a denom's fees are forwarded to. nil means
		// no denom has one.
		denomCpProvider types.DenomCommunityPoolProvider

		// escrowChecker recognises the IBC transfer escrow accounts, which never pay the transfer fee. It is set after
		// the IBC keepers are built (SetEscrowAddressChecker); nil disables the transfer fee.
		escrowChecker types.EscrowAddressChecker
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	tradeKeeper types.TradeKeeper,
	distrKeeper types.DistrKeeper,
	denomCpProvider types.DenomCommunityPoolProvider,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:             cdc,
		storeService:    storeService,
		authority:       authority,
		logger:          logger,
		bankKeeper:      bankKeeper,
		accountKeeper:   accountKeeper,
		tradeKeeper:     tradeKeeper,
		distrKeeper:     distrKeeper,
		denomCpProvider: denomCpProvider,
	}
}

// SetEscrowAddressChecker sets the checker the transfer fee uses to exempt IBC transfer escrow accounts. Until it is
// set the transfer fee charges nothing.
func (k *Keeper) SetEscrowAddressChecker(checker types.EscrowAddressChecker) {
	k.escrowChecker = checker
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
