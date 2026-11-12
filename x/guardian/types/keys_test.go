package types_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

// storeKeys lists every key and key prefix of the x/guardian store, with the
// bytes it must keep: a stored key that changes loses its state on upgrade.
// Every new key or prefix is added here.
var storeKeys = []struct {
	name string
	key  []byte
	want string
}{
	{"params", types.ParamsKey, "p_guardian"},
	{"automation paused", types.AutomationPausedKey, "automation_paused"},
}

func TestStoreKeys_Pinned(t *testing.T) {
	for _, k := range storeKeys {
		require.Equal(t, k.want, string(k.key), k.name)
	}
}

// No key is equal to or a prefix of another, so no key or prefix iteration can
// read another one's entries.
func TestStoreKeys_DistinctAndPrefixFree(t *testing.T) {
	for i, a := range storeKeys {
		require.NotEmpty(t, a.key, a.name)
		for j, b := range storeKeys {
			if i == j {
				continue
			}
			require.False(t, bytes.HasPrefix(b.key, a.key), "%s (%q) is a prefix of %s (%q)", a.name, a.key, b.name, b.key)
		}
	}
}
