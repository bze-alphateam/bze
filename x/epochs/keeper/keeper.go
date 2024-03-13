package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/bze-alphateam/bze/x/epochs/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	Keeper struct {
		cdc      codec.BinaryCodec
		storeKey sdk.StoreKey
		memKey   sdk.StoreKey

		hooks []types.EpochHook
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	hooks []types.EpochHook,
) *Keeper {

	return &Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		memKey:   memKey,
		hooks:    hooks,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
