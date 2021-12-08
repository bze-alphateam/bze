package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/bze-alphateam/bzedgev5/testutil/keeper"
	"github.com/bze-alphateam/bzedgev5/x/scavenge/keeper"
	"github.com/bze-alphateam/bzedgev5/x/scavenge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.ScavengeKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
