package types

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

const testAddr = "bze1qlxlyq2vhcqmg9hlgltryyg9r3rrxky9x25ntt"

// Denoms may legally contain "/" (ibc/HASH, factory/{creator}/{sub}), so composite keys must
// not collide across different (segment, segment) splits of the same joined string, and a
// denom's iteration prefix must not match keys of a denom that merely extends it.
func TestDenomRewardKeys_SlashedDenoms_NoCollision(t *testing.T) {
	require.NotEqual(t,
		DenomRewardPrizeKey("ubze/x", "uatom"),
		DenomRewardPrizeKey("ubze", "x/uatom"),
	)

	require.NotEqual(t,
		DenomRewardParticipantIndexKey(testAddr, "ubze/x", "uatom"),
		DenomRewardParticipantIndexKey(testAddr, "ubze", "x/uatom"),
	)
}

func TestDenomRewardKeys_SlashedDenoms_NoPrefixBleed(t *testing.T) {
	factoryDenom := "factory/" + testAddr + "/sub"

	// a prize of DR "factory/{addr}/sub" must be invisible to DR "factory"'s prize scan
	full := append(KeyPrefix(DenomRewardPrizeKeyPrefix), DenomRewardPrizeKey(factoryDenom, "ubze")...)
	require.False(t, bytes.HasPrefix(full, DenomRewardPrizePrefix("factory")))

	full = append(KeyPrefix(DenomRewardParticipantKeyPrefix), DenomRewardParticipantKey(factoryDenom, testAddr)...)
	require.False(t, bytes.HasPrefix(full, DenomRewardParticipantPrefix("factory")))

	full = append(KeyPrefix(DenomRewardScheduleKeyPrefix), DenomRewardScheduleKey(factoryDenom, "000000000001")...)
	require.False(t, bytes.HasPrefix(full, DenomRewardSchedulePrefix("factory")))

	full = append(KeyPrefix(DenomRewardParticipantIndexKeyPrefix), DenomRewardParticipantIndexKey(testAddr, factoryDenom, "ubze")...)
	require.False(t, bytes.HasPrefix(full, DenomRewardParticipantIndexPrefix(testAddr, "factory")))
}
