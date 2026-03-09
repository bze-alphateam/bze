package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/stretchr/testify/require"
)

func (suite *IntegrationTestSuite) TestStoreRaffle_SetAndGetRaffle() {
	// Test data
	raffle := types.Raffle{
		Pot:         "1000utoken",
		Duration:    3600,
		Chances:     100,
		Ratio:       "1:10",
		EndAt:       12345,
		Winners:     5,
		TicketPrice: "10utoken",
		Denom:       "utoken",
		TotalWon:    "500utoken",
	}

	// Test SetRaffle
	suite.k.SetRaffle(suite.ctx, raffle)

	// Test GetRaffle - should find the raffle
	retrievedRaffle, found := suite.k.GetRaffle(suite.ctx, raffle.Denom)
	require.True(suite.T(), found)
	require.Equal(suite.T(), raffle.Pot, retrievedRaffle.Pot)
	require.Equal(suite.T(), raffle.Duration, retrievedRaffle.Duration)
	require.Equal(suite.T(), raffle.Chances, retrievedRaffle.Chances)
	require.Equal(suite.T(), raffle.Ratio, retrievedRaffle.Ratio)
	require.Equal(suite.T(), raffle.EndAt, retrievedRaffle.EndAt)
	require.Equal(suite.T(), raffle.Winners, retrievedRaffle.Winners)
	require.Equal(suite.T(), raffle.TicketPrice, retrievedRaffle.TicketPrice)
	require.Equal(suite.T(), raffle.Denom, retrievedRaffle.Denom)
	require.Equal(suite.T(), raffle.TotalWon, retrievedRaffle.TotalWon)

	// Test GetRaffle with non-existent denom
	_, found = suite.k.GetRaffle(suite.ctx, "nonexistent")
	require.False(suite.T(), found)
}

func (suite *IntegrationTestSuite) TestStoreRaffle_GetAllRaffle() {
	// Create multiple raffles
	raffles := []types.Raffle{
		{
			Pot:         "1000utoken1",
			Duration:    3600,
			Chances:     100,
			Ratio:       "1:10",
			EndAt:       12345,
			Winners:     5,
			TicketPrice: "10utoken1",
			Denom:       "utoken1",
			TotalWon:    "500utoken1",
		},
		{
			Pot:         "2000utoken2",
			Duration:    7200,
			Chances:     200,
			Ratio:       "1:20",
			EndAt:       23456,
			Winners:     10,
			TicketPrice: "20utoken2",
			Denom:       "utoken2",
			TotalWon:    "1000utoken2",
		},
		{
			Pot:         "3000utoken3",
			Duration:    10800,
			Chances:     300,
			Ratio:       "1:30",
			EndAt:       34567,
			Winners:     15,
			TicketPrice: "30utoken3",
			Denom:       "utoken3",
			TotalWon:    "1500utoken3",
		},
	}

	// Set all raffles
	for _, raffle := range raffles {
		suite.k.SetRaffle(suite.ctx, raffle)
	}

	// Test GetAllRaffle
	allRaffles := suite.k.GetAllRaffle(suite.ctx)
	require.Len(suite.T(), allRaffles, 3)

	// Verify all raffles are present
	denomMap := make(map[string]types.Raffle)
	for _, raffle := range allRaffles {
		denomMap[raffle.Denom] = raffle
	}

	for _, originalRaffle := range raffles {
		retrievedRaffle, exists := denomMap[originalRaffle.Denom]
		require.True(suite.T(), exists)
		require.Equal(suite.T(), originalRaffle.Pot, retrievedRaffle.Pot)
		require.Equal(suite.T(), originalRaffle.Duration, retrievedRaffle.Duration)
		require.Equal(suite.T(), originalRaffle.Chances, retrievedRaffle.Chances)
		require.Equal(suite.T(), originalRaffle.Ratio, retrievedRaffle.Ratio)
		require.Equal(suite.T(), originalRaffle.EndAt, retrievedRaffle.EndAt)
		require.Equal(suite.T(), originalRaffle.Winners, retrievedRaffle.Winners)
		require.Equal(suite.T(), originalRaffle.TicketPrice, retrievedRaffle.TicketPrice)
		require.Equal(suite.T(), originalRaffle.TotalWon, retrievedRaffle.TotalWon)
	}
}

func (suite *IntegrationTestSuite) TestStoreRaffle_RemoveRaffle() {
	// Create and set a raffle
	raffle := types.Raffle{
		Pot:         "1000utoken",
		Duration:    3600,
		Chances:     100,
		Ratio:       "1:10",
		EndAt:       12345,
		Winners:     5,
		TicketPrice: "10utoken",
		Denom:       "utoken",
		TotalWon:    "500utoken",
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	// Verify it exists
	_, found := suite.k.GetRaffle(suite.ctx, raffle.Denom)
	require.True(suite.T(), found)

	// Remove the raffle
	suite.k.RemoveRaffle(suite.ctx, raffle.Denom)

	// Verify it's been removed
	_, found = suite.k.GetRaffle(suite.ctx, raffle.Denom)
	require.False(suite.T(), found)

	// Verify GetAllRaffle is empty
	allRaffles := suite.k.GetAllRaffle(suite.ctx)
	require.Len(suite.T(), allRaffles, 0)
}

func (suite *IntegrationTestSuite) TestStoreRaffle_SetAndRemoveRaffleDeleteHook() {
	// Test data
	deleteHook := types.RaffleDeleteHook{
		Denom: "utoken",
		EndAt: 12345,
	}

	// Set raffle delete hook
	suite.k.SetRaffleDeleteHook(suite.ctx, deleteHook)

	// Get raffle delete hooks by prefix
	hooks := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, deleteHook.EndAt)
	require.Len(suite.T(), hooks, 1)
	require.Equal(suite.T(), deleteHook.Denom, hooks[0].Denom)
	require.Equal(suite.T(), deleteHook.EndAt, hooks[0].EndAt)

	// Remove raffle delete hook
	suite.k.RemoveRaffleDeleteHook(suite.ctx, deleteHook)

	// Verify it's been removed
	hooks = suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, deleteHook.EndAt)
	require.Len(suite.T(), hooks, 0)
}

func (suite *IntegrationTestSuite) TestStoreRaffle_GetRaffleDeleteHookByEndAtPrefix() {
	// Create multiple delete hooks with same EndAt
	endAt := uint64(12345)
	hooks := []types.RaffleDeleteHook{
		{Denom: "utoken1", EndAt: endAt},
		{Denom: "utoken2", EndAt: endAt},
		{Denom: "utoken3", EndAt: endAt},
	}

	// Set all hooks
	for _, hook := range hooks {
		suite.k.SetRaffleDeleteHook(suite.ctx, hook)
	}

	// Create a hook with different EndAt
	differentHook := types.RaffleDeleteHook{Denom: "utoken4", EndAt: 99999}
	suite.k.SetRaffleDeleteHook(suite.ctx, differentHook)

	// Test GetRaffleDeleteHookByEndAtPrefix
	retrievedHooks := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, endAt)
	require.Len(suite.T(), retrievedHooks, 3)

	// Verify correct hooks are returned
	denomMap := make(map[string]bool)
	for _, hook := range retrievedHooks {
		denomMap[hook.Denom] = true
		require.Equal(suite.T(), endAt, hook.EndAt)
	}

	require.True(suite.T(), denomMap["utoken1"])
	require.True(suite.T(), denomMap["utoken2"])
	require.True(suite.T(), denomMap["utoken3"])
	require.False(suite.T(), denomMap["utoken4"])
}

func (suite *IntegrationTestSuite) TestStoreRaffle_SetAndGetRaffleWinners() {
	denom := "utoken"
	winners := []types.RaffleWinner{
		{Index: "1", Denom: denom, Amount: "100utoken", Winner: "addr1"},
		{Index: "2", Denom: denom, Amount: "200utoken", Winner: "addr2"},
		{Index: "3", Denom: denom, Amount: "300utoken", Winner: "addr3"},
	}

	// Set all winners
	for _, winner := range winners {
		suite.k.SetRaffleWinner(suite.ctx, winner)
	}

	// Test GetRaffleWinners
	retrievedWinners := suite.k.GetRaffleWinners(suite.ctx, denom)
	require.Len(suite.T(), retrievedWinners, 3)

	// Verify all winners are present
	indexMap := make(map[string]types.RaffleWinner)
	for _, winner := range retrievedWinners {
		indexMap[winner.Index] = winner
	}

	for _, originalWinner := range winners {
		retrievedWinner, exists := indexMap[originalWinner.Index]
		require.True(suite.T(), exists)
		require.Equal(suite.T(), originalWinner.Denom, retrievedWinner.Denom)
		require.Equal(suite.T(), originalWinner.Amount, retrievedWinner.Amount)
		require.Equal(suite.T(), originalWinner.Winner, retrievedWinner.Winner)
	}
}

func (suite *IntegrationTestSuite) TestStoreRaffle_RemoveRaffleWinner() {
	// Create and set winners
	winner1 := types.RaffleWinner{Index: "1", Denom: "utoken", Amount: "100utoken", Winner: "addr1"}
	winner2 := types.RaffleWinner{Index: "2", Denom: "utoken", Amount: "200utoken", Winner: "addr2"}

	suite.k.SetRaffleWinner(suite.ctx, winner1)
	suite.k.SetRaffleWinner(suite.ctx, winner2)

	// Verify both exist
	winners := suite.k.GetRaffleWinners(suite.ctx, "utoken")
	require.Len(suite.T(), winners, 2)

	// Remove one winner
	suite.k.RemoveRaffleWinner(suite.ctx, winner1)

	// Verify only one remains
	winners = suite.k.GetRaffleWinners(suite.ctx, "utoken")
	require.Len(suite.T(), winners, 1)
	require.Equal(suite.T(), winner2.Index, winners[0].Index)
	require.Equal(suite.T(), winner2.Amount, winners[0].Amount)
	require.Equal(suite.T(), winner2.Winner, winners[0].Winner)
}

func (suite *IntegrationTestSuite) TestStoreRaffle_SetAndGetRaffleParticipants() {
	executeAt := int64(12345)
	participants := []types.RaffleParticipant{
		{Index: 1, Denom: "utoken", Participant: "addr1", ExecuteAt: executeAt},
		{Index: 2, Denom: "utoken", Participant: "addr2", ExecuteAt: executeAt},
		{Index: 3, Denom: "utoken", Participant: "addr3", ExecuteAt: executeAt},
	}

	// Set all participants
	for _, participant := range participants {
		suite.k.SetRaffleParticipant(suite.ctx, participant)
	}

	// Test GetAllPrefixedRaffleParticipants
	retrievedParticipants := suite.k.GetAllPrefixedRaffleParticipants(suite.ctx, executeAt)
	require.Len(suite.T(), retrievedParticipants, 3)

	// Verify all participants are present
	indexMap := make(map[uint64]types.RaffleParticipant)
	for _, participant := range retrievedParticipants {
		indexMap[participant.Index] = participant
	}

	for _, originalParticipant := range participants {
		retrievedParticipant, exists := indexMap[originalParticipant.Index]
		require.True(suite.T(), exists)
		require.Equal(suite.T(), originalParticipant.Denom, retrievedParticipant.Denom)
		require.Equal(suite.T(), originalParticipant.Participant, retrievedParticipant.Participant)
		require.Equal(suite.T(), originalParticipant.ExecuteAt, retrievedParticipant.ExecuteAt)
	}
}

func (suite *IntegrationTestSuite) TestStoreRaffle_GetAllRaffleParticipants() {
	// Create participants with different ExecuteAt values
	participants := []types.RaffleParticipant{
		{Index: 1, Denom: "utoken1", Participant: "addr1", ExecuteAt: 100},
		{Index: 2, Denom: "utoken2", Participant: "addr2", ExecuteAt: 200},
		{Index: 3, Denom: "utoken3", Participant: "addr3", ExecuteAt: 300},
	}

	// Set all participants
	for _, participant := range participants {
		suite.k.SetRaffleParticipant(suite.ctx, participant)
	}

	// Test GetAllRaffleParticipants
	allParticipants := suite.k.GetAllRaffleParticipants(suite.ctx)
	require.Len(suite.T(), allParticipants, 3)

	// Verify all participants are present
	indexMap := make(map[uint64]types.RaffleParticipant)
	for _, participant := range allParticipants {
		indexMap[participant.Index] = participant
	}

	for _, originalParticipant := range participants {
		retrievedParticipant, exists := indexMap[originalParticipant.Index]
		require.True(suite.T(), exists)
		require.Equal(suite.T(), originalParticipant.Denom, retrievedParticipant.Denom)
		require.Equal(suite.T(), originalParticipant.Participant, retrievedParticipant.Participant)
		require.Equal(suite.T(), originalParticipant.ExecuteAt, retrievedParticipant.ExecuteAt)
	}
}

func (suite *IntegrationTestSuite) TestStoreRaffle_RemoveRaffleParticipant() {
	// Create and set participants
	participant1 := types.RaffleParticipant{Index: 1, Denom: "utoken", Participant: "addr1", ExecuteAt: 12345}
	participant2 := types.RaffleParticipant{Index: 2, Denom: "utoken", Participant: "addr2", ExecuteAt: 12345}

	suite.k.SetRaffleParticipant(suite.ctx, participant1)
	suite.k.SetRaffleParticipant(suite.ctx, participant2)

	// Verify both exist
	participants := suite.k.GetAllPrefixedRaffleParticipants(suite.ctx, 12345)
	require.Len(suite.T(), participants, 2)

	// Remove one participant
	suite.k.RemoveRaffleParticipant(suite.ctx, participant1)

	// Verify only one remains
	participants = suite.k.GetAllPrefixedRaffleParticipants(suite.ctx, 12345)
	require.Len(suite.T(), participants, 1)
	require.Equal(suite.T(), participant2.Index, participants[0].Index)
	require.Equal(suite.T(), participant2.Denom, participants[0].Denom)
	require.Equal(suite.T(), participant2.Participant, participants[0].Participant)
}

func (suite *IntegrationTestSuite) TestStoreRaffle_ParticipantCounter() {
	// Test initial counter value
	counter := suite.k.GetParticipantCounter(suite.ctx)
	require.Equal(suite.T(), uint64(0), counter)

	// Test SetParticipantCounter
	suite.k.SetParticipantCounter(suite.ctx, 100)
	counter = suite.k.GetParticipantCounter(suite.ctx)
	require.Equal(suite.T(), uint64(100), counter)

	// Test SetRaffleParticipant increments counter
	participant := types.RaffleParticipant{
		Index:       999, // This will be overridden by the auto-increment
		Denom:       "utoken",
		Participant: "addr1",
		ExecuteAt:   12345,
	}

	suite.k.SetRaffleParticipant(suite.ctx, participant)
	counter = suite.k.GetParticipantCounter(suite.ctx)
	require.Equal(suite.T(), uint64(101), counter)

	// Test another participant increments counter again
	participant2 := types.RaffleParticipant{
		Index:       999, // This will be overridden by the auto-increment
		Denom:       "utoken",
		Participant: "addr2",
		ExecuteAt:   12345,
	}

	suite.k.SetRaffleParticipant(suite.ctx, participant2)
	counter = suite.k.GetParticipantCounter(suite.ctx)
	require.Equal(suite.T(), uint64(102), counter)
}

func (suite *IntegrationTestSuite) TestStoreRaffle_CountPrefixedRaffleParticipants() {
	// Empty prefix should return 0
	count := suite.k.CountPrefixedRaffleParticipants(suite.ctx, 500)
	require.Equal(suite.T(), 0, count)

	// Insert entries at prefix 500
	for i := uint64(0); i < 5; i++ {
		suite.k.SetRaffleParticipant(suite.ctx, types.RaffleParticipant{
			Index:       i,
			Denom:       "utoken",
			Participant: "addr1",
			ExecuteAt:   500,
		})
	}

	count = suite.k.CountPrefixedRaffleParticipants(suite.ctx, 500)
	require.Equal(suite.T(), 5, count)

	// Different prefix should still be 0
	count = suite.k.CountPrefixedRaffleParticipants(suite.ctx, 600)
	require.Equal(suite.T(), 0, count)

	// Insert entries at prefix 600
	for i := uint64(10); i < 13; i++ {
		suite.k.SetRaffleParticipant(suite.ctx, types.RaffleParticipant{
			Index:       i,
			Denom:       "utoken",
			Participant: "addr2",
			ExecuteAt:   600,
		})
	}

	// Verify independence between prefixes
	count = suite.k.CountPrefixedRaffleParticipants(suite.ctx, 500)
	require.Equal(suite.T(), 5, count)

	count = suite.k.CountPrefixedRaffleParticipants(suite.ctx, 600)
	require.Equal(suite.T(), 3, count)
}

func (suite *IntegrationTestSuite) TestStoreRaffle_ParticipantCounterBinaryEncoding() {
	// Test edge cases for binary encoding
	testValues := []uint64{0, 1, 255, 256, 65535, 65536, 4294967295, 4294967296}

	for _, value := range testValues {
		suite.k.SetParticipantCounter(suite.ctx, value)
		retrieved := suite.k.GetParticipantCounter(suite.ctx)
		require.Equal(suite.T(), value, retrieved, "Failed for value: %d", value)
	}
}
