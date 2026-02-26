package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

func (suite *IntegrationTestSuite) TestStoreStakingRewardParticipant_SetAndGet() {
	participant := types.StakingRewardParticipant{
		Address:  "bze1abc123",
		RewardId: "reward-1",
		Amount:   math.NewInt(1000),
		JoinedAt: math.LegacyNewDec(500),
	}

	// Test SetStakingRewardParticipant
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	// Test GetStakingRewardParticipant
	retrievedParticipant, found := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1abc123", "reward-1")
	suite.Require().True(found)
	suite.Require().Equal(participant.Address, retrievedParticipant.Address)
	suite.Require().Equal(participant.RewardId, retrievedParticipant.RewardId)
	suite.Require().Equal(participant.Amount, retrievedParticipant.Amount)
	suite.Require().Equal(participant.JoinedAt, retrievedParticipant.JoinedAt)
}

func (suite *IntegrationTestSuite) TestStoreStakingRewardParticipant_GetNonExistent() {
	// Test getting non-existent participant
	_, found := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1nonexistent", "reward-1")
	suite.Require().False(found)

	// Test with existing address but non-existent reward
	participant := types.StakingRewardParticipant{
		Address:  "bze1existing",
		RewardId: "reward-1",
		Amount:   math.NewInt(100),
		JoinedAt: math.LegacyNewDec(50),
	}
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	_, found = suite.k.GetStakingRewardParticipant(suite.ctx, "bze1existing", "reward-2")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreStakingRewardParticipant_SetMultiple() {
	participant1 := types.StakingRewardParticipant{
		Address:  "bze1user1",
		RewardId: "reward-1",
		Amount:   math.NewInt(500),
		JoinedAt: math.LegacyNewDec(100),
	}

	participant2 := types.StakingRewardParticipant{
		Address:  "bze1user2",
		RewardId: "reward-1",
		Amount:   math.NewInt(750),
		JoinedAt: math.LegacyNewDec(150),
	}

	participant3 := types.StakingRewardParticipant{
		Address:  "bze1user1",
		RewardId: "reward-2",
		Amount:   math.NewInt(1000),
		JoinedAt: math.LegacyNewDec(200),
	}

	// Set multiple participants
	suite.k.SetStakingRewardParticipant(suite.ctx, participant1)
	suite.k.SetStakingRewardParticipant(suite.ctx, participant2)
	suite.k.SetStakingRewardParticipant(suite.ctx, participant3)

	// Verify each can be retrieved independently
	retrievedParticipant1, found1 := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1user1", "reward-1")
	suite.Require().True(found1)
	suite.Require().Equal(participant1.Address, retrievedParticipant1.Address)
	suite.Require().Equal(participant1.RewardId, retrievedParticipant1.RewardId)
	suite.Require().Equal(participant1.Amount, retrievedParticipant1.Amount)

	retrievedParticipant2, found2 := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1user2", "reward-1")
	suite.Require().True(found2)
	suite.Require().Equal(participant2.Address, retrievedParticipant2.Address)
	suite.Require().Equal(participant2.Amount, retrievedParticipant2.Amount)

	retrievedParticipant3, found3 := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1user1", "reward-2")
	suite.Require().True(found3)
	suite.Require().Equal(participant3.RewardId, retrievedParticipant3.RewardId)
	suite.Require().Equal(participant3.Amount, retrievedParticipant3.Amount)
}

func (suite *IntegrationTestSuite) TestStoreStakingRewardParticipant_UpdateExisting() {
	originalParticipant := types.StakingRewardParticipant{
		Address:  "bze1update",
		RewardId: "reward-update",
		Amount:   math.NewInt(300),
		JoinedAt: math.LegacyNewDec(75),
	}

	updatedParticipant := types.StakingRewardParticipant{
		Address:  "bze1update",
		RewardId: "reward-update",
		Amount:   math.NewInt(800),
		JoinedAt: math.LegacyNewDec(125),
	}

	// Set original participant
	suite.k.SetStakingRewardParticipant(suite.ctx, originalParticipant)

	// Update the participant
	suite.k.SetStakingRewardParticipant(suite.ctx, updatedParticipant)

	// Verify the participant was updated
	retrievedParticipant, found := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1update", "reward-update")
	suite.Require().True(found)
	suite.Require().Equal(updatedParticipant.Amount, retrievedParticipant.Amount)
	suite.Require().Equal(updatedParticipant.JoinedAt, retrievedParticipant.JoinedAt)
}

func (suite *IntegrationTestSuite) TestStoreStakingRewardParticipant_Remove() {
	participant := types.StakingRewardParticipant{
		Address:  "bze1toremove",
		RewardId: "reward-remove",
		Amount:   math.NewInt(600),
		JoinedAt: math.LegacyNewDec(90),
	}

	// Set the participant
	suite.k.SetStakingRewardParticipant(suite.ctx, participant)

	// Verify it exists
	_, found := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1toremove", "reward-remove")
	suite.Require().True(found)

	// Remove the participant
	suite.k.RemoveStakingRewardParticipant(suite.ctx, "bze1toremove", "reward-remove")

	// Verify it no longer exists
	_, found = suite.k.GetStakingRewardParticipant(suite.ctx, "bze1toremove", "reward-remove")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreStakingRewardParticipant_RemoveNonExistent() {
	// Removing non-existent participant should not cause issues
	suite.Require().NotPanics(func() {
		suite.k.RemoveStakingRewardParticipant(suite.ctx, "bze1nonexistent", "reward-1")
	})
}

func (suite *IntegrationTestSuite) TestStoreStakingRewardParticipant_GetAllEmpty() {
	// Test GetAllStakingRewardParticipant when no participants exist
	allParticipants := suite.k.GetAllStakingRewardParticipant(suite.ctx)
	suite.Require().Empty(allParticipants)
}

func (suite *IntegrationTestSuite) TestStoreStakingRewardParticipant_GetAllMultiple() {
	participants := []types.StakingRewardParticipant{
		{
			Address:  "bze1participant1",
			RewardId: "reward-1",
			Amount:   math.NewInt(100),
			JoinedAt: math.LegacyNewDec(10),
		},
		{
			Address:  "bze1participant2",
			RewardId: "reward-1",
			Amount:   math.NewInt(200),
			JoinedAt: math.LegacyNewDec(20),
		},
		{
			Address:  "bze1participant1",
			RewardId: "reward-2",
			Amount:   math.NewInt(300),
			JoinedAt: math.LegacyNewDec(30),
		},
		{
			Address:  "bze1participant3",
			RewardId: "reward-2",
			Amount:   math.NewInt(400),
			JoinedAt: math.LegacyNewDec(40),
		},
	}

	// Set all participants
	for _, participant := range participants {
		suite.k.SetStakingRewardParticipant(suite.ctx, participant)
	}

	// Get all participants
	allParticipants := suite.k.GetAllStakingRewardParticipant(suite.ctx)
	suite.Require().Len(allParticipants, 4)

	// Verify all participants are present (order might vary)
	participantKeys := make(map[string]bool)
	for _, participant := range allParticipants {
		key := participant.Address + "-" + participant.RewardId
		participantKeys[key] = true
	}

	suite.Require().True(participantKeys["bze1participant1-reward-1"])
	suite.Require().True(participantKeys["bze1participant2-reward-1"])
	suite.Require().True(participantKeys["bze1participant1-reward-2"])
	suite.Require().True(participantKeys["bze1participant3-reward-2"])
}

func (suite *IntegrationTestSuite) TestStoreStakingRewardParticipant_CompositeKey() {
	// Test that the composite key (address + rewardId) works correctly
	participant1 := types.StakingRewardParticipant{
		Address:  "bze1same",
		RewardId: "reward-A",
		Amount:   math.NewInt(100),
		JoinedAt: math.LegacyNewDec(10),
	}

	participant2 := types.StakingRewardParticipant{
		Address:  "bze1same",
		RewardId: "reward-B",
		Amount:   math.NewInt(200),
		JoinedAt: math.LegacyNewDec(20),
	}

	participant3 := types.StakingRewardParticipant{
		Address:  "bze1different",
		RewardId: "reward-A",
		Amount:   math.NewInt(300),
		JoinedAt: math.LegacyNewDec(30),
	}

	// Set all participants
	suite.k.SetStakingRewardParticipant(suite.ctx, participant1)
	suite.k.SetStakingRewardParticipant(suite.ctx, participant2)
	suite.k.SetStakingRewardParticipant(suite.ctx, participant3)

	// Verify each can be retrieved with correct composite key
	retrieved1, found1 := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1same", "reward-A")
	suite.Require().True(found1)
	suite.Require().Equal(participant1.Amount, retrieved1.Amount)

	retrieved2, found2 := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1same", "reward-B")
	suite.Require().True(found2)
	suite.Require().Equal(participant2.Amount, retrieved2.Amount)

	retrieved3, found3 := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1different", "reward-A")
	suite.Require().True(found3)
	suite.Require().Equal(participant3.Amount, retrieved3.Amount)

	// Verify wrong combinations don't exist
	_, found4 := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1different", "reward-B")
	suite.Require().False(found4)
}

func (suite *IntegrationTestSuite) TestStoreStakingRewardParticipant_RemoveIndependence() {
	// Test that removing one participant doesn't affect others
	participant1 := types.StakingRewardParticipant{
		Address:  "bze1user1",
		RewardId: "reward-1",
		Amount:   math.NewInt(100),
		JoinedAt: math.LegacyNewDec(10),
	}

	participant2 := types.StakingRewardParticipant{
		Address:  "bze1user2",
		RewardId: "reward-1",
		Amount:   math.NewInt(200),
		JoinedAt: math.LegacyNewDec(20),
	}

	// Set both participants
	suite.k.SetStakingRewardParticipant(suite.ctx, participant1)
	suite.k.SetStakingRewardParticipant(suite.ctx, participant2)

	// Remove first participant
	suite.k.RemoveStakingRewardParticipant(suite.ctx, "bze1user1", "reward-1")

	// Verify first is removed
	_, found1 := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1user1", "reward-1")
	suite.Require().False(found1)

	// Verify second still exists
	retrieved2, found2 := suite.k.GetStakingRewardParticipant(suite.ctx, "bze1user2", "reward-1")
	suite.Require().True(found2)
	suite.Require().Equal(participant2.Amount, retrieved2.Amount)
}
