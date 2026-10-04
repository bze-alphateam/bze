package types

const (
	// HaltedDenomKeyPrefix is the prefix under which every governance-halted denom is stored as its own
	// key (the denom itself) with a one-byte value. One key per denom keeps the halt check a single store
	// Has() at a flat gas cost, however many denoms are halted.
	HaltedDenomKeyPrefix = "halted_denom/"
)

func HaltedDenomPrefix() []byte {
	return []byte(HaltedDenomKeyPrefix)
}

func HaltedDenomKey(denom string) []byte {
	return []byte(denom)
}
