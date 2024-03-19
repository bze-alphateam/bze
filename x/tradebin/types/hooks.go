package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type OnMarketOrderFill func(ctx sdk.Context, marketId, amountTraded, userAddress string)
