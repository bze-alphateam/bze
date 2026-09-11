package rewards_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/rewards/types"
)

// countingHooks records the staking-reward hook calls it receives.
type countingHooks struct {
	joins    []string
	increase int
	exits    int
	removals int
}

func (h *countingHooks) AfterStakingRewardJoin(_ sdk.Context, rewardId, address string, _ math.Int, _ string) error {
	h.joins = append(h.joins, rewardId+"/"+address)
	return nil
}

func (h *countingHooks) AfterStakingRewardIncrease(_ sdk.Context, _, _ string, _, _ math.Int, _ string) error {
	h.increase++
	return nil
}

func (h *countingHooks) AfterStakingRewardExit(_ sdk.Context, _, _ string, _ math.Int, _ string) error {
	h.exits++
	return nil
}

func (h *countingHooks) BeforeStakingRewardRemoval(_ sdk.Context, _ string) error {
	h.removals++
	return nil
}

// TestHooksRegisteredAfterRegisterServices_ReachJoinStaking reproduces the app's wiring order:
// RegisterServices (inside appBuilder.Build) constructs the msg server first, and only afterwards
// does app.go register hooks on the keeper. A msg server holding a by-value keeper copy would have
// frozen the nil hooks at registration time; with the pointer shared, a MsgJoinStaking dispatched
// through the real msg service router fires the hook registered later.
func TestHooksRegisteredAfterRegisterServices_ReachJoinStaking(t *testing.T) {
	f := newWiringFixture(t)
	require.NoError(t, f.k.SetParams(f.ctx, types.DefaultParams()))

	// 1. RegisterServices — the msg server is built here, before any hook exists
	_, _, msr, _ := f.manager(t)

	// 2. hooks are registered afterwards, exactly like app.go does after appBuilder.Build
	hooks := &countingHooks{}
	f.k.SetHooks(hooks)

	creator := sdk.AccAddress("hooks-wiring-creator")
	const rewardId = "hooks-wiring-reward"
	f.k.SetStakingReward(f.ctx, types.StakingReward{
		RewardId:         rewardId,
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         100,
		Lock:             7,
		StakedAmount:     "0",
		DistributedStake: "0",
	})

	f.bank.EXPECT().
		SpendableCoins(gomock.Any(), creator).
		Return(sdk.NewCoins(sdk.NewInt64Coin("ubze", 10_000))).
		Times(1)
	f.bank.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), creator, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 500))).
		Return(nil).
		Times(1)

	// 3. the message goes through the router the way BaseApp delivers a tx
	msg := &types.MsgJoinStaking{Creator: creator.String(), RewardId: rewardId, Amount: "500"}
	handler := msr.Handler(msg)
	require.NotNil(t, handler, "MsgJoinStaking must be routable after RegisterServices")

	_, err := handler(f.ctx, msg)
	require.NoError(t, err)

	require.Equal(t, []string{rewardId + "/" + creator.String()}, hooks.joins, "the hook registered after RegisterServices must fire")
	require.Zero(t, hooks.increase)
	require.Zero(t, hooks.exits)
	require.Zero(t, hooks.removals)

	participant, found := f.k.GetStakingRewardParticipant(f.ctx, creator.String(), rewardId)
	require.True(t, found)
	require.Equal(t, "500", participant.Amount)
}
