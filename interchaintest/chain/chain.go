// Package chain defines how the e2e suites start a bze network: the bzed
// Docker image, the interchaintest chain spec and the fast-epoch genesis.
package chain

import (
	"os"
	"slices"

	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
)

const (
	// ChainID is the chain id of every e2e bze network.
	ChainID = "bze-e2e-1"
	// Name is the interchaintest chain name.
	Name = "bze"
	// Bin is the daemon binary inside the image.
	Bin = "bzed"
	// Bech32Prefix is the account address prefix.
	Bech32Prefix = "bze"
	// Denom is the native denom.
	Denom = "ubze"
	// GasPrices is the minimum gas price set on every node and used by default for txs.
	GasPrices = "0.01" + Denom
	// GasAdjustment multiplies simulated gas.
	GasAdjustment = 1.5

	// ImageRepository is the local Docker repository built by scripts/e2e/build-image.sh.
	ImageRepository = "bze"
	// ImageTagEnv selects the default image tag.
	ImageTagEnv = "E2E_BZE_IMAGE_TAG"
	// DefaultImageTag is used when ImageTagEnv is unset.
	DefaultImageTag = "local"
	// ImageVersionLabel is the image label holding the version bzed was built
	// with (set by scripts/e2e/build-image.sh).
	ImageVersionLabel = "org.opencontainers.image.version"
	// ImageUIDGID is the non-root user of the image (see the repo Dockerfile).
	ImageUIDGID = "1025:1025"

	trustingPeriod = "336h"
)

// DefaultTag returns the image tag from E2E_BZE_IMAGE_TAG, or "local".
func DefaultTag() string {
	if tag := os.Getenv(ImageTagEnv); tag != "" {
		return tag
	}
	return DefaultImageTag
}

// BzeImage returns the bze:<tag> image; an empty tag means DefaultTag().
func BzeImage(tag string) ibc.DockerImage {
	if tag == "" {
		tag = DefaultTag()
	}
	return ibc.DockerImage{
		Repository: ImageRepository,
		Version:    tag,
		UIDGID:     ImageUIDGID,
	}
}

type spec struct {
	validators int
	image      string
	genesis    []cosmos.GenesisKV
}

// Option customises BzeChainSpec.
type Option func(*spec)

// WithValidators sets the number of validators (default 1).
func WithValidators(n int) Option {
	return func(s *spec) { s.validators = n }
}

// WithImage sets the bze image tag (default DefaultTag()).
func WithImage(tag string) Option {
	return func(s *spec) { s.image = tag }
}

// WithGenesis appends genesis overrides, applied after DefaultGenesis().
func WithGenesis(extra ...cosmos.GenesisKV) Option {
	return func(s *spec) { s.genesis = append(s.genesis, extra...) }
}

// WithoutGenesis drops the genesis overrides set so far on the given keys,
// e.g. DenomRewardLockKey for a release that predates the param (its genesis
// would not parse with an unknown field).
func WithoutGenesis(keys ...string) Option {
	return func(s *spec) {
		kept := s.genesis[:0:0]
		for _, kv := range s.genesis {
			if !slices.Contains(keys, kv.Key) {
				kept = append(kept, kv)
			}
		}
		s.genesis = kept
	}
}

// BzeChainSpec returns the interchaintest spec of a bze e2e network: one
// validator, no full nodes, 1 s blocks and the fast-epoch genesis.
func BzeChainSpec(opts ...Option) *interchaintest.ChainSpec {
	s := spec{validators: 1, image: DefaultTag(), genesis: DefaultGenesis()}
	for _, opt := range opts {
		opt(&s)
	}

	fullNodes := 0
	validators := s.validators
	image := BzeImage(s.image)

	return &interchaintest.ChainSpec{
		Name:      Name,
		ChainName: Name,
		Version:   image.Version,
		ChainConfig: ibc.ChainConfig{
			Type:           "cosmos",
			Name:           Name,
			ChainID:        ChainID,
			Images:         []ibc.DockerImage{image},
			Bin:            Bin,
			Bech32Prefix:   Bech32Prefix,
			Denom:          Denom,
			CoinType:       "118",
			GasPrices:      GasPrices,
			GasAdjustment:  GasAdjustment,
			TrustingPeriod: trustingPeriod,
			ModifyGenesis:  cosmos.ModifyGenesis(s.genesis),
			ConfigFileOverrides: map[string]any{
				"config/config.toml": testutil.Toml{
					"consensus": testutil.Toml{
						"timeout_commit": "1s",
					},
				},
				"config/app.toml": testutil.Toml{
					"minimum-gas-prices": GasPrices,
				},
			},
		},
		NumValidators: &validators,
		NumFullNodes:  &fullNodes,
	}
}
