package v830_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bze-alphateam/bze/bzeutils"

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
	_, err := v830.CreateUpgradeHandler(cfg, mm, newFakeAccounts(), &fakeGuardian{}, acceptingRateLimits(t), emptyBank(t), noPools{}, nil, rewards, noTraces{}, nil, emptyIBCStore())(
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

// Stored prize amounts that are not canonical base-10 integers are rewritten to canonical form; canonical amounts
// are not rewritten; every other field stays as stored. The stored form is read with Go integer literal rules, so
// a leading 0 means octal, also after a sign. Zero and negative values are rewritten too: the step only makes the
// form canonical, it does not check the value.
func TestHandler_NormalisesStakingRewardPrizeAmounts(t *testing.T) {
	cases := []struct {
		id, stored, want string
	}{
		{id: "canonical", stored: "1000", want: "1000"},
		{id: "padded", stored: "0100", want: "64"},
		{id: "prefixed-x", stored: "0x10", want: "16"},
		{id: "prefixed-o", stored: "0o10", want: "8"},
		{id: "prefixed-b", stored: "0b10", want: "2"},
		{id: "underscore", stored: "1_000", want: "1000"},
		{id: "plus", stored: "+10", want: "10"},
		{id: "negative-padded", stored: "-010", want: "-8"},
		{id: "double-zero", stored: "00", want: "0"},
	}
	var rewards []rewardstypes.StakingReward
	for i, tc := range cases {
		rewards = append(rewards, distinctStakingReward(i, tc.id, tc.stored))
	}
	store := newFakeStakingRewards(rewards...)

	runStakingRewardNormalisation(t, store)

	for i, tc := range cases {
		got := store.byId[tc.id]
		require.Equal(t, tc.want, got.PrizeAmount, tc.id)

		expected := distinctStakingReward(i, tc.id, tc.want)
		require.Equal(t, expected, got, "%s: only the prize amount may change", tc.id)
		require.Equal(t, expected.DistributedStake, got.DistributedStake, tc.id)
		require.Equal(t, expected.Payouts, got.Payouts, tc.id)

		_, err := bzeutils.ParseCanonicalInt(got.PrizeAmount)
		require.NoError(t, err, tc.id)
	}
	require.NotContains(t, store.written, "canonical", "a canonical amount is not rewritten")
	require.Len(t, store.written, len(cases)-1)

	// the zero and negative results stay unusable as a prize amount
	for _, id := range []string{"negative-padded", "double-zero"} {
		_, err := bzeutils.ParseCanonicalAmount(store.byId[id].PrizeAmount)
		require.Error(t, err, id)
	}
}

// distinctStakingReward is stakingReward with a distributed stake and payout count of its own, so a rewrite that
// mixes up rewards or resets these fields is seen.
func distinctStakingReward(i int, id, prizeAmount string) rewardstypes.StakingReward {
	sr := stakingReward(id, prizeAmount)
	sr.Payouts = uint32(3 + i)
	sr.DistributedStake = fmt.Sprintf("0.%d5", i+1)

	return sr
}

// A stored value that is not an integer in any form, or is beyond the integer range, is left as it is.
func TestHandler_LeavesUnreadableStakingRewardPrizeAmount(t *testing.T) {
	// 2 * 10^77 (78 digits) does not fit the 256-bit integer range; with a leading 0 the value is read as
	// octal, so the padded one needs more of them to be out of range too (2 * 8^90)
	overRange := "2" + strings.Repeat("0", 77)
	overRangePadded := "02" + strings.Repeat("0", 90)
	store := newFakeStakingRewards(
		stakingReward("garbage", "12abc"),
		stakingReward("empty", ""),
		stakingReward("over-range", overRange),
		stakingReward("over-range-padded", overRangePadded),
	)

	runStakingRewardNormalisation(t, store)

	require.Equal(t, "12abc", store.byId["garbage"].PrizeAmount)
	require.Equal(t, "", store.byId["empty"].PrizeAmount)
	require.Equal(t, overRange, store.byId["over-range"].PrizeAmount)
	require.Equal(t, overRangePadded, store.byId["over-range-padded"].PrizeAmount)
	require.Empty(t, store.written)
}

// Without staking rewards the step writes nothing.
func TestHandler_NoStakingRewards(t *testing.T) {
	store := newFakeStakingRewards()

	runStakingRewardNormalisation(t, store)

	require.Empty(t, store.written)
}
