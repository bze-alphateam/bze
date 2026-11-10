package v830_test

import (
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"

	v830 "github.com/bze-alphateam/bze/app/upgrades/v830"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// fakeStakingRewards is an in-memory staking reward store that records which rewards were written.
type fakeStakingRewards struct {
	byId    map[string]rewardstypes.StakingReward
	order   []string
	written []string
}

func newFakeStakingRewards(rewards ...rewardstypes.StakingReward) *fakeStakingRewards {
	f := &fakeStakingRewards{byId: map[string]rewardstypes.StakingReward{}}
	for _, sr := range rewards {
		f.byId[sr.RewardId] = sr
		f.order = append(f.order, sr.RewardId)
	}
	return f
}

func (f *fakeStakingRewards) GetAllStakingReward(sdk.Context) []rewardstypes.StakingReward {
	list := make([]rewardstypes.StakingReward, 0, len(f.order))
	for _, id := range f.order {
		list = append(list, f.byId[id])
	}
	return list
}

func (f *fakeStakingRewards) SetStakingReward(_ sdk.Context, sr rewardstypes.StakingReward) {
	if f.byId == nil {
		f.byId = map[string]rewardstypes.StakingReward{}
	}
	f.byId[sr.RewardId] = sr
	f.written = append(f.written, sr.RewardId)
}

func runStakingRewardNormalisation(t *testing.T, rewards v830.RewardsKeeper) {
	t.Helper()
	cfg, mm, ctx := guardianSetup(t)
	_, err := v830.CreateUpgradeHandler(cfg, mm, &fakeGuardian{}, acceptingRateLimits(t), emptyBank(t), noPools{}, nil, rewards)(
		ctx, upgradetypes.Plan{Name: v830.UpgradeName}, module.VersionMap{fakeModuleName: 2},
	)
	require.NoError(t, err)
}

func stakingReward(id, prizeAmount string) rewardstypes.StakingReward {
	return rewardstypes.StakingReward{
		RewardId:         id,
		PrizeAmount:      prizeAmount,
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         10,
		Payouts:          3,
		MinStake:         5,
		Lock:             2,
		StakedAmount:     "1234",
		DistributedStake: "0.5",
	}
}

// Stored prize amounts that are not canonical base-10 integers are rewritten to the integer the module read from
// them (math.NewIntFromString); canonical amounts are not rewritten; every other field stays as stored.
func TestHandler_NormalisesStakingRewardPrizeAmounts(t *testing.T) {
	cases := map[string]string{
		"canonical":    "1000",
		"leading-zero": "0100",
		"hex":          "0x10",
		"octal-prefix": "0o10",
		"binary":       "0b10",
		"underscore":   "1_000",
		"plus":         "+10",
	}
	var rewards []rewardstypes.StakingReward
	for _, id := range []string{"canonical", "leading-zero", "hex", "octal-prefix", "binary", "underscore", "plus"} {
		rewards = append(rewards, stakingReward(id, cases[id]))
	}
	store := newFakeStakingRewards(rewards...)

	runStakingRewardNormalisation(t, store)

	want := map[string]string{
		"canonical":    "1000",
		"leading-zero": "64",
		"hex":          "16",
		"octal-prefix": "8",
		"binary":       "2",
		"underscore":   "1000",
		"plus":         "10",
	}
	for id, amount := range want {
		got := store.byId[id]
		require.Equal(t, amount, got.PrizeAmount, id)

		expected := stakingReward(id, amount)
		require.Equal(t, expected, got, "%s: only the prize amount may change", id)

		_, err := rewardstypes.ParseCanonicalAmount(got.PrizeAmount)
		require.NoError(t, err, id)
	}
	require.NotContains(t, store.written, "canonical", "a canonical amount is not rewritten")
	require.Len(t, store.written, len(want)-1)
}

// A stored value that is not an integer in any form is left as it is.
func TestHandler_LeavesUnreadableStakingRewardPrizeAmount(t *testing.T) {
	store := newFakeStakingRewards(stakingReward("garbage", "12abc"), stakingReward("empty", ""))

	runStakingRewardNormalisation(t, store)

	require.Equal(t, "12abc", store.byId["garbage"].PrizeAmount)
	require.Equal(t, "", store.byId["empty"].PrizeAmount)
	require.Empty(t, store.written)
}

// Without staking rewards the step writes nothing.
func TestHandler_NoStakingRewards(t *testing.T) {
	store := newFakeStakingRewards()

	runStakingRewardNormalisation(t, store)

	require.Empty(t, store.written)
}
