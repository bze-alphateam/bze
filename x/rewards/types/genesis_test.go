package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validBoostGenesis() *GenesisState {
	gs := DefaultGenesis()
	gs.BoostList = []Boost{
		{Id: "000000000001", RewardId: "000000000001", Denom: "ubze", DailyAmount: "1000", Duration: 5, Payouts: 1, DistributedStake: "0.2", Creator: "bze1creator"},
		{Id: "000000000002", RewardId: "000000000001", Denom: "uvdl", DailyAmount: "300", Duration: 3, Payouts: 3, DistributedStake: "0.5", Creator: "bze1creator"},
	}
	gs.BoostParticipantList = []BoostParticipant{
		{Address: "bze1participant", RewardId: "000000000001", BoostId: "000000000001", JoinedAt: "0.1"},
	}
	gs.BoostCounter = 2

	return gs
}

func TestGenesisState_Validate_Default(t *testing.T) {
	require.NoError(t, DefaultGenesis().Validate())
}

func TestGenesisState_Validate_ValidBoosts(t *testing.T) {
	require.NoError(t, validBoostGenesis().Validate())
}

func TestGenesisState_Validate_DuplicateBoostId(t *testing.T) {
	gs := validBoostGenesis()
	gs.BoostList = append(gs.BoostList, Boost{Id: "000000000001", RewardId: "000000000002"})

	err := gs.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate boost id")
}

func TestGenesisState_Validate_InvalidBoostId(t *testing.T) {
	gs := validBoostGenesis()
	gs.BoostList[0].Id = "not-a-number"

	err := gs.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid boost id")
}

func TestGenesisState_Validate_OrphanParticipant(t *testing.T) {
	// unknown boost id
	gs := validBoostGenesis()
	gs.BoostParticipantList[0].BoostId = "000000000009"

	err := gs.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "references missing boost")

	// known boost id but on a different reward
	gs = validBoostGenesis()
	gs.BoostParticipantList[0].RewardId = "000000000002"

	err = gs.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "references missing boost")
}

func TestGenesisState_Validate_CounterBehindMaxId(t *testing.T) {
	gs := validBoostGenesis()
	gs.BoostCounter = 1

	err := gs.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "behind the highest boost id")
}
