package types

const (
	// ModuleName defines the module name
	ModuleName = "guardian"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

var (
	ParamsKey = []byte("p_guardian")

	// ClosedChannelKeyPrefix holds one entry per closed transfer channel, keyed by
	// channel id.
	ClosedChannelKeyPrefix = []byte("closed_channel/")
)
