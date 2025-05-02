package tokenfactory

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tokenfactory/keeper"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
	for _, genDenom := range genState.GetFactoryDenoms() {
		creator, _, err := types.DeconstructDenom(genDenom.GetDenom())
		if err != nil {
			panic(err)
		}
		err = k.CreateDenomAfterValidation(ctx, creator, genDenom.GetDenom())
		if err != nil {
			panic(err)
		}
		err = k.SetDenomAuthority(ctx, genDenom.GetDenom(), genDenom.GetDenomAuthority())
		if err != nil {
			panic(err)
		}
	}

	k.InitGenesis(ctx)
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genDenoms := make([]types.GenesisDenom, 0)
	iterator := k.GetAllDenomsIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Value())
		dAuth, err := k.GetDenomAuthority(ctx, denom)
		if err != nil {
			panic(err)
		}

		genDenoms = append(genDenoms, types.GenesisDenom{
			Denom:          denom,
			DenomAuthority: dAuth,
		})
	}

	genesis.FactoryDenoms = genDenoms
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
