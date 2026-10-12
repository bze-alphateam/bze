package chain

import (
	"context"
	"testing"

	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/relayer"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	// SimdName is interchaintest's built-in ibc-go simapp chain.
	SimdName = "ibc-go-simd"
	// SimdVersion is the ibc-go-simd image tag. It follows bze's ibc-go major
	// line (v8, SDK 0.50); BZE-154 bumps it to the ibc-go v10 line together
	// with the chain.
	SimdVersion = "v8.5.1"
	// SimdChainID is the chain id of the counterparty chain.
	SimdChainID = "simd-e2e-1"
	// SimdDenom is the counterparty's staking and fee denom.
	SimdDenom = "stake"
	// SimdBech32Prefix is the counterparty's account address prefix.
	SimdBech32Prefix = "cosmos"

	// HermesImage and HermesVersion pin the relayer image (interchaintest
	// v10's default Hermes image).
	HermesImage   = "ghcr.io/informalsystems/hermes"
	HermesVersion = "1.13.1"
	// hermesUIDGID is the non-root user of the Hermes image.
	hermesUIDGID = "1000:1000"

	// IbcPath is the relayer path between bze and simd.
	IbcPath = "bze-simd"
	// TransferPort is the ICS-20 port on both chains.
	TransferPort = "transfer"
)

// IbcNetwork is a running bze network, an ibc-go simd counterparty and a
// Hermes relayer with an open ICS-20 channel between them.
type IbcNetwork struct {
	Bze, Simd *cosmos.CosmosChain
	Relayer   ibc.Relayer
	Reporter  ibc.RelayerExecReporter

	// BzeChannel is the transfer channel on bze, SimdChannel its counterparty
	// on simd.
	BzeChannel, SimdChannel string
}

// SimdChainSpec returns the interchaintest spec of the counterparty: one
// validator, no full nodes, free txs (0 gas price).
func SimdChainSpec() *interchaintest.ChainSpec {
	validators, fullNodes := 1, 0
	return &interchaintest.ChainSpec{
		Name:      SimdName,
		ChainName: "simd",
		Version:   SimdVersion,
		ChainConfig: ibc.ChainConfig{
			ChainID:      SimdChainID,
			Bech32Prefix: SimdBech32Prefix,
			Denom:        SimdDenom,
			GasPrices:    "0.0" + SimdDenom,
		},
		NumValidators: &validators,
		NumFullNodes:  &fullNodes,
	}
}

// StartIbc starts a bze network and a simd counterparty, connects them with a
// Hermes relayer (client, connection and an ics20-1 channel on the transfer
// port) and starts relaying. simdGenesis adds genesis accounts to simd, e.g.
// a user holding a second simd-origin denom. Everything, the relayer
// included, is removed when the test ends.
func StartIbc(t *testing.T, ctx context.Context, simdGenesis []ibc.WalletAmount, opts ...Option) IbcNetwork {
	t.Helper()

	sdkConfigOnce.Do(func() { cosmos.SetSDKConfig(Bech32Prefix) })

	logger := zaptest.NewLogger(t)
	cf := interchaintest.NewBuiltinChainFactory(logger, []*interchaintest.ChainSpec{BzeChainSpec(opts...), SimdChainSpec()})
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	bze, simd := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	dockerClient, network := interchaintest.DockerSetup(t)
	r := interchaintest.NewBuiltinRelayerFactory(ibc.Hermes, logger,
		relayer.CustomDockerImage(HermesImage, HermesVersion, hermesUIDGID),
	).Build(t, dockerClient, network)

	ic := interchaintest.NewInterchain().
		AddChain(bze).
		AddChain(simd, simdGenesis...).
		AddRelayer(r, "hermes").
		AddLink(interchaintest.InterchainLink{
			Chain1:  bze,
			Chain2:  simd,
			Relayer: r,
			Path:    IbcPath,
		})

	rep := testreporter.NewNopReporter().RelayerExecReporter(t)
	require.NoError(t, ic.Build(ctx, rep, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    dockerClient,
		NetworkID: network,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	channel, err := ibc.GetTransferChannel(ctx, r, rep, ChainID, SimdChainID)
	require.NoError(t, err)

	require.NoError(t, r.StartRelayer(ctx, rep, IbcPath))
	t.Cleanup(func() {
		_ = r.StopRelayer(ctx, rep)
	})

	return IbcNetwork{
		Bze:         bze,
		Simd:        simd,
		Relayer:     r,
		Reporter:    rep,
		BzeChannel:  channel.ChannelID,
		SimdChannel: channel.Counterparty.ChannelID,
	}
}
