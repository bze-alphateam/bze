package bzeutils

import "strings"

func IsIBCDenom(denom string) bool {
	return strings.HasPrefix(denom, "ibc/")
}

func IsLpTokenDenom(denom string) bool {
	return strings.HasPrefix(denom, "ulp_")
}

func IsTokenFactoryDenom(denom string) bool {
	return strings.HasPrefix(denom, "factory/")
}
