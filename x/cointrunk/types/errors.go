package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/cointrunk module sentinel errors
var (
	ErrInvalidSigner          = sdkerrors.Register(ModuleName, 1001, "expected gov account as only signer for proposal message")
	ErrInvalidProposalContent = sdkerrors.Register(ModuleName, 1002, "invalid proposal content")
)
