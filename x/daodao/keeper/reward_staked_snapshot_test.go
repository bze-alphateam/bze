package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// TestRewardStaked_SnapshotAll_WritesAllParticipants: SnapshotAll iterates
// every participant of the reward program and writes (snapshot_id, addr) →
// power rows plus a SnapshotTotal row. Zero-power entries are dropped.
//
// We exercise this directly via the backend rather than MsgCreateProposal
// because MsgCreateProposal rejects creation on a REWARD_STAKED DAO until
// Epic 5 lands MsgUpdateVotingBackend. Once that exists, the path here
// becomes the snapshot half of MsgCreateProposal end-to-end.
func (suite *IntegrationTestSuite) TestRewardStaked_SnapshotAll_WritesAllParticipants() {
	rewardID := "00000000-0000-0000-0000-000000000ab1"
	dao := suite.setupRewardStakedDao(rewardID)

	addr1 := suite.mustAcc(freshAddr())
	addr2 := suite.mustAcc(freshAddr())
	addr3 := suite.mustAcc(freshAddr())

	participants := []rewardstypes.StakingRewardParticipant{
		{Address: addr1.String(), RewardId: rewardID, Amount: "100"},
		{Address: addr2.String(), RewardId: rewardID, Amount: "250"},
		// addr3 has Amount = "0" — backend should NOT write a row for them.
		{Address: addr3.String(), RewardId: rewardID, Amount: "0"},
	}

	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), rewardID).
		Return(rewardstypes.StakingReward{
			RewardId:     rewardID,
			StakedAmount: "350", // = sum of non-zero amounts
		}, true).
		Times(1)

	// The mock plays back the participant list one entry at a time.
	suite.rewards.EXPECT().
		IterateStakingRewardParticipantsByReward(gomock.Any(), rewardID, gomock.Any()).
		DoAndReturn(func(_ sdk.Context, _ string, cb func(rewardstypes.StakingRewardParticipant) bool) {
			for _, p := range participants {
				if cb(p) {
					return
				}
			}
		}).
		Times(1)

	// Drive a snapshot via the public CreateSnapshot helper (the same path
	// MsgCreateProposal will call once Epic 5 lets REWARD_STAKED DAOs exist).
	snapshotID, err := suite.k.CreateSnapshot(suite.ctx, dao)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1), snapshotID)

	suite.Require().Equal(uint64(100), suite.k.SnapshotPower(suite.ctx, dao.Id, snapshotID, addr1))
	suite.Require().Equal(uint64(250), suite.k.SnapshotPower(suite.ctx, dao.Id, snapshotID, addr2))
	// addr3 was zero-power → no row written → snapshot read returns 0.
	suite.Require().Equal(uint64(0), suite.k.SnapshotPower(suite.ctx, dao.Id, snapshotID, addr3))
	suite.Require().Equal(uint64(350), suite.k.SnapshotTotal(suite.ctx, dao.Id, snapshotID))
}

// TestRewardStaked_SnapshotAll_MissingReward: if the reward program no
// longer exists at snapshot time, SnapshotAll surfaces a hard error rather
// than silently writing a zero-power snapshot.
func (suite *IntegrationTestSuite) TestRewardStaked_SnapshotAll_MissingReward() {
	rewardID := "00000000-0000-0000-0000-000000000ab2"
	dao := suite.setupRewardStakedDao(rewardID)

	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), rewardID).
		Return(rewardstypes.StakingReward{}, false).
		Times(1)

	_, err := suite.k.CreateSnapshot(suite.ctx, dao)
	suite.Require().Error(err)
}

// TestRewardStaked_SnapshotAll_InvalidAmountFails: a participant whose
// Amount can't be parsed as a uint64 errors out — preventing a corrupted
// snapshot from going to disk.
func (suite *IntegrationTestSuite) TestRewardStaked_SnapshotAll_InvalidAmountFails() {
	rewardID := "00000000-0000-0000-0000-000000000ab3"
	dao := suite.setupRewardStakedDao(rewardID)

	bad := suite.mustAcc(freshAddr())
	suite.rewards.EXPECT().
		GetStakingReward(gomock.Any(), rewardID).
		Return(rewardstypes.StakingReward{
			RewardId:     rewardID,
			StakedAmount: "100",
		}, true).
		Times(1)
	suite.rewards.EXPECT().
		IterateStakingRewardParticipantsByReward(gomock.Any(), rewardID, gomock.Any()).
		DoAndReturn(func(_ sdk.Context, _ string, cb func(rewardstypes.StakingRewardParticipant) bool) {
			cb(rewardstypes.StakingRewardParticipant{
				Address: bad.String(), RewardId: rewardID, Amount: "not-a-number",
			})
		}).
		Times(1)

	_, err := suite.k.CreateSnapshot(suite.ctx, dao)
	suite.Require().Error(err)
}
