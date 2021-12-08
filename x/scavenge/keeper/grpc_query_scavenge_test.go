package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/bze-alphateam/bzedgev5/testutil/keeper"
	"github.com/bze-alphateam/bzedgev5/x/scavenge/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestScavengeQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.ScavengeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNScavenge(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetScavengeRequest
		response *types.QueryGetScavengeResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetScavengeRequest{
				Index: msgs[0].Index,
			},
			response: &types.QueryGetScavengeResponse{Scavenge: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetScavengeRequest{
				Index: msgs[1].Index,
			},
			response: &types.QueryGetScavengeResponse{Scavenge: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetScavengeRequest{
				Index: strconv.Itoa(100000),
			},
			err: status.Error(codes.InvalidArgument, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Scavenge(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.Equal(t, tc.response, response)
			}
		})
	}
}

func TestScavengeQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.ScavengeKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNScavenge(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllScavengeRequest {
		return &types.QueryAllScavengeRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.ScavengeAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Scavenge), step)
			require.Subset(t, msgs, resp.Scavenge)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.ScavengeAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Scavenge), step)
			require.Subset(t, msgs, resp.Scavenge)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.ScavengeAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.ScavengeAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
