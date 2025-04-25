package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/cointrunk module sentinel errors
var (
	ErrInvalidProposalContent = sdkerrors.Register(ModuleName, 5, "invalid proposal content")
)
