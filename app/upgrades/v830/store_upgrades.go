package v830

import (
	storetypes "cosmossdk.io/store/types"

	daotypes "github.com/bze-alphateam/bze/x/dao/types"
)

const (
	// capabilityStoreName is the store key of the x/capability module
	// (capabilitytypes.StoreKey in ibc-go/modules/capability), removed with ibc-go v10.
	// Its memory store is transient and needs no deletion.
	capabilityStoreName = "capability"
	// ibcFeeStoreName is the store key of the ICS-29 fee middleware
	// (ibcfeetypes.StoreKey in ibc-go v8), removed with ibc-go v10.
	ibcFeeStoreName = "feeibc"
	// groupStoreName is the store key of x/group (group.StoreKey), removed from the app:
	// unused on mainnet and moved to a non-open license in cosmos-sdk 0.55.
	groupStoreName = "group"
	// nftStoreName is the store key of x/nft (nft.StoreKey), removed for the same reasons.
	nftStoreName = "nft"
)

// GetStoreUpgrades adds the store of the new x/dao module and deletes the stores
// of the modules ibc-go v10 no longer has and of the unused x/group and x/nft
// modules. x/dao has no VersionMap entry yet, so RunMigrations runs its InitGenesis.
func GetStoreUpgrades() *storetypes.StoreUpgrades {
	return &storetypes.StoreUpgrades{
		Added:   []string{daotypes.StoreKey},
		Deleted: []string{capabilityStoreName, ibcFeeStoreName, groupStoreName, nftStoreName},
	}
}
