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
)

// GetStoreUpgrades deletes the stores of the modules ibc-go v10 no longer has.
func GetStoreUpgrades() *storetypes.StoreUpgrades {
	return &storetypes.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{capabilityStoreName, ibcFeeStoreName},
	}
}
