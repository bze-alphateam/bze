// Package chain defines how the e2e suites start a bze network: the bzed
// Docker image, the interchaintest chain spec and the fast-epoch genesis.
package chain

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
)

const (
	// ChainID is the default chain id of the e2e bze networks (WithChainID).
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
	chainID      string
	validators   int
	image        string
	genesis      []cosmos.GenesisKV
	edits        []func(map[string]any) error
	minGasPrices string
}

// Option customises BzeChainSpec.
type Option func(*spec)

// WithValidators sets the number of validators (default 1).
func WithValidators(n int) Option {
	return func(s *spec) { s.validators = n }
}

// WithChainID sets the chain id (default ChainID), e.g. a live chain's id for
// code that branches on it, like an upgrade handler.
func WithChainID(id string) Option {
	return func(s *spec) { s.chainID = id }
}

// WithImage sets the bze image tag (default DefaultTag()).
func WithImage(tag string) Option {
	return func(s *spec) { s.image = tag }
}

// WithMinGasPrices sets the nodes' minimum-gas-prices (default GasPrices).
// Txs still pay GasPrices by default.
func WithMinGasPrices(prices string) Option {
	return func(s *spec) { s.minGasPrices = prices }
}

// WithGenesis appends genesis overrides, applied after DefaultGenesis().
func WithGenesis(extra ...cosmos.GenesisKV) Option {
	return func(s *spec) { s.genesis = append(s.genesis, extra...) }
}

// WithGenesisEdit edits the genesis JSON after the overrides, e.g. to append
// bank balances or module state a key override would replace whole.
func WithGenesisEdit(edit func(genesis map[string]any) error) Option {
	return func(s *spec) { s.edits = append(s.edits, edit) }
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
	s := spec{chainID: ChainID, validators: 1, image: DefaultTag(), genesis: DefaultGenesis(), minGasPrices: GasPrices}
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
			ChainID:        s.chainID,
			Images:         []ibc.DockerImage{image},
			Bin:            Bin,
			Bech32Prefix:   Bech32Prefix,
			Denom:          Denom,
			CoinType:       "118",
			GasPrices:      GasPrices,
			GasAdjustment:  GasAdjustment,
			TrustingPeriod: trustingPeriod,
			ModifyGenesis:  modifyGenesis(s.genesis, s.edits),
			ConfigFileOverrides: map[string]any{
				"config/config.toml": testutil.Toml{
					"consensus": testutil.Toml{
						"timeout_commit": "1s",
					},
				},
				"config/app.toml": testutil.Toml{
					"minimum-gas-prices": s.minGasPrices,
				},
			},
		},
		NumValidators: &validators,
		NumFullNodes:  &fullNodes,
	}
}

// modifyGenesis applies the key overrides, then the edits.
func modifyGenesis(kvs []cosmos.GenesisKV, edits []func(map[string]any) error) func(ibc.ChainConfig, []byte) ([]byte, error) {
	overrides := cosmos.ModifyGenesis(kvs)
	return func(cfg ibc.ChainConfig, genesis []byte) ([]byte, error) {
		out, err := overrides(cfg, genesis)
		if err != nil || len(edits) == 0 {
			return out, err
		}
		var g map[string]any
		if err := json.Unmarshal(out, &g); err != nil {
			return nil, fmt.Errorf("decode genesis: %w", err)
		}
		for _, edit := range edits {
			if err := edit(g); err != nil {
				return nil, fmt.Errorf("edit genesis: %w", err)
			}
		}
		return json.Marshal(g)
	}
}
