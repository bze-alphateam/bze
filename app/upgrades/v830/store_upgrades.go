package v830

import (
	storetypes "cosmossdk.io/store/types"
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

// GetStoreUpgrades deletes the stores of the modules ibc-go v10 no longer has
// and of the unused x/group and x/nft modules.
func GetStoreUpgrades() *storetypes.StoreUpgrades {
	return &storetypes.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{capabilityStoreName, ibcFeeStoreName, groupStoreName, nftStoreName},
	}
}
