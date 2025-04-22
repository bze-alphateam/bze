package types

const (
	// ModuleName defines the module name
	ModuleName = "burner"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_burner"
)

var (
	ParamsKey = []byte("p_burner")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
