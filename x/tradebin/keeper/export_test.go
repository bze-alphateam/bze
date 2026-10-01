package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// SwapTokens exposes swapTokens to the keeper_test package so the halted-denom no-swap invariant can
// be pinned at the choke point itself, independently of the msg-server guards in front of it.
func (k Keeper) SwapTokens(ctx sdk.Context, input sdk.Coin, pool *types.LiquidityPool) (sdk.Coin, error) {
	return k.swapTokens(ctx, input, pool)
}
