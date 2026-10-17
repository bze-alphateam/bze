package types

const (
	// ModuleName defines the module name
	ModuleName = "dao"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

var (
	ParamsKey = []byte("p_dao")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
