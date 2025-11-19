package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestServiceUnlock_UnlockAllPendingUnlockParticipantsByEpochEmpty() {
	// Test with no pending unlock participants
	suite.Require().NotPanics(func() {
		suite.k.UnlockAllPendingUnlockParticipantsByEpoch(suite.ctx, 100)
	})
}

func (suite *IntegrationTestSuite) TestServiceUnlock_UnlockAllPendingUnlockParticipantsByEpochSingle() {
	epochNumber := int64(50)
	addr := sdk.AccAddress("addr1")
	participant := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr.String())),
		Address: addr.String(),
		Amount:  "1000",
		Denom:   "ubze",
	}

	// Set up mock expectations
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000))),
		).
		Return(nil).
		Times(1)

	// Set pending unlock participant
	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)

	// Execute unlock
	suite.k.UnlockAllPendingUnlockParticipantsByEpoch(suite.ctx, epochNumber)

	// Verify participant was removed
	participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Empty(participants)
}

func (suite *IntegrationTestSuite) TestServiceUnlock_UnlockAllPendingUnlockParticipantsByEpochMultiple() {
	epochNumber := int64(75)
	addr1 := sdk.AccAddress("addr1")
	addr2 := sdk.AccAddress("addr2")
	addr3 := sdk.AccAddress("addr3")

	participants := []types.PendingUnlockParticipant{
		{
			Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr1.String())),
			Address: addr1.String(),
			Amount:  "500",
			Denom:   "ubze",
		},
		{
			Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_2", addr2.String())),
			Address: addr2.String(),
			Amount:  "750",
			Denom:   "utoken",
		},
		{
			Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_3", addr3.String())),
			Address: addr3.String(),
			Amount:  "1000",
			Denom:   "ubze",
		},
	}

	// Set up mock expectations for each participant
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr1,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500))),
		).
		Return(nil).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr2,
			sdk.NewCoins(sdk.NewCoin("utoken", math.NewInt(750))),
		).
		Return(nil).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr3,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000))),
		).
		Return(nil).
		Times(1)

	// Set all pending unlock participants
	for _, participant := range participants {
		suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	}

	// Execute unlock
	suite.k.UnlockAllPendingUnlockParticipantsByEpoch(suite.ctx, epochNumber)

	// Verify all participants were removed
	remainingParticipants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Empty(remainingParticipants)
}

func (suite *IntegrationTestSuite) TestServiceUnlock_UnlockAllPendingUnlockParticipantsByEpochBankError() {
	epochNumber := int64(100)
	addr := sdk.AccAddress("addr1")
	participant := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr.String())),
		Address: addr.String(),
		Amount:  "500",
		Denom:   "ubze",
	}

	// Set up mock to return error
	bankError := fmt.Errorf("insufficient funds")
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500))),
		).
		Return(bankError).
		Times(1)

	// Set pending unlock participant
	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)

	// Execute unlock (should not panic despite error)
	suite.Require().NotPanics(func() {
		suite.k.UnlockAllPendingUnlockParticipantsByEpoch(suite.ctx, epochNumber)
	})

	// Verify participant was still removed (error is logged but process continues)
	participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	//should not work since bank sending returned an error
	suite.Require().NotEmpty(participants)
}

func (suite *IntegrationTestSuite) TestServiceUnlock_UnlockAllPendingUnlockParticipantsByEpochInvalidAddress() {
	epochNumber := int64(125)
	participant := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", "invalid-address")),
		Address: "invalid-address",
		Amount:  "500",
		Denom:   "ubze",
	}

	// No bank expectation since address parsing should fail

	// Set pending unlock participant
	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)

	suite.k.UnlockAllPendingUnlockParticipantsByEpoch(suite.ctx, epochNumber)

	// Verify participant was still removed (error is logged but process continues)
	participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	//should not be empty since the participant has an invalid address. should be ignored
	suite.Require().NotEmpty(participants)
}

func (suite *IntegrationTestSuite) TestServiceUnlock_UnlockAllPendingUnlockParticipantsByEpochMixedSuccess() {
	epochNumber := int64(150)
	addr1 := sdk.AccAddress("addr1")
	addr2 := sdk.AccAddress("addr2")

	participants := []types.PendingUnlockParticipant{
		{
			Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr1.String())),
			Address: addr1.String(),
			Amount:  "500",
			Denom:   "ubze",
		},
		{
			Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_2", "invalid-address")),
			Address: "invalid-address",
			Amount:  "750",
			Denom:   "ubze",
		},
		{
			Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_3", addr2.String())),
			Address: addr2.String(),
			Amount:  "1000",
			Denom:   "utoken",
		},
	}

	// Set up mock expectations only for valid addresses
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr1,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500))),
		).
		Return(nil).
		Times(1)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr2,
			sdk.NewCoins(sdk.NewCoin("utoken", math.NewInt(1000))),
		).
		Return(nil).
		Times(1)

	// Set all pending unlock participants
	for _, participant := range participants {
		suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	}

	// Execute unlock (should not panic despite mixed results)
	suite.Require().NotPanics(func() {
		suite.k.UnlockAllPendingUnlockParticipantsByEpoch(suite.ctx, epochNumber)
	})

	// Verify all participants were removed
	remainingParticipants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Len(remainingParticipants, 1)
}

func (suite *IntegrationTestSuite) TestServiceUnlock_UnlockAllPendingUnlockParticipantsByEpochDifferentEpochs() {
	// Set participants for different epochs
	addr1 := sdk.AccAddress("addr1")
	addr2 := sdk.AccAddress("addr2")

	participant1 := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", 200, fmt.Sprintf("%s/%s", "reward_1", addr1.String())),
		Address: addr1.String(),
		Amount:  "500",
		Denom:   "ubze",
	}

	participant2 := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", 201, fmt.Sprintf("%s/%s", "reward_2", addr2.String())),
		Address: addr2.String(),
		Amount:  "750",
		Denom:   "ubze",
	}

	// Set up mock expectation only for epoch 200
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			addr1,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500))),
		).
		Return(nil).
		Times(1)

	// Set participants in different epochs
	suite.k.SetPendingUnlockParticipant(suite.ctx, participant1)
	suite.k.SetPendingUnlockParticipant(suite.ctx, participant2)

	// Execute unlock for epoch 200 only
	suite.k.UnlockAllPendingUnlockParticipantsByEpoch(suite.ctx, 200)

	// Verify only epoch 200 participant was removed
	epoch200Participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, 200)
	suite.Require().Empty(epoch200Participants)

	epoch201Participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, 201)
	suite.Require().Len(epoch201Participants, 1)
	suite.Require().Equal(addr2.String(), epoch201Participants[0].Address)
}

func (suite *IntegrationTestSuite) TestServiceUnlock_UnlockAllPendingUnlockParticipantsByEpochZeroAmount() {
	epochNumber := int64(250)
	addr := sdk.AccAddress("addr1")
	participant := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr.String())),
		Address: addr.String(),
		Amount:  "0",
		Denom:   "ubze",
	}

	// Set pending unlock participant
	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)

	// Execute unlock
	suite.k.UnlockAllPendingUnlockParticipantsByEpoch(suite.ctx, epochNumber)

	// Verify participant was removed
	participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	//amount is 0 and can not be sent, so the participant should remain untouched
	suite.Require().Len(participants, 1)
}
