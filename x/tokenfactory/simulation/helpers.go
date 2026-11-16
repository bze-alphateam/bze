package simulation

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/bze-alphateam/bze/x/tokenfactory/keeper"
)

// FindAccount find a specific address from an account list
func FindAccount(accs []simtypes.Account, address string) (simtypes.Account, bool) {
	creator, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}
	return simtypes.FindAccount(accs, creator)
}

// RandomMintableDenom returns a random factory denom administered by one of the simulation accounts
// whose minting is still on. Non-mintable denoms are skipped: the admin can no longer mint them.
func RandomMintableDenom(r *rand.Rand, ctx sdk.Context, k keeper.Keeper, accs []simtypes.Account) (simtypes.Account, string, bool) {
	type candidate struct {
		acc   simtypes.Account
		denom string
	}
	var candidates []candidate

	iterator := k.GetAllDenomsIterator(ctx)
	defer func() { _ = iterator.Close() }()
	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Value())
		dAuth, err := k.GetDenomAuthority(ctx, denom)
		if err != nil || !dAuth.GetMintable() || dAuth.GetAdmin() == "" {
			continue
		}

		acc, found := FindAccount(accs, dAuth.GetAdmin())
		if !found {
			continue
		}
		candidates = append(candidates, candidate{acc: acc, denom: denom})
	}

	if len(candidates) == 0 {
		return simtypes.Account{}, "", false
	}

	picked := candidates[r.Intn(len(candidates))]

	return picked.acc, picked.denom, true
}
