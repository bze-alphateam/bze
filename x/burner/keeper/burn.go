package keeper

import (
	"github.com/bze-alphateam/bze/bzeutils"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) burnModuleCoins(ctx sdk.Context) error {
	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
	allCoins := k.bankKeeper.GetAllBalances(ctx, moduleAcc.GetAddress())
	if allCoins.IsZero() {
		//nothing to burn at this moment
		return nil
	}

	// filter out IBC tokens
	coins := sdk.NewCoins()
	for _, c := range allCoins {
		//make sure IBC and LP Tokens are not burned
		if bzeutils.IsIBCDenom(c.Denom) || bzeutils.IsLpTokenDenom(c.Denom) {
			//TODO: swap IBC tokens for BZE to burn them
			//TODO: take into consideration that external apps might not have trading module
			continue
		}

		coins = coins.Add(c)
	}

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

	k.Logger().With("coins", coins.String()).Info("coins successfully burned")

	return nil
}
