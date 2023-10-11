package containers

// ImageConfig contains all images and their respective tags
// needed for running e2e tests.
type ImageConfig struct {
	InitRepository string
	InitTag        string

	BzeRepository string
	BzeTag        string

	RelayerRepository string
	RelayerTag        string
}

//nolint:deadcode
const (
	// Current Git branch Bze repo/version. It is meant to be built locally.
	// It is used when skipping upgrade by setting BZE_E2E_SKIP_UPGRADE to true).
	// This image should be pre-built with `make docker-build-debug` either in CI or locally.
	CurrentBranchBzeRepository = "bze"
	CurrentBranchBzeTag        = "debug"
	// Pre-upgrade bze repo/tag to pull.
	// It should be uploaded to Docker Hub. BZE_E2E_SKIP_UPGRADE should be unset
	// for this functionality to be used.
	previousVersionBzeRepository = "osmolabs/osmosis"
	previousVersionBzeTag        = "18.0.0-alpine"
	// Pre-upgrade repo/tag for osmosis initialization (this should be one version below upgradeVersion)
	previousVersionInitRepository = "osmolabs/osmosis-e2e-init-chain"
	previousVersionInitTag        = "v18-faster-blocks"
	// Hermes repo/version for relayer
	relayerRepository = "informalsystems/hermes"
	relayerTag        = "1.5.1"
)

// Returns ImageConfig needed for running e2e test.
// If isUpgrade is true, returns images for running the upgrade
// If isFork is true, utilizes provided fork height to initiate fork logic
func NewImageConfig(isUpgrade, isFork bool) ImageConfig {
	config := ImageConfig{
		RelayerRepository: relayerRepository,
		RelayerTag:        relayerTag,
	}

	if !isUpgrade {
		// If upgrade is not tested, we do not need InitRepository and InitTag
		// because we directly call the initialization logic without
		// the need for Docker.
		config.BzeRepository = CurrentBranchBzeRepository
		config.BzeTag = CurrentBranchBzeTag
		return config
	}

	// If upgrade is tested, we need to utilize InitRepository and InitTag
	// to initialize older state with Docker
	config.InitRepository = previousVersionInitRepository
	config.InitTag = previousVersionInitTag

	if isFork {
		// Forks are state compatible with earlier versions before fork height.
		// Normally, validators switch the binaries pre-fork height
		// Then, once the fork height is reached, the state breaking-logic
		// is run.
		config.BzeRepository = CurrentBranchBzeRepository
		config.BzeTag = CurrentBranchBzeTag
	} else {
		// Upgrades are run at the time when upgrade height is reached
		// and are submitted via a governance proposal. Thefore, we
		// must start running the previous Bze version. Then, the node
		// should auto-upgrade, at which point we can restart the updated
		// Bze validator container.
		config.BzeRepository = previousVersionBzeRepository
		config.BzeTag = previousVersionBzeTag
	}

	return config
}
