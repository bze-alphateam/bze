package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgUpdateVotingBackend{}

// ValidateBasic performs stateless validation of MsgUpdateVotingBackend.
//
// Rules:
//   - authority is valid bech32.
//   - dao_id is non-zero.
//   - voting_config is one of the two oneof variants.
//   - STATIC variant: members list satisfies ValidateStaticMembers.
//   - REWARD_STAKED variant: reward_id is non-empty.
//
// Stateful checks deferred to the keeper:
//   - authority equals the DAO's current admin.
//   - New backend TYPE matches current backend (v1 same-type only).
//   - For REWARD_STAKED: reward exists, creator == dao.account_address,
//     lock >= dao.governance.voting_period.
//   - For STATIC → STATIC: rejected with a "use MsgUpdateMembers" pointer
//     (intentional — one well-named op per concept).
func (m *MsgUpdateVotingBackend) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "authority: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	cfg := m.GetVotingConfig()
	if cfg == nil {
		return ErrMissingVotingConfig
	}
	switch v := cfg.(type) {
	case *MsgUpdateVotingBackend_Static:
		if v.Static == nil {
			return ErrMissingVotingConfig
		}
		return ValidateStaticMembers(v.Static.Members)
	case *MsgUpdateVotingBackend_RewardStaked:
		if v.RewardStaked == nil || v.RewardStaked.RewardId == "" {
			return errorsmod.Wrap(ErrMissingVotingConfig, "reward_staked.reward_id must be non-empty")
		}
		return nil
	default:
		return errorsmod.Wrapf(ErrMissingVotingConfig, "unknown voting_config type %T", cfg)
	}
}
