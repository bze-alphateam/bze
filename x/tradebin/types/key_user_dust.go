package types

const (
	UserDustKeyPrefix = "Tb/ud/"
)

func UserDustKey(address, denom string) []byte {
	return []byte(address + "/" + denom + "/")
}

func UserDustKeyAddressPrefix(address string) []byte {
	return []byte(address + "/")
}
