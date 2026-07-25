package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// encodeStringField encodes a proto3 length-delimited (wire type 2) string field.
// Only used for test values shorter than 128 bytes (single-byte length prefix).
func encodeStringField(fieldNum int, val string) []byte {
	if len(val) >= 128 {
		panic("test helper only supports values < 128 bytes")
	}
	tag := byte(fieldNum<<3 | 2)
	out := []byte{tag, byte(len(val))}
	return append(out, []byte(val)...)
}

// TestStakingRewardParticipant_WireCompat proves that bytes written by the
// pre-boost binary (only fields 1-4, no field 5) decode into the current
// StakingRewardParticipant with an empty boost_snapshots map — i.e. the added
// map field is wire-compatible.
func TestStakingRewardParticipant_WireCompat_OldBytesDecode(t *testing.T) {
	// Hand-craft the exact bytes an old binary would have written for a
	// participant, using only fields 1 (address), 2 (reward_id), 3 (amount),
	// 4 (joined_at). No field 5 (boost_snapshots, tag 0x2a) is present.
	var oldBytes []byte
	oldBytes = append(oldBytes, encodeStringField(1, "bze1participant")...)
	oldBytes = append(oldBytes, encodeStringField(2, "000000000001")...)
	oldBytes = append(oldBytes, encodeStringField(3, "1000")...)
	oldBytes = append(oldBytes, encodeStringField(4, "500")...)

	// Sanity: the old bytes must not carry a boost_snapshots field (tag 0x2a).
	require.NotContains(t, oldBytes, byte(0x2a))

	var p StakingRewardParticipant
	require.NoError(t, p.Unmarshal(oldBytes))

	require.Equal(t, "bze1participant", p.Address)
	require.Equal(t, "000000000001", p.RewardId)
	require.Equal(t, "1000", p.Amount)
	require.Equal(t, "500", p.JoinedAt)
	require.Empty(t, p.BoostSnapshots)
}

// TestStakingRewardParticipant_WireCompat_MapRoundTrips proves the new map field
// itself marshals and unmarshals correctly when populated.
func TestStakingRewardParticipant_WireCompat_MapRoundTrips(t *testing.T) {
	original := StakingRewardParticipant{
		Address:  "bze1participant",
		RewardId: "000000000001",
		Amount:   "1000",
		JoinedAt: "500",
		BoostSnapshots: map[string]*BoostSnapshot{
			"ubze": {Uid: 7, S0: "1.5"},
			"ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2": {Uid: 8, S0: "0"},
		},
	}

	bz, err := original.Marshal()
	require.NoError(t, err)

	var decoded StakingRewardParticipant
	require.NoError(t, decoded.Unmarshal(bz))

	require.Equal(t, original.Address, decoded.Address)
	require.Len(t, decoded.BoostSnapshots, 2)
	require.Equal(t, uint64(7), decoded.BoostSnapshots["ubze"].Uid)
	require.Equal(t, "1.5", decoded.BoostSnapshots["ubze"].S0)
	require.Equal(t, uint64(8), decoded.BoostSnapshots["ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"].Uid)
}
