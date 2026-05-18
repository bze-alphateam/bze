package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgDepositOnPoll{}

// ValidateBasic performs stateless validation of MsgDepositOnPoll.
//
// Mirrors MsgDeposit's shape — same rules, just keyed by (dao, poll)
// instead of (dao, proposal). Denom-matches-poll-snapshot and
// status==DEPOSIT_PERIOD checks live in the keeper.
func (m *MsgDepositOnPoll) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Depositor); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "depositor: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	if m.PollId == 0 {
		return errorsmod.Wrap(ErrPollNotFound, "poll_id must be non-zero")
	}
	if err := m.Amount.Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidDepositAmount, "amount: %s", err.Error())
	}
	if m.Amount.Amount.IsNil() || !m.Amount.Amount.IsPositive() {
		return errorsmod.Wrap(ErrInvalidDepositAmount, "amount must be > 0")
	}
	return nil
}
