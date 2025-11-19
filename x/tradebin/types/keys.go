package types

const (
	// ModuleName defines the module name
	ModuleName = "tradebin"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_tradebin"
)

var (
	ParamsKey = []byte("p_tradebin")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
