package types

const (
	// ModuleName defines the module name
	ModuleName = "txfeecollector"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_txfeecollector"

	BurnerFeeCollector = "txfeecollector_burner"
	CpFeeCollector     = "txfeecollector_cp"

	FeeDestBurner        = "burner"
	FeeDestCommunityPool = "community_pool"
	FeeDestStakers       = "stakers"
)

var (
	ParamsKey = []byte("p_txfeecollector")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
