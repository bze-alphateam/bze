package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestStorePendingUnlockParticipant_SetAndGet() {
	participant := types.PendingUnlockParticipant{
		Index:   "test-index-1",
		Address: sdk.AccAddress("participant").String(),
		Amount:  "1000",
		Denom:   "ubze",
	}

	// Test SetPendingUnlockParticipant
	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)

	// Test GetPendingUnlockParticipant
	retrievedParticipant, found := suite.k.GetPendingUnlockParticipant(suite.ctx, participant.Index)
	suite.Require().True(found)
	suite.Require().Equal(participant.Index, retrievedParticipant.Index)
	suite.Require().Equal(participant.Address, retrievedParticipant.Address)
	suite.Require().Equal(participant.Amount, retrievedParticipant.Amount)
	suite.Require().Equal(participant.Denom, retrievedParticipant.Denom)
}

func (suite *IntegrationTestSuite) TestStorePendingUnlockParticipant_GetNotFound() {
	// Test GetPendingUnlockParticipant with non-existent index
	_, found := suite.k.GetPendingUnlockParticipant(suite.ctx, "nonexistent-index")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStorePendingUnlockParticipant_UpdateExisting() {
	index := "update-test"

	// Set initial participant
	initialParticipant := types.PendingUnlockParticipant{
		Index:   index,
		Address: sdk.AccAddress("participant1").String(),
		Amount:  "500",
		Denom:   "ubze",
	}
	suite.k.SetPendingUnlockParticipant(suite.ctx, initialParticipant)

	// Update participant
	updatedParticipant := types.PendingUnlockParticipant{
		Index:   index,
		Address: sdk.AccAddress("participant2").String(),
		Amount:  "1500",
		Denom:   "uatom",
	}
	suite.k.SetPendingUnlockParticipant(suite.ctx, updatedParticipant)

	// Verify updated values
	retrievedParticipant, found := suite.k.GetPendingUnlockParticipant(suite.ctx, index)
	suite.Require().True(found)
	suite.Require().Equal(updatedParticipant.Address, retrievedParticipant.Address)
	suite.Require().Equal(updatedParticipant.Amount, retrievedParticipant.Amount)
	suite.Require().Equal(updatedParticipant.Denom, retrievedParticipant.Denom)
}

func (suite *IntegrationTestSuite) TestStorePendingUnlockParticipant_Remove() {
	participant := types.PendingUnlockParticipant{
		Index:   "remove-test",
		Address: sdk.AccAddress("participant").String(),
		Amount:  "750",
		Denom:   "ubze",
	}

	// Set participant
	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)

	// Verify it exists
	_, found := suite.k.GetPendingUnlockParticipant(suite.ctx, participant.Index)
	suite.Require().True(found)

	// Remove participant
	suite.k.RemovePendingUnlockParticipant(suite.ctx, participant)

	// Verify it's removed
	_, found = suite.k.GetPendingUnlockParticipant(suite.ctx, participant.Index)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStorePendingUnlockParticipant_GetAllPendingUnlockParticipant() {
	// Create multiple participants
	participants := []types.PendingUnlockParticipant{
		{
			Index:   "participant-1",
			Address: sdk.AccAddress("addr1").String(),
			Amount:  "1000",
			Denom:   "ubze",
		},
		{
			Index:   "participant-2",
			Address: sdk.AccAddress("addr2").String(),
			Amount:  "2000",
			Denom:   "uatom",
		},
		{
			Index:   "participant-3",
			Address: sdk.AccAddress("addr3").String(),
			Amount:  "3000",
			Denom:   "ustake",
		},
	}

	// Set all participants
	for _, participant := range participants {
		suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	}

	// Test GetAllPendingUnlockParticipant
	allParticipants := suite.k.GetAllPendingUnlockParticipant(suite.ctx)
	suite.Require().Len(allParticipants, 3)

	// Verify all participants are present
	indexMap := make(map[string]types.PendingUnlockParticipant)
	for _, participant := range allParticipants {
		indexMap[participant.Index] = participant
	}

	for _, originalParticipant := range participants {
		retrievedParticipant, exists := indexMap[originalParticipant.Index]
		suite.Require().True(exists)
		suite.Require().Equal(originalParticipant.Address, retrievedParticipant.Address)
		suite.Require().Equal(originalParticipant.Amount, retrievedParticipant.Amount)
		suite.Require().Equal(originalParticipant.Denom, retrievedParticipant.Denom)
	}
}

func (suite *IntegrationTestSuite) TestStorePendingUnlockParticipant_GetAllEpochPendingUnlockParticipant() {
	epoch1 := int64(100)
	epoch2 := int64(200)

	// Create participants for different epochs (based on index structure)
	participants := []types.PendingUnlockParticipant{
		{
			Index:   fmt.Sprintf("%d/%s", epoch1, fmt.Sprintf("%s/%s", "reward_1", sdk.AccAddress("addr1").String())),
			Address: sdk.AccAddress("addr1").String(),
			Amount:  "1000",
			Denom:   "ubze",
		},
		{
			Index:   fmt.Sprintf("%d/%s", epoch1, fmt.Sprintf("%s/%s", "reward_2", sdk.AccAddress("addr2").String())),
			Address: sdk.AccAddress("addr2").String(),
			Amount:  "2000",
			Denom:   "ubze",
		},
		{
			Index:   fmt.Sprintf("%d/%s", epoch2, fmt.Sprintf("%s/%s", "reward_3", sdk.AccAddress("addr3").String())),
			Address: sdk.AccAddress("addr3").String(),
			Amount:  "3000",
			Denom:   "ubze",
		},
	}

	// Set all participants
	for _, participant := range participants {
		suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	}

	// Test GetAllEpochPendingUnlockParticipant for epoch1
	// Note: This test assumes the key structure includes epoch information
	// The actual behavior depends on how PendingUnlockParticipantPrefix(epoch) works
	epoch1Participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epoch1)

	// Since we don't know the exact key structure, we'll just verify the method works
	// without asserting specific counts
	suite.Require().NotNil(epoch1Participants)
	suite.Require().Len(epoch1Participants, 2)

	// Test GetAllEpochPendingUnlockParticipant for epoch2
	epoch2Participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epoch2)
	suite.Require().NotNil(epoch2Participants)
	suite.Require().Len(epoch2Participants, 1)
}

func (suite *IntegrationTestSuite) TestStorePendingUnlockParticipant_EmptyStore() {
	// Test GetAllPendingUnlockParticipant with empty store
	allParticipants := suite.k.GetAllPendingUnlockParticipant(suite.ctx)
	suite.Require().Len(allParticipants, 0)

	// Test GetAllEpochPendingUnlockParticipant with empty store
	epochParticipants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, 123)
	suite.Require().Len(epochParticipants, 0)
}

func (suite *IntegrationTestSuite) TestStorePendingUnlockParticipant_EmptyFields() {
	// Test with empty fields
	participant := types.PendingUnlockParticipant{
		Index:   "empty-test",
		Address: "",
		Amount:  "",
		Denom:   "",
	}

	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)

	retrievedParticipant, found := suite.k.GetPendingUnlockParticipant(suite.ctx, participant.Index)
	suite.Require().True(found)
	suite.Require().Equal("", retrievedParticipant.Address)
	suite.Require().Equal("", retrievedParticipant.Amount)
	suite.Require().Equal("", retrievedParticipant.Denom)
}

func (suite *IntegrationTestSuite) TestStorePendingUnlockParticipant_LargeAmounts() {
	// Test with large amount values
	participant := types.PendingUnlockParticipant{
		Index:   "large-amount-test",
		Address: sdk.AccAddress("participant").String(),
		Amount:  "999999999999999999999999999999",
		Denom:   "ubze",
	}

	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)

	retrievedParticipant, found := suite.k.GetPendingUnlockParticipant(suite.ctx, participant.Index)
	suite.Require().True(found)
	suite.Require().Equal(participant.Amount, retrievedParticipant.Amount)
}

func (suite *IntegrationTestSuite) TestStorePendingUnlockParticipant_RemoveNonExistent() {
	// Test removing non-existent participant (should not panic)
	nonExistentParticipant := types.PendingUnlockParticipant{
		Index: "does-not-exist",
	}

	// Should not panic
	suite.k.RemovePendingUnlockParticipant(suite.ctx, nonExistentParticipant)

	// Verify it still doesn't exist
	_, found := suite.k.GetPendingUnlockParticipant(suite.ctx, nonExistentParticipant.Index)
	suite.Require().False(found)
}
