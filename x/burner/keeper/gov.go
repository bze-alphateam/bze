package keeper

import (
	"github.com/bze-alphateam/bze/x/burner/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

	err = k.SaveBurnedCoins(ctx, coins)
	if err != nil {
		ctx.Logger().Error("error saving burned coins", "error", err)
	}

	err = ctx.EventManager().EmitTypedEvent(&types.CoinsBurnedEvent{Burned: coins.String()})
	if err != nil {
		return err
	}

	return nil
}
