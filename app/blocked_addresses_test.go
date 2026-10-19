package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
)

// The modules removed in v8.3.0 keep their addresses blocked: nobody holds a
// key for them, so funds sent there would be stranded.
func TestBlockedAddresses_RemovedModulesStayBlocked(t *testing.T) {
	blocked := app.BlockedAddresses()
	for _, module := range []string{"nft", "feeibc"} {
		require.True(t, blocked[module], "%s module account must be blocked", module)
	}
}

// The x/dao module account escrows proposal deposits; it never takes plain
// bank sends.
func TestBlockedAddresses_DaoModuleIsBlocked(t *testing.T) {
	require.True(t, app.BlockedAddresses()["dao"])
}
