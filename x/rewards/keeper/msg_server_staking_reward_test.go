package keeper_test

import (
    "strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

    keepertest "github.com/bze-alphateam/bze/testutil/keeper"
    "github.com/bze-alphateam/bze/x/rewards/keeper"
    "github.com/bze-alphateam/bze/x/rewards/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestStakingRewardMsgServerCreate(t *testing.T) {
	k, ctx := keepertest.RewardsKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)
	creator := "A"
	for i := 0; i < 5; i++ {
		expected := &types.MsgCreateStakingReward{Creator: creator,
		    RewardId: strconv.Itoa(i),
            
		}
		_, err := srv.CreateStakingReward(wctx, expected)
		require.NoError(t, err)
		rst, found := k.GetStakingReward(ctx,
		    expected.RewardId,
            
		)
		require.True(t, found)
		require.Equal(t, expected.Creator, rst.Creator)
	}
}

func TestStakingRewardMsgServerUpdate(t *testing.T) {
	creator := "A"

	for _, tc := range []struct {
		desc    string
		request *types.MsgUpdateStakingReward
		err     error
	}{
		{
			desc:    "Completed",
			request: &types.MsgUpdateStakingReward{Creator: creator,
			    RewardId: strconv.Itoa(0),
                
			},
		},
		{
			desc:    "Unauthorized",
			request: &types.MsgUpdateStakingReward{Creator: "B",
			    RewardId: strconv.Itoa(0),
                
			},
			err:     sdkerrors.ErrUnauthorized,
		},
		{
			desc:    "KeyNotFound",
			request: &types.MsgUpdateStakingReward{Creator: creator,
			    RewardId: strconv.Itoa(100000),
                
			},
			err:     sdkerrors.ErrKeyNotFound,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			k, ctx := keepertest.RewardsKeeper(t)
			srv := keeper.NewMsgServerImpl(*k)
			wctx := sdk.WrapSDKContext(ctx)
			expected := &types.MsgCreateStakingReward{Creator: creator,
			    RewardId: strconv.Itoa(0),
                
			}
			_, err := srv.CreateStakingReward(wctx, expected)
			require.NoError(t, err)

			_, err = srv.UpdateStakingReward(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				rst, found := k.GetStakingReward(ctx,
				    expected.RewardId,
                    
				)
				require.True(t, found)
				require.Equal(t, expected.Creator, rst.Creator)
			}
		})
	}
}

func TestStakingRewardMsgServerDelete(t *testing.T) {
	creator := "A"

	for _, tc := range []struct {
		desc    string
		request *types.MsgDeleteStakingReward
		err     error
	}{
		{
			desc:    "Completed",
			request: &types.MsgDeleteStakingReward{Creator: creator,
			    RewardId: strconv.Itoa(0),
                
			},
		},
		{
			desc:    "Unauthorized",
			request: &types.MsgDeleteStakingReward{Creator: "B",
			    RewardId: strconv.Itoa(0),
                
			},
			err:     sdkerrors.ErrUnauthorized,
		},
		{
			desc:    "KeyNotFound",
			request: &types.MsgDeleteStakingReward{Creator: creator,
			    RewardId: strconv.Itoa(100000),
                
			},
			err:     sdkerrors.ErrKeyNotFound,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			k, ctx := keepertest.RewardsKeeper(t)
			srv := keeper.NewMsgServerImpl(*k)
			wctx := sdk.WrapSDKContext(ctx)

			_, err := srv.CreateStakingReward(wctx, &types.MsgCreateStakingReward{Creator: creator,
			    RewardId: strconv.Itoa(0),
                
			})
			require.NoError(t, err)
			_, err = srv.DeleteStakingReward(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				_, found := k.GetStakingReward(ctx,
				    tc.request.RewardId,
                    
				)
				require.False(t, found)
			}
		})
	}
}
