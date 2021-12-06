package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/cosmonaut/bzedgev5/testutil/keeper"
	"github.com/cosmonaut/bzedgev5/x/scavenge/keeper"
	"github.com/cosmonaut/bzedgev5/x/scavenge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.ScavengeKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
