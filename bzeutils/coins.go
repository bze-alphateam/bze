package bzeutils

import "strings"

func IsIBCDenom(denom string) bool {
	return strings.HasPrefix(denom, "ibc/")
}

// IsLpTokenDenom reports whether denom is a tradebin LP share: "ulp/<hash>" for pools created
// since v8.1.1, "ulp_<base>_<quote>" for older pools.
func IsLpTokenDenom(denom string) bool {
	return strings.HasPrefix(denom, "ulp/") || strings.HasPrefix(denom, "ulp_")
}

func IsTokenFactoryDenom(denom string) bool {
	return strings.HasPrefix(denom, "factory/")
}
