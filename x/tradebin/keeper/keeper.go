package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/bze-alphateam/bze/x/tradebin/types"
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
		distrKeeper types.DistrKeeper

		onOrderFillHooks []types.OnMarketOrderFill
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
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
		distrKeeper: distrKeeper,
	}
}

func (k *Keeper) SetOnOrderFillHooks(hooks []types.OnMarketOrderFill) {
	k.onOrderFillHooks = hooks
}

func (k Keeper) GetOnOrderFillHooks() []types.OnMarketOrderFill {
	return k.onOrderFillHooks
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) largeZeroFillId(id uint64) string {
	return fmt.Sprintf("%024d", id)
}

func (k Keeper) smallZeroFillId(id uint64) string {
	return fmt.Sprintf("%012d", id)
}
