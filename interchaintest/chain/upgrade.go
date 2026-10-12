package chain

import "os"

// Environment variables of the upgrade suite.
const (
	// UpgradeFromEnv is the bze image tag the network starts on: the previous release.
	UpgradeFromEnv = "E2E_UPGRADE_FROM"
	// UpgradeToEnv is the bze image tag the validators switch to at the upgrade height.
	UpgradeToEnv = "E2E_UPGRADE_TO"
	// UpgradeNameEnv is the plan name; it must equal the candidate's UpgradeName.
	UpgradeNameEnv = "E2E_UPGRADE_NAME"
)

// Upgrade defaults: the next release (v8.3.0) upgrades from v8.2.0. There is no
// final v8.2.0 tag yet, so the latest release candidate stands in for it; switch
// to v8.2.0 once it is tagged.
const (
	DefaultUpgradeFrom = "v8.2.0-rc5"
	DefaultUpgradeTo   = DefaultImageTag
	DefaultUpgradeName = "v8.3.0"
)

// Upgrade is one software upgrade the upgrade suite runs: start on the From
// image, pass a plan named Name, switch to the To image at the plan height.
type Upgrade struct {
	From, To, Name string
}

// UpgradeFromEnvironment reads the upgrade from E2E_UPGRADE_FROM,
// E2E_UPGRADE_TO and E2E_UPGRADE_NAME, with the defaults above for unset ones.
func UpgradeFromEnvironment() Upgrade {
	return Upgrade{
		From: envOr(UpgradeFromEnv, DefaultUpgradeFrom),
		To:   envOr(UpgradeToEnv, DefaultUpgradeTo),
		Name: envOr(UpgradeNameEnv, DefaultUpgradeName),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
