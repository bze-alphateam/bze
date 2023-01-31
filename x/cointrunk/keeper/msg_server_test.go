package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.CointrunkKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
