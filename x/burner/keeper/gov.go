package keeper

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

func (k Keeper) HandleBurnCoinsProposal(ctx sdk.Context, proposal *types.BurnCoinsProposal) error {
	moduleAcc := k.accKeeper.GetModuleAccount(ctx, types.ModuleName)
	coins := k.bankKeeper.GetAllBalances(ctx, moduleAcc.GetAddress())
	if coins.IsZero() {
		//nothing to burn at this moment
		return nil
	}

	err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
	if err != nil {
		panic(err)
	}

	var burnedCoins = types.BurnedCoins{
		Burned: coins.String(),
		Height: strconv.FormatInt(ctx.BlockHeader().Height, 10),
	}
	k.SetBurnedCoins(ctx, burnedCoins)

	err = ctx.EventManager().EmitTypedEvent(&types.CoinsBurnedEvent{Burned: burnedCoins.Burned})
	if err != nil {
		return err
	}

	return nil
}
