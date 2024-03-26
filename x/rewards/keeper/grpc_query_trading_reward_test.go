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
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestTradingRewardQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNTradingReward(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetTradingRewardRequest
		response *types.QueryGetTradingRewardResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetTradingRewardRequest{
				RewardId: msgs[0].RewardId,
			},
			response: &types.QueryGetTradingRewardResponse{TradingReward: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetTradingRewardRequest{
				RewardId: msgs[1].RewardId,
			},
			response: &types.QueryGetTradingRewardResponse{TradingReward: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetTradingRewardRequest{
				RewardId: strconv.Itoa(100000),
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.TradingReward(wctx, tc.request)
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

func TestTradingRewardQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNTradingReward(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllTradingRewardRequest {
		return &types.QueryAllTradingRewardRequest{
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
			resp, err := keeper.TradingRewardAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.TradingReward), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.TradingReward),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.TradingRewardAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.TradingReward), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.TradingReward),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.TradingRewardAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.TradingReward),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.TradingRewardAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}