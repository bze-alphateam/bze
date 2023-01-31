package keeper_test

import (
	"testing"

	testkeeper "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.BurnerKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
