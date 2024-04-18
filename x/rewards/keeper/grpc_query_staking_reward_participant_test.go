package keeper_test

import (
	"strconv"
)

// Prevent strconv unused error
var _ = strconv.IntSize

//
//func TestStakingRewardParticipantQuerySingle(t *testing.T) {
//	keeper, ctx := keepertest.RewardsKeeper(t)
//	wctx := sdk.WrapSDKContext(ctx)
//	msgs := createNStakingRewardParticipant(keeper, ctx, 2)
//	for _, tc := range []struct {
//		desc     string
//		request  *types.QueryGetStakingRewardParticipantRequest
//		response *types.QueryGetStakingRewardParticipantResponse
//		err      error
//	}{
//		{
//			desc: "First",
//			request: &types.QueryGetStakingRewardParticipantRequest{
//				Index: msgs[0].Index,
//			},
//			response: &types.QueryGetStakingRewardParticipantResponse{StakingRewardParticipant: msgs[0]},
//		},
//		{
//			desc: "Second",
//			request: &types.QueryGetStakingRewardParticipantRequest{
//				Index: msgs[1].Index,
//			},
//			response: &types.QueryGetStakingRewardParticipantResponse{StakingRewardParticipant: msgs[1]},
//		},
//		{
//			desc: "KeyNotFound",
//			request: &types.QueryGetStakingRewardParticipantRequest{
//				Index: strconv.Itoa(100000),
//			},
//			err: status.Error(codes.NotFound, "not found"),
//		},
//		{
//			desc: "InvalidRequest",
//			err:  status.Error(codes.InvalidArgument, "invalid request"),
//		},
//	} {
//		t.Run(tc.desc, func(t *testing.T) {
//			response, err := keeper.StakingRewardParticipant(wctx, tc.request)
//			if tc.err != nil {
//				require.ErrorIs(t, err, tc.err)
//			} else {
//				require.NoError(t, err)
//				require.Equal(t,
//					nullify.Fill(tc.response),
//					nullify.Fill(response),
//				)
//			}
//		})
//	}
//}
//
//func TestStakingRewardParticipantQueryPaginated(t *testing.T) {
//	keeper, ctx := keepertest.RewardsKeeper(t)
//	wctx := sdk.WrapSDKContext(ctx)
//	msgs := createNStakingRewardParticipant(keeper, ctx, 5)
//
//	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllStakingRewardParticipantRequest {
//		return &types.QueryAllStakingRewardParticipantRequest{
//			Pagination: &query.PageRequest{
//				Key:        next,
//				Offset:     offset,
//				Limit:      limit,
//				CountTotal: total,
//			},
//		}
//	}
//	t.Run("ByOffset", func(t *testing.T) {
//		step := 2
//		for i := 0; i < len(msgs); i += step {
//			resp, err := keeper.StakingRewardParticipantAll(wctx, request(nil, uint64(i), uint64(step), false))
//			require.NoError(t, err)
//			require.LessOrEqual(t, len(resp.StakingRewardParticipant), step)
//			require.Subset(t,
//				nullify.Fill(msgs),
//				nullify.Fill(resp.StakingRewardParticipant),
//			)
//		}
//	})
//	t.Run("ByKey", func(t *testing.T) {
//		step := 2
//		var next []byte
//		for i := 0; i < len(msgs); i += step {
//			resp, err := keeper.StakingRewardParticipantAll(wctx, request(next, 0, uint64(step), false))
//			require.NoError(t, err)
//			require.LessOrEqual(t, len(resp.StakingRewardParticipant), step)
//			require.Subset(t,
//				nullify.Fill(msgs),
//				nullify.Fill(resp.StakingRewardParticipant),
//			)
//			next = resp.Pagination.NextKey
//		}
//	})
//	t.Run("Total", func(t *testing.T) {
//		resp, err := keeper.StakingRewardParticipantAll(wctx, request(nil, 0, 0, true))
//		require.NoError(t, err)
//		require.Equal(t, len(msgs), int(resp.Pagination.Total))
//		require.ElementsMatch(t,
//			nullify.Fill(msgs),
//			nullify.Fill(resp.StakingRewardParticipant),
//		)
//	})
//	t.Run("InvalidRequest", func(t *testing.T) {
//		_, err := keeper.StakingRewardParticipantAll(wctx, nil)
//		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
//	})
//}
