package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/cointrunk module sentinel errors
var (
	ErrInvalidProposalContent = sdkerrors.Register(ModuleName, 5, "invalid proposal content")
)
