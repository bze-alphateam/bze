package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgVoteOnPoll{}

// ValidateBasic performs stateless validation of MsgVoteOnPoll.
//
// Rules:
//   - voter is valid bech32.
//   - dao_id and poll_id are non-zero.
//   - choice_indices is non-empty.
//
// The richer rules — index range, duplicate detection, NOTA exclusivity,
// max_selections — depend on the target Poll's stored `choices` slice and
// `include_nota` flag, so they live in the keeper (which has access to
// the Poll record). The keeper calls ValidatePollSelection there.
func (m *MsgVoteOnPoll) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Voter); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "voter: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	if m.PollId == 0 {
		return errorsmod.Wrap(ErrPollNotFound, "poll_id must be non-zero")
	}
	if len(m.ChoiceIndices) == 0 {
		return errorsmod.Wrap(ErrInvalidPollSelection, "choice_indices is empty")
	}
	return nil
}
