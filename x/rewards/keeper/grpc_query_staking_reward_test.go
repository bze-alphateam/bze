package keeper_test

import (
    "strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/bze-alphateam/bze/testutil/nullify"
	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestStakingRewardQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNStakingReward(keeper, ctx, 2)
	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetStakingRewardRequest
		response *types.QueryGetStakingRewardResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetStakingRewardRequest{
			    RewardId: msgs[0].RewardId,
                
			},
			response: &types.QueryGetStakingRewardResponse{StakingReward: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetStakingRewardRequest{
			    RewardId: msgs[1].RewardId,
                
			},
			response: &types.QueryGetStakingRewardResponse{StakingReward: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetStakingRewardRequest{
			    RewardId:strconv.Itoa(100000),
                
			},
			err:     status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.StakingReward(wctx, tc.request)
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

func TestStakingRewardQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNStakingReward(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllStakingRewardRequest {
		return &types.QueryAllStakingRewardRequest{
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
			resp, err := keeper.StakingRewardAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.StakingReward), step)
			require.Subset(t,
            	nullify.Fill(msgs),
            	nullify.Fill(resp.StakingReward),
            )
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.StakingRewardAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.StakingReward), step)
			require.Subset(t,
            	nullify.Fill(msgs),
            	nullify.Fill(resp.StakingReward),
            )
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.StakingRewardAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.StakingReward),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.StakingRewardAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
