package keeper_test

import (
	"sort"

	"github.com/bze-alphateam/bze/testutil/sample"
)

// TestSrParticipantIndex_SetHasRemove: set/has/remove round-trip with a real
// bech32 address, and set idempotency (top-up semantics).
func (suite *IntegrationTestSuite) TestSrParticipantIndex_SetHasRemove() {
	addr := sample.AccAddress()
	suite.Require().False(suite.k.HasStakingRewardParticipantIndexEntry(suite.ctx, "000000000001", addr))

	suite.k.SetStakingRewardParticipantIndexEntry(suite.ctx, "000000000001", addr)
	suite.Require().True(suite.k.HasStakingRewardParticipantIndexEntry(suite.ctx, "000000000001", addr))
	//the address survives the key round-trip intact
	suite.Require().Equal([]string{addr}, suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 100))

	//setting again is a no-op overwrite, not a duplicate
	suite.k.SetStakingRewardParticipantIndexEntry(suite.ctx, "000000000001", addr)
	suite.Require().Equal([]string{addr}, suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 100))

	suite.k.RemoveStakingRewardParticipantIndexEntry(suite.ctx, "000000000001", addr)
	suite.Require().False(suite.k.HasStakingRewardParticipantIndexEntry(suite.ctx, "000000000001", addr))
	suite.Require().Empty(suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 100))
}

// TestSrParticipantIndex_IterateCursorAndLimit: the per-reward iteration is
// bounded by limit and resumes strictly after the cursor address, in key
// order — partial sweeps cover every entry exactly once.
func (suite *IntegrationTestSuite) TestSrParticipantIndex_IterateCursorAndLimit() {
	addrs := make([]string, 5)
	for i := range addrs {
		addrs[i] = sample.AccAddress()
	}
	sort.Strings(addrs)
	for _, addr := range addrs {
		suite.k.SetStakingRewardParticipantIndexEntry(suite.ctx, "000000000001", addr)
	}

	//full iteration in key order
	suite.Require().Equal(addrs, suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 100))
	//limit bounds the batch
	suite.Require().Equal(addrs[:2], suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 2))
	//resuming after a cursor continues with the next address, no skip, no repeat
	suite.Require().Equal(addrs[2:4], suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", addrs[1], 2))
	suite.Require().Equal(addrs[4:], suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", addrs[3], 100))
	//cursor at the last entry → nothing left
	suite.Require().Empty(suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", addrs[4], 100))
	//limit 0 → nothing
	suite.Require().Empty(suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 0))
}

// TestSrParticipantIndex_RewardIsolation: entries of one reward are invisible
// to another reward's iteration and unaffected by its removals.
func (suite *IntegrationTestSuite) TestSrParticipantIndex_RewardIsolation() {
	shared := sample.AccAddress()
	only1 := sample.AccAddress()

	suite.k.SetStakingRewardParticipantIndexEntry(suite.ctx, "000000000001", shared)
	suite.k.SetStakingRewardParticipantIndexEntry(suite.ctx, "000000000001", only1)
	suite.k.SetStakingRewardParticipantIndexEntry(suite.ctx, "000000000002", shared)

	expected1 := []string{shared, only1}
	sort.Strings(expected1)
	suite.Require().Equal(expected1, suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 100))
	suite.Require().Equal([]string{shared}, suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000002", "", 100))

	//removing from one reward leaves the other reward's entry for the same address
	suite.k.RemoveStakingRewardParticipantIndexEntry(suite.ctx, "000000000001", shared)
	suite.Require().Equal([]string{only1}, suite.k.GetStakingRewardParticipantIndexAddresses(suite.ctx, "000000000001", "", 100))
	suite.Require().True(suite.k.HasStakingRewardParticipantIndexEntry(suite.ctx, "000000000002", shared))
}
