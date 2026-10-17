package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	daotypes "github.com/bze-alphateam/bze/x/dao/types"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// The app registers x/dao on the x/rewards Denom Reward hooks: joining a
// Denom Reward a DAO votes with writes the join record, exiting deletes it.
func TestDaoDenomRewardHooks_Wired(t *testing.T) {
	bApp, ctx, admin, _ := newAdminDaoApp(t)
	staker := sdk.AccAddress("dao_staker__________")

	bApp.RewardsKeeper.SetDenomReward(ctx, rewardstypes.DenomReward{StakingDenom: "ubze", Lock: 5, StakedAmount: math.ZeroInt()})
	_, err := runAsTx(bApp, ctx, &daotypes.MsgCreateDao{
		Creator:    admin.String(),
		Admin:      admin.String(),
		Type:       daotypes.DaoType_DAO_TYPE_STAKE,
		Name:       "stakers",
		Config:     newAdminDaoConfig(),
		Electorate: &daotypes.StakeElectorate{DrDenom: "ubze"},
	})
	require.NoError(t, err)
	// the lock is checked only when the source joins the DAO; dropping it here
	// lets the exit pay out at once, with no epoch module state needed
	bApp.RewardsKeeper.SetDenomReward(ctx, rewardstypes.DenomReward{StakingDenom: "ubze", Lock: 0, StakedAmount: math.ZeroInt()})

	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, staker, coins))

	ctx = ctx.WithBlockHeight(3)
	_, err = runAsTx(bApp, ctx, rewardstypes.NewMsgJoinDenomReward(staker.String(), "ubze", math.NewInt(400)))
	require.NoError(t, err)

	record, found := bApp.DaoKeeper.GetJoinRecord(ctx, daotypes.DenomRewardSource("ubze"), staker)
	require.True(t, found)
	require.Equal(t, int64(3), record.Height)

	_, err = runAsTx(bApp, ctx, &rewardstypes.MsgExitDenomReward{Creator: staker.String(), Denom: "ubze"})
	require.NoError(t, err)
	_, found = bApp.DaoKeeper.GetJoinRecord(ctx, daotypes.DenomRewardSource("ubze"), staker)
	require.False(t, found)
}
