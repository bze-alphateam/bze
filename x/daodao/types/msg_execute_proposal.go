package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgExecuteProposal{}

// ValidateBasic performs stateless validation of MsgExecuteProposal.
//
// Rules:
//   - executor is valid bech32.
//   - dao_id and proposal_id are non-zero.
//
// Stateful checks deferred to the keeper:
//   - Proposal exists and is in PROPOSAL_STATUS_PASSED.
//   - Re-validate msgs[]' signers (defense-in-depth — Epic 5 plan).
//   - Dispatch each msg via the MsgServiceRouter inside a cached context.
func (m *MsgExecuteProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Executor); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "executor: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	if m.ProposalId == 0 {
		return errorsmod.Wrap(ErrProposalNotFound, "proposal_id must be non-zero")
	}
	return nil
}
