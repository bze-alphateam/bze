package keeper_test

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// --- EnqueueUnlockParticipants tests ---
// This function now only enqueues the epoch number into the UnlockParticipantsQueue.
// It does NOT process bank sends or remove participants.

func (suite *IntegrationTestSuite) TestServiceUnlock_EnqueueEmpty() {
	// No pending participants - should not add epoch to queue
	suite.k.EnqueueUnlockParticipants(suite.ctx, 100)

	_, found := suite.k.GetUnlockParticipantsQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestServiceUnlock_EnqueueSingle() {
	epochNumber := int64(50)
	addr := sdk.AccAddress("addr1")
	participant := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr.String())),
		Address: addr.String(),
		Amount:  "1000",
		Denom:   "ubze",
	}

	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	suite.k.EnqueueUnlockParticipants(suite.ctx, epochNumber)

	// Verify epoch was added to queue
	queue, found := suite.k.GetUnlockParticipantsQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.UnlockEpochs, 1)
	suite.Require().Equal(uint64(epochNumber), queue.UnlockEpochs[0])

	// Verify participants are still in store (not processed yet)
	participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Len(participants, 1)
}

func (suite *IntegrationTestSuite) TestServiceUnlock_EnqueueMultipleEpochs() {
	addr := sdk.AccAddress("addr1")

	for _, epoch := range []int64{100, 200} {
		suite.k.SetPendingUnlockParticipant(suite.ctx, types.PendingUnlockParticipant{
			Index:   fmt.Sprintf("%d/%s", epoch, fmt.Sprintf("%s/%s", "reward_1", addr.String())),
			Address: addr.String(),
			Amount:  "500",
			Denom:   "ubze",
		})
		suite.k.EnqueueUnlockParticipants(suite.ctx, epoch)
	}

	queue, found := suite.k.GetUnlockParticipantsQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.UnlockEpochs, 2)
	suite.Require().Equal(uint64(100), queue.UnlockEpochs[0])
	suite.Require().Equal(uint64(200), queue.UnlockEpochs[1])
}

// --- ProcessUnlockParticipantsQueue tests ---

func (suite *IntegrationTestSuite) TestProcessQueue_EmptyQueue() {
	// No queue at all - should not panic
	suite.Require().NotPanics(func() {
		suite.k.ProcessUnlockParticipantsQueue(suite.ctx)
	})
}

func (suite *IntegrationTestSuite) TestProcessQueue_SingleEntry() {
	epochNumber := int64(50)
	addr := sdk.AccAddress("addr1")
	participant := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr.String())),
		Address: addr.String(),
		Amount:  "1000",
		Denom:   "ubze",
	}

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(
			gomock.Any(),
			types.ModuleName,
			addr,
			sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000))),
		).
		Return(nil).
		Times(1)

	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	suite.k.EnqueueUnlockParticipants(suite.ctx, epochNumber)

	suite.k.ProcessUnlockParticipantsQueue(suite.ctx)

	// Verify participant was removed
	participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Empty(participants)

	// Verify epoch was removed from queue
	queue, found := suite.k.GetUnlockParticipantsQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Empty(queue.UnlockEpochs)
}

func (suite *IntegrationTestSuite) TestProcessQueue_MultipleEntries() {
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

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin("utoken", math.NewInt(750)))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr3, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1000)))).
		Return(nil).Times(1)

	for _, p := range participants {
		suite.k.SetPendingUnlockParticipant(suite.ctx, p)
	}
	suite.k.EnqueueUnlockParticipants(suite.ctx, epochNumber)

	suite.k.ProcessUnlockParticipantsQueue(suite.ctx)

	remaining := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Empty(remaining)
}

func (suite *IntegrationTestSuite) TestProcessQueue_BankErrorKeepsEpochInQueue() {
	epochNumber := int64(100)
	addr := sdk.AccAddress("addr1")
	participant := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr.String())),
		Address: addr.String(),
		Amount:  "500",
		Denom:   "ubze",
	}

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(fmt.Errorf("insufficient funds")).
		Times(1)

	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	suite.k.EnqueueUnlockParticipants(suite.ctx, epochNumber)

	suite.Require().NotPanics(func() {
		suite.k.ProcessUnlockParticipantsQueue(suite.ctx)
	})

	// Participant should still be in store (bank send failed)
	participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Len(participants, 1)

	// Epoch should remain in queue for retry
	queue, found := suite.k.GetUnlockParticipantsQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.UnlockEpochs, 1)
	suite.Require().Equal(uint64(epochNumber), queue.UnlockEpochs[0])
}

func (suite *IntegrationTestSuite) TestProcessQueue_InvalidAddressKeepsEpochInQueue() {
	epochNumber := int64(125)
	participant := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", "invalid-address")),
		Address: "invalid-address",
		Amount:  "500",
		Denom:   "ubze",
	}

	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	suite.k.EnqueueUnlockParticipants(suite.ctx, epochNumber)

	suite.k.ProcessUnlockParticipantsQueue(suite.ctx)

	// Participant should remain (invalid address = error)
	participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Len(participants, 1)

	// Epoch should remain in queue
	queue, found := suite.k.GetUnlockParticipantsQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.UnlockEpochs, 1)
}

func (suite *IntegrationTestSuite) TestProcessQueue_MixedSuccessAndFailure() {
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

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewCoin("utoken", math.NewInt(1000)))).
		Return(nil).Times(1)

	for _, p := range participants {
		suite.k.SetPendingUnlockParticipant(suite.ctx, p)
	}
	suite.k.EnqueueUnlockParticipants(suite.ctx, epochNumber)

	suite.k.ProcessUnlockParticipantsQueue(suite.ctx)

	// Only the invalid-address participant should remain
	remaining := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Len(remaining, 1)
	suite.Require().Equal("invalid-address", remaining[0].Address)

	// Epoch should stay in queue (had errors)
	queue, found := suite.k.GetUnlockParticipantsQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.UnlockEpochs, 1)
}

func (suite *IntegrationTestSuite) TestProcessQueue_DifferentEpochsOnlyRequestedProcessed() {
	addr1 := sdk.AccAddress("addr1")
	addr2 := sdk.AccAddress("addr2")

	suite.k.SetPendingUnlockParticipant(suite.ctx, types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", 200, fmt.Sprintf("%s/%s", "reward_1", addr1.String())),
		Address: addr1.String(),
		Amount:  "500",
		Denom:   "ubze",
	})
	suite.k.SetPendingUnlockParticipant(suite.ctx, types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", 201, fmt.Sprintf("%s/%s", "reward_2", addr2.String())),
		Address: addr2.String(),
		Amount:  "750",
		Denom:   "ubze",
	})

	// Only enqueue epoch 200
	suite.k.EnqueueUnlockParticipants(suite.ctx, 200)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).Times(1)

	suite.k.ProcessUnlockParticipantsQueue(suite.ctx)

	// Epoch 200 should be processed
	epoch200 := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, 200)
	suite.Require().Empty(epoch200)

	// Epoch 201 should be untouched
	epoch201 := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, 201)
	suite.Require().Len(epoch201, 1)
}

func (suite *IntegrationTestSuite) TestProcessQueue_ZeroAmountKeepsEpochInQueue() {
	epochNumber := int64(250)
	addr := sdk.AccAddress("addr1")
	participant := types.PendingUnlockParticipant{
		Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("%s/%s", "reward_1", addr.String())),
		Address: addr.String(),
		Amount:  "0",
		Denom:   "ubze",
	}

	suite.k.SetPendingUnlockParticipant(suite.ctx, participant)
	suite.k.EnqueueUnlockParticipants(suite.ctx, epochNumber)

	suite.k.ProcessUnlockParticipantsQueue(suite.ctx)

	// Zero amount fails in getAmountToCapture, so participant should remain
	participants := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, epochNumber)
	suite.Require().Len(participants, 1)

	// Epoch should remain in queue
	queue, found := suite.k.GetUnlockParticipantsQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Len(queue.UnlockEpochs, 1)
}

// --- Store tests for new functions ---

func (suite *IntegrationTestSuite) TestStore_UnlockParticipantsQueue_SetAndGet() {
	queue := types.UnlockParticipantsQueue{UnlockEpochs: []uint64{10, 20, 30}}
	suite.k.SetUnlockParticipantsQueue(suite.ctx, queue)

	result, found := suite.k.GetUnlockParticipantsQueue(suite.ctx)
	suite.Require().True(found)
	suite.Require().Equal(queue.UnlockEpochs, result.UnlockEpochs)
}

func (suite *IntegrationTestSuite) TestStore_UnlockParticipantsQueue_NotFound() {
	_, found := suite.k.GetUnlockParticipantsQueue(suite.ctx)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStore_GetBatchEpochPendingUnlockParticipant() {
	epochNumber := int64(100)
	addr1 := sdk.AccAddress("addr1")
	addr2 := sdk.AccAddress("addr2")
	addr3 := sdk.AccAddress("addr3")

	for i, addr := range []sdk.AccAddress{addr1, addr2, addr3} {
		suite.k.SetPendingUnlockParticipant(suite.ctx, types.PendingUnlockParticipant{
			Index:   fmt.Sprintf("%d/%s", epochNumber, fmt.Sprintf("reward_%d/%s", i+1, addr.String())),
			Address: addr.String(),
			Amount:  "100",
			Denom:   "ubze",
		})
	}

	// Limit to 2 - should return only 2
	batch := suite.k.GetBatchEpochPendingUnlockParticipant(suite.ctx, epochNumber, 2)
	suite.Require().Len(batch, 2)

	// Limit higher than entries - should return all 3
	batch = suite.k.GetBatchEpochPendingUnlockParticipant(suite.ctx, epochNumber, 10)
	suite.Require().Len(batch, 3)

	// Empty epoch
	batch = suite.k.GetBatchEpochPendingUnlockParticipant(suite.ctx, 999, 10)
	suite.Require().Empty(batch)
}
