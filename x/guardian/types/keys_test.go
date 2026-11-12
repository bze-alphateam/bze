package types_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// storeKeys lists every key and key prefix of the x/guardian store. Add each new
// one here: two prefixes that overlap would read and delete each other's entries.
var storeKeys = map[string][]byte{
	"ParamsKey":              types.ParamsKey,
	"ClosedChannelKeyPrefix": types.ClosedChannelKeyPrefix,
}

func TestStoreKeys_DistinctAndPrefixFree(t *testing.T) {
	for nameA, a := range storeKeys {
		require.NotEmpty(t, a, nameA)
		for nameB, b := range storeKeys {
			if nameA == nameB {
				continue
			}
			require.False(t, bytes.HasPrefix(b, a), "%s (%q) is a prefix of %s (%q)", nameA, a, nameB, b)
		}
	}
}

// The stored bytes are consensus state: changing a key orphans existing entries.
func TestStoreKeys_Pinned(t *testing.T) {
	require.Equal(t, []byte("p_guardian"), types.ParamsKey)
	require.Equal(t, []byte("closed_channel/"), types.ClosedChannelKeyPrefix)
}
