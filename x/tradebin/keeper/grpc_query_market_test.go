package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/testutil/nullify"
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestMarketQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.TradebinKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNMarket(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetMarketRequest
		response *types.QueryGetMarketResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetMarketRequest{
				Asset1: msgs[0].Asset1,
				Asset2: msgs[0].Asset2,
			},
			response: &types.QueryGetMarketResponse{Market: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetMarketRequest{
				Asset1: msgs[1].Asset1,
				Asset2: msgs[1].Asset2,
			},
			response: &types.QueryGetMarketResponse{Market: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetMarketRequest{
				Asset1: strconv.Itoa(100000),
				Asset2: strconv.Itoa(100000),
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Market(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestMarketQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.TradebinKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNMarket(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllMarketRequest {
		return &types.QueryAllMarketRequest{
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
			resp, err := keeper.MarketAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Market), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Market),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.MarketAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Market), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Market),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.MarketAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Market),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.MarketAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
