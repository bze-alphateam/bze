package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// LP denoms are derived from the pool ID ("<base>_<quote>", assets sorted
// alphabetically) by hashing, mirroring the "ibc/<hash>" convention:
//
//	base denom (exponent 0):    ulp/<uppercase hex sha256(poolId)>  (68 chars)
//	display denom (exponent 6): lp/<uppercase hex sha256(poolId)>   (67 chars)
//
// Hashing keeps the denom at a fixed length regardless of the underlying
// asset denoms. The previous "ulp_<base>_<quote>" concatenation exceeded the
// SDK's 128-character denom limit when both assets were IBC denoms (68 chars
// each), making pool creation panic. Pools created before this scheme keep
// their legacy denom: the LP denom is generated once at pool creation and
// persisted on the LiquidityPool object, which is the only source of truth —
// never derive a pool's denom for an existing pool, read LiquidityPool.LpDenom.
const lpDenomNamespace = "lp"

// LpDenomHash returns the uppercase hex-encoded sha256 hash of a pool ID.
func LpDenomHash(poolId string) string {
	sum := sha256.Sum256([]byte(poolId))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// GetLpDenom returns the base (exponent 0) LP denom for a pool ID: "ulp/<hash>".
func GetLpDenom(poolId string) string {
	return fmt.Sprintf("u%s", GetLpScaledDenom(poolId))
}

// GetLpScaledDenom returns the display (exponent 6) LP denom for a pool ID: "lp/<hash>".
func GetLpScaledDenom(poolId string) string {
	return fmt.Sprintf("%s/%s", lpDenomNamespace, LpDenomHash(poolId))
}
