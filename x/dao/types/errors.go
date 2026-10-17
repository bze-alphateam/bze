package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/dao module sentinel errors
var (
	ErrInvalidSigner      = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidDaoType     = sdkerrors.Register(ModuleName, 1101, "invalid dao type")
	ErrInvalidMetadata    = sdkerrors.Register(ModuleName, 1102, "invalid dao metadata")
	ErrInvalidConfig      = sdkerrors.Register(ModuleName, 1103, "invalid dao config")
	ErrInvalidMembers     = sdkerrors.Register(ModuleName, 1104, "invalid dao members")
	ErrInvalidElectorate  = sdkerrors.Register(ModuleName, 1105, "invalid dao electorate")
	ErrDaoAccountExists   = sdkerrors.Register(ModuleName, 1106, "an account already exists at the dao address")
	ErrSourcePinned       = sdkerrors.Register(ModuleName, 1107, "the reward is used by a dao electorate")
	ErrDaoNotFound        = sdkerrors.Register(ModuleName, 1108, "dao not found")
	ErrNotDaoAdmin        = sdkerrors.Register(ModuleName, 1109, "signer is not the dao's admin")
	ErrInvalidMsgs        = sdkerrors.Register(ModuleName, 1110, "invalid dao messages")
	ErrUnauthorizedSigner = sdkerrors.Register(ModuleName, 1111, "message must be signed by the dao account only")
	ErrNominationPending  = sdkerrors.Register(ModuleName, 1112, "an admin nomination is already pending")
	ErrElectorateNotLive  = sdkerrors.Register(ModuleName, 1113, "the dao electorate cannot vote")
	ErrInvalidNominee     = sdkerrors.Register(ModuleName, 1114, "invalid admin nominee")
)
