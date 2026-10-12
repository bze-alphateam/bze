package chain

import (
	"context"
	"sync"
	"testing"

	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// sdkConfigOnce guards the process-wide SDK config (bech32 prefixes), so suites
// may run with t.Parallel().
var sdkConfigOnce sync.Once

// Network is a running bze e2e network.
type Network struct {
	Chain  *cosmos.CosmosChain
	Docker *client.Client
}

// Start starts a bze network for the test and stops it (removing containers,
// volumes and the Docker network) when the test ends. Every suite starts its
// own network.
func Start(t *testing.T, ctx context.Context, opts ...Option) Network {
	t.Helper()

	sdkConfigOnce.Do(func() { cosmos.SetSDKConfig(Bech32Prefix) })

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{BzeChainSpec(opts...)})
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	bze := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().AddChain(bze)
	dockerClient, network := interchaintest.DockerSetup(t)
	require.NoError(t, ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           dockerClient,
		NetworkID:        network,
		SkipPathCreation: true,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	return Network{Chain: bze, Docker: dockerClient}
}
