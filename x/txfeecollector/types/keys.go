package types

const (
	// ModuleName defines the module name
	ModuleName = "txfeecollector"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_txfeecollector"

	CpFeeCollector = "txfeecollector_cp"
)

var (
	ParamsKey = []byte("p_txfeecollector")

	// CollectorCursorPrefix prefixes the cursor of a collector's EndBlock pass, one per collector module account:
	// the last denom the pass looked at. See CollectorCursorKey.
	CollectorCursorPrefix = []byte("collector_cursor/")
)

// CollectorCursorKey returns the store key of the module account's collector cursor.
func CollectorCursorKey(module string) []byte {
	return append(append([]byte{}, CollectorCursorPrefix...), module...)
}

func KeyPrefix(p string) []byte {
	return []byte(p)
}
