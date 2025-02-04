package keeper

import (
	"github.com/bze-alphateam/bze/x/burner/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) HandleBurnCoinsProposal(ctx sdk.Context, _ *types.BurnCoinsProposal) error {
	return k.burnModuleCoins(ctx)
}
