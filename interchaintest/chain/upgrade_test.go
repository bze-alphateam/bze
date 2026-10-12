package chain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/interchaintest/chain"
)

func TestUpgradeFromEnvironment(t *testing.T) {
	t.Setenv(chain.UpgradeFromEnv, "")
	t.Setenv(chain.UpgradeToEnv, "")
	t.Setenv(chain.UpgradeNameEnv, "")
	require.Equal(t, chain.Upgrade{From: chain.DefaultUpgradeFrom, To: chain.DefaultUpgradeTo, Name: chain.DefaultUpgradeName},
		chain.UpgradeFromEnvironment())

	t.Setenv(chain.UpgradeFromEnv, "v8.1.1")
	t.Setenv(chain.UpgradeNameEnv, "v8.2.0")
	require.Equal(t, chain.Upgrade{From: "v8.1.1", To: "local", Name: "v8.2.0"}, chain.UpgradeFromEnvironment())
}

// minimalGenesis holds the objects the overrides write into.
const minimalGenesis = `{"app_state":{"epochs":{},"gov":{"params":{}},"rewards":{"params":{}},"staking":{"params":{}},"mint":{"params":{}}}}`

// genesisAfter applies the spec's genesis overrides to a minimal genesis.
func genesisAfter(t *testing.T, opts ...chain.Option) string {
	t.Helper()
	cfg := chain.BzeChainSpec(opts...).ChainConfig
	out, err := cfg.ModifyGenesis(cfg, []byte(minimalGenesis))
	require.NoError(t, err)
	return string(out)
}

func TestWithoutGenesis(t *testing.T) {
	require.Contains(t, genesisAfter(t), `"denomRewardLock":1`)

	without := genesisAfter(t, chain.WithoutGenesis(chain.DenomRewardLockKey))
	require.NotContains(t, without, "denomRewardLock")
	require.Contains(t, without, `"voting_period":"20s"`, "the other overrides stay")
}
