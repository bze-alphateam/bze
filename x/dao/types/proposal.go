package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
)

var _ codectypes.UnpackInterfacesMessage = &Proposal{}

// UnpackInterfaces implements UnpackInterfacesMessage, so the messages of a
// stored proposal are resolved when it is read.
func (p Proposal) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return sdktx.UnpackInterfaces(unpacker, p.Msgs)
}
