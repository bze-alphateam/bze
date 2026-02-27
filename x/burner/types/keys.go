package types

const (
	// ModuleName defines the module name
	ModuleName = "burner"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_burner"
	// PeriodicBurnQueueKey is the store key for the periodic burn queue
	PeriodicBurnQueueKey = "pbq/"
)

var (
	ParamsKey = []byte("p_burner")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
