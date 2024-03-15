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

func TestTradingRewardMsgServerCreate(t *testing.T) {
	k, ctx := keepertest.RewardsKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)
	creator := "A"
	for i := 0; i < 5; i++ {
		expected := &types.MsgCreateTradingReward{Creator: creator,
		    RewardId: strconv.Itoa(i),
            
		}
		_, err := srv.CreateTradingReward(wctx, expected)
		require.NoError(t, err)
		rst, found := k.GetTradingReward(ctx,
		    expected.RewardId,
            
		)
		require.True(t, found)
		require.Equal(t, expected.Creator, rst.Creator)
	}
}

func TestTradingRewardMsgServerUpdate(t *testing.T) {
	creator := "A"

	for _, tc := range []struct {
		desc    string
		request *types.MsgUpdateTradingReward
		err     error
	}{
		{
			desc:    "Completed",
			request: &types.MsgUpdateTradingReward{Creator: creator,
			    RewardId: strconv.Itoa(0),
                
			},
		},
		{
			desc:    "Unauthorized",
			request: &types.MsgUpdateTradingReward{Creator: "B",
			    RewardId: strconv.Itoa(0),
                
			},
			err:     sdkerrors.ErrUnauthorized,
		},
		{
			desc:    "KeyNotFound",
			request: &types.MsgUpdateTradingReward{Creator: creator,
			    RewardId: strconv.Itoa(100000),
                
			},
			err:     sdkerrors.ErrKeyNotFound,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			k, ctx := keepertest.RewardsKeeper(t)
			srv := keeper.NewMsgServerImpl(*k)
			wctx := sdk.WrapSDKContext(ctx)
			expected := &types.MsgCreateTradingReward{Creator: creator,
			    RewardId: strconv.Itoa(0),
                
			}
			_, err := srv.CreateTradingReward(wctx, expected)
			require.NoError(t, err)

			_, err = srv.UpdateTradingReward(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				rst, found := k.GetTradingReward(ctx,
				    expected.RewardId,
                    
				)
				require.True(t, found)
				require.Equal(t, expected.Creator, rst.Creator)
			}
		})
	}
}

func TestTradingRewardMsgServerDelete(t *testing.T) {
	creator := "A"

	for _, tc := range []struct {
		desc    string
		request *types.MsgDeleteTradingReward
		err     error
	}{
		{
			desc:    "Completed",
			request: &types.MsgDeleteTradingReward{Creator: creator,
			    RewardId: strconv.Itoa(0),
                
			},
		},
		{
			desc:    "Unauthorized",
			request: &types.MsgDeleteTradingReward{Creator: "B",
			    RewardId: strconv.Itoa(0),
                
			},
			err:     sdkerrors.ErrUnauthorized,
		},
		{
			desc:    "KeyNotFound",
			request: &types.MsgDeleteTradingReward{Creator: creator,
			    RewardId: strconv.Itoa(100000),
                
			},
			err:     sdkerrors.ErrKeyNotFound,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			k, ctx := keepertest.RewardsKeeper(t)
			srv := keeper.NewMsgServerImpl(*k)
			wctx := sdk.WrapSDKContext(ctx)

			_, err := srv.CreateTradingReward(wctx, &types.MsgCreateTradingReward{Creator: creator,
			    RewardId: strconv.Itoa(0),
                
			})
			require.NoError(t, err)
			_, err = srv.DeleteTradingReward(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				_, found := k.GetTradingReward(ctx,
				    tc.request.RewardId,
                    
				)
				require.False(t, found)
			}
		})
	}
}
