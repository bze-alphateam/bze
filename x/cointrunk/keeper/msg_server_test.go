package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/bze-alphateam/bze/x/cointrunk/types"
    "github.com/bze-alphateam/bze/x/cointrunk/keeper"
    keepertest "github.com/bze-alphateam/bze/testutil/keeper"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.CointrunkKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
