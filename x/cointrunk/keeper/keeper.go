package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper  types.BankKeeper
		govKeeper   types.GovKeeper
		accKeeper   types.AccountKeeper
		distrKeeper types.DistrKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper,
	govKeeper types.GovKeeper,
	accKeeper types.AccountKeeper,
	distrKeeper types.DistrKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{

		cdc:         cdc,
		storeKey:    storeKey,
		memKey:      memKey,
		paramstore:  ps,
		bankKeeper:  bankKeeper,
		govKeeper:   govKeeper,
		accKeeper:   accKeeper,
		distrKeeper: distrKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
