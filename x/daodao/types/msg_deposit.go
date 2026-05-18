package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgDeposit{}

// ValidateBasic performs stateless validation of MsgDeposit.
//
// Stateless rules:
//   - depositor is valid bech32.
//   - dao_id and proposal_id are non-zero.
//   - amount is a valid Coin (positive, valid denom).
//
// Stateful checks deferred to the keeper:
//   - Proposal exists and is in PROPOSAL_STATUS_DEPOSIT_PERIOD.
//   - amount.denom matches proposal.deposit_snapshot.min_deposit.denom.
//   - depositor has the spendable balance to cover `amount`.
//   - Aggregating into the existing DepositRecord (if any).
//   - Transition to VOTING when collected >= min_deposit.
func (m *MsgDeposit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Depositor); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "depositor: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	if m.ProposalId == 0 {
		return errorsmod.Wrap(ErrProposalNotFound, "proposal_id must be non-zero")
	}
	if err := m.Amount.Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidDepositAmount, "amount: %s", err.Error())
	}
	if m.Amount.Amount.IsNil() || !m.Amount.Amount.IsPositive() {
		return errorsmod.Wrap(ErrInvalidDepositAmount, "amount must be > 0")
	}
	return nil
}
