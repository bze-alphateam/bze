package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgCreatePoll{}

// ValidateBasic performs stateless validation of MsgCreatePoll.
//
// Stateless rules:
//   - proposer is valid bech32.
//   - dao_id is non-zero.
//   - title is 1..MaxProposalTitleLen chars; description is
//     0..MaxProposalDescriptionLen chars (re-using Epic 3 caps).
//   - choices satisfies ValidatePollChoices (count, label length,
//     deduplication, NOTA-label reservation).
//   - max_selections in [1, len(user_choices)] — the NOTA index is not
//     counted toward the cap because [NOTA] alone is always legal.
//   - quorum_bps in [0, MaxQuorumBps].
//   - initial_deposit is structurally valid (Coin.Validate; amount may
//     be 0 — keeper enforces member-vs-non-member gating).
//
// Stateful checks deferred to the keeper:
//   - DAO exists.
//   - initial_deposit.denom matches DAO's min_deposit denom.
//   - Member/non-member gating against initial_deposit.amount.
//   - Bank send to escrow; snapshot.
func (m *MsgCreatePoll) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "proposer: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	if l := len(m.Title); l == 0 || l > MaxProposalTitleLen {
		return errorsmod.Wrapf(ErrInvalidPollContent,
			"title must be 1..%d chars, got %d", MaxProposalTitleLen, l)
	}
	if l := len(m.Description); l > MaxProposalDescriptionLen {
		return errorsmod.Wrapf(ErrInvalidPollContent,
			"description must be <= %d chars, got %d", MaxProposalDescriptionLen, l)
	}
	if err := ValidatePollChoices(m.Choices); err != nil {
		return err
	}
	if m.MaxSelections == 0 || int(m.MaxSelections) > len(m.Choices) {
		return errorsmod.Wrapf(ErrInvalidPollContent,
			"max_selections %d not in [1, %d]", m.MaxSelections, len(m.Choices))
	}
	if m.QuorumBps > MaxQuorumBps {
		return errorsmod.Wrapf(ErrInvalidPollContent,
			"quorum_bps %d exceeds cap %d", m.QuorumBps, MaxQuorumBps)
	}
	if err := m.InitialDeposit.Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidDepositAmount, "initial_deposit: %s", err.Error())
	}
	return nil
}
