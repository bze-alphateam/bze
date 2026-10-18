package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
)

var _ codectypes.UnpackInterfacesMessage = &Proposal{}

// UnpackInterfaces implements UnpackInterfacesMessage, so the messages of a
// stored proposal are resolved when it is read.
func (p Proposal) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return sdktx.UnpackInterfaces(unpacker, p.Msgs)
}

// UnpackMsgs returns the proposal's messages.
func (p Proposal) UnpackMsgs() ([]sdk.Msg, error) {
	return sdktx.GetMsgs(p.Msgs, "dao proposal")
}

// IsFinished reports whether the status is terminal: nothing can change the
// proposal any more, so its ballots and record can be cleaned up. VOTING and
// EXECUTION_FAILED (retryable until it expires) are not.
func (s ProposalStatus) IsFinished() bool {
	switch s {
	case ProposalStatus_REJECTED, ProposalStatus_EXECUTED, ProposalStatus_EXPIRED, ProposalStatus_CANCELLED:
		return true
	default:
		return false
	}
}
