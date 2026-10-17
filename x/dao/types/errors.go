package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/dao module sentinel errors
var (
	ErrInvalidSigner     = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidDaoType    = sdkerrors.Register(ModuleName, 1101, "invalid dao type")
	ErrInvalidMetadata   = sdkerrors.Register(ModuleName, 1102, "invalid dao metadata")
	ErrInvalidConfig     = sdkerrors.Register(ModuleName, 1103, "invalid dao config")
	ErrInvalidMembers    = sdkerrors.Register(ModuleName, 1104, "invalid dao members")
	ErrInvalidElectorate = sdkerrors.Register(ModuleName, 1105, "invalid dao electorate")
	ErrDaoAccountExists  = sdkerrors.Register(ModuleName, 1106, "an account already exists at the dao address")
)
