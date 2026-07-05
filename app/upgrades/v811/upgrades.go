package v811

// UpgradeName is the name validators use in the software-upgrade proposal for v8.1.1.
//
// v8.1.1 changes how LP denoms are derived for NEW liquidity pools: "ulp/<sha256(poolId)>"
// instead of "ulp_<base>_<quote>", which exceeded the SDK's 128-character denom limit when
// both assets were IBC denoms. Existing pools keep the LpDenom stored on their LiquidityPool
// object, so there is no state migration and no store change — the upgrade only coordinates
// the binary swap (see upgrades.EmptyUpgradeHandler in app.go).
const UpgradeName = "v8.1.1"
