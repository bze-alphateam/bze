package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgUpdateGovernanceConfig{}

// ValidateBasic performs stateless validation of MsgUpdateGovernanceConfig.
//
// Rules:
//   - authority is valid bech32.
//   - dao_id is non-zero.
//   - governance passes the stateless brick-prevention caps
//     (ValidateGovernanceConfigStateless).
//
// Stateful checks deferred to the keeper:
//   - authority equals the DAO's admin.
//   - voting_period <= Params.max_voting_period.
//   - For REWARD_STAKED DAOs (once Epic 5 lands the backend swap), the
//     flash-vote rule StakingReward.lock >= voting_period.
func (m *MsgUpdateGovernanceConfig) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "authority: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	return ValidateGovernanceConfigStateless(m.Governance)
}
