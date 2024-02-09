package tokenfactory

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/keeper"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
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
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genDenoms := []types.GenesisDenom{}
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
