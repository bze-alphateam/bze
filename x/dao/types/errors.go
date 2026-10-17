package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/dao module sentinel errors
var (
	ErrInvalidSigner       = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidDaoType      = sdkerrors.Register(ModuleName, 1101, "invalid dao type")
	ErrInvalidMetadata     = sdkerrors.Register(ModuleName, 1102, "invalid dao metadata")
	ErrInvalidConfig       = sdkerrors.Register(ModuleName, 1103, "invalid dao config")
	ErrInvalidMembers      = sdkerrors.Register(ModuleName, 1104, "invalid dao members")
	ErrInvalidElectorate   = sdkerrors.Register(ModuleName, 1105, "invalid dao electorate")
	ErrDaoAccountExists    = sdkerrors.Register(ModuleName, 1106, "an account already exists at the dao address")
	ErrSourcePinned        = sdkerrors.Register(ModuleName, 1107, "the reward is used by a dao electorate")
	ErrDaoNotFound         = sdkerrors.Register(ModuleName, 1108, "dao not found")
	ErrNotDaoAdmin         = sdkerrors.Register(ModuleName, 1109, "signer is not the dao's admin")
	ErrInvalidMsgs         = sdkerrors.Register(ModuleName, 1110, "invalid dao messages")
	ErrUnauthorizedSigner  = sdkerrors.Register(ModuleName, 1111, "message must be signed by the dao account only")
	ErrNominationPending   = sdkerrors.Register(ModuleName, 1112, "an admin nomination is already pending")
	ErrElectorateNotLive   = sdkerrors.Register(ModuleName, 1113, "the dao electorate cannot vote")
	ErrInvalidNominee      = sdkerrors.Register(ModuleName, 1114, "invalid admin nominee")
	ErrNoPendingNomination = sdkerrors.Register(ModuleName, 1115, "no admin nomination is pending")
	ErrNotNominee          = sdkerrors.Register(ModuleName, 1116, "signer is not the dao's pending admin")
	ErrInvalidProposal     = sdkerrors.Register(ModuleName, 1117, "invalid proposal")
	ErrNotProposer         = sdkerrors.Register(ModuleName, 1118, "signer cannot submit proposals to this dao")
	ErrNoVotingPower       = sdkerrors.Register(ModuleName, 1119, "no voting power")
	ErrTooManyProposals    = sdkerrors.Register(ModuleName, 1120, "the dao has too many open proposals")
	ErrExpediteNotAllowed  = sdkerrors.Register(ModuleName, 1121, "the dao does not allow expedited proposals")
	ErrInvalidVote         = sdkerrors.Register(ModuleName, 1122, "invalid vote")
	ErrProposalNotFound    = sdkerrors.Register(ModuleName, 1123, "proposal not found")
	ErrVotingClosed        = sdkerrors.Register(ModuleName, 1124, "the proposal is not open for voting")
	ErrSameVote            = sdkerrors.Register(ModuleName, 1125, "the ballot already has this option")
	ErrNotProposalOwner    = sdkerrors.Register(ModuleName, 1126, "signer is not the proposal's proposer")
)
