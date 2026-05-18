package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Field length limits for DaoMetadata. Hardcoded — these are UI/UX bounds,
// not protocol policy, so they don't earn a Params slot.
const (
	MaxDaoNameLen        = 128
	MaxDaoDescriptionLen = 4096
	MaxDaoImageURLLen    = 512
	MaxDaoLinkLen        = 256

	// MaxStaticMembers is the hardcoded ceiling on a STATIC DAO's member
	// count. Bounded so SnapshotAll iteration cost stays reasonable per
	// proposal creation (Epic 3 cost analysis).
	MaxStaticMembers = 10_000
)

var _ sdk.Msg = &MsgCreateDao{}

// ValidateBasic performs stateless validation of MsgCreateDao.
func (m *MsgCreateDao) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "creator: %s", err.Error())
	}
	if m.Admin != "" {
		if _, err := sdk.AccAddressFromBech32(m.Admin); err != nil {
			return errorsmod.Wrapf(ErrInvalidAddress, "admin: %s", err.Error())
		}
	}
	if err := ValidateDaoMetadata(m.Metadata); err != nil {
		return err
	}
	if err := validateCreateDaoVotingConfig(m); err != nil {
		return err
	}
	// Epic 3: governance config is required at creation and must satisfy the
	// stateless brick-prevention caps. The keeper additionally enforces the
	// Param-dependent voting_period upper bound (and, for REWARD_STAKED DAOs
	// once Epic 5 lands the backend swap, the flash-vote lock check).
	if err := ValidateGovernanceConfigStateless(m.Governance); err != nil {
		return err
	}
	// Epic 4: deposit config is also required at creation. The keeper
	// enforces the Param-driven deposit_period upper bound.
	return ValidateDepositConfigStateless(m.Deposit)
}

// validateCreateDaoVotingConfig enforces that voting_config is present and
// that the selected variant is allowed at creation time.
//
// Epic 2 accepts only the STATIC variant. REWARD_STAKED is reserved in the
// proto and explicitly rejected here — chicken-and-egg: a reward program
// owned by the DAO can only be created after the DAO exists, so the
// backend swap happens via Epic 5's MsgUpdateVotingBackend instead.
func validateCreateDaoVotingConfig(m *MsgCreateDao) error {
	cfg := m.GetVotingConfig()
	if cfg == nil {
		return ErrMissingVotingConfig
	}
	switch v := cfg.(type) {
	case *MsgCreateDao_Static:
		if v.Static == nil {
			return ErrMissingVotingConfig
		}
		return ValidateStaticMembers(v.Static.Members)
	case *MsgCreateDao_RewardStaked:
		return errorsmod.Wrap(ErrVotingConfigNotAllowed,
			"REWARD_STAKED at creation is rejected; create a STATIC DAO and swap backends via MsgUpdateVotingBackend (Epic 5)")
	default:
		// Should be unreachable — the oneof has only two variants.
		return errorsmod.Wrapf(ErrMissingVotingConfig, "unknown voting_config type %T", cfg)
	}
}

// ValidateStaticMembers enforces the STATIC member list invariants used at
// both creation time (MsgCreateDao) and update time (MsgUpdateMembers's
// `add` slice).
//
// Rules:
//   - 1..MaxStaticMembers entries
//   - every address is valid bech32
//   - no duplicate addresses
//   - every weight is strictly positive
func ValidateStaticMembers(members []StaticMember) error {
	if len(members) == 0 {
		return errorsmod.Wrap(ErrInvalidStaticMembers, "member list is empty")
	}
	if len(members) > MaxStaticMembers {
		return errorsmod.Wrapf(ErrInvalidStaticMembers, "too many members: got %d, max %d", len(members), MaxStaticMembers)
	}
	seen := make(map[string]struct{}, len(members))
	for i, mem := range members {
		addr, err := sdk.AccAddressFromBech32(mem.Address)
		if err != nil {
			return errorsmod.Wrapf(ErrInvalidStaticMembers, "members[%d].address: %s", i, err.Error())
		}
		key := addr.String() // normalize
		if _, dup := seen[key]; dup {
			return errorsmod.Wrapf(ErrInvalidStaticMembers, "duplicate member address %s", key)
		}
		seen[key] = struct{}{}
		if mem.Weight == 0 {
			return errorsmod.Wrapf(ErrInvalidStaticMembers, "members[%d].weight must be > 0", i)
		}
	}
	return nil
}

// ValidateDaoMetadata enforces length caps on every DaoMetadata field.
// Shared between MsgCreateDao and MsgUpdateDaoMetadata.
func ValidateDaoMetadata(md DaoMetadata) error {
	if l := len(md.Name); l == 0 || l > MaxDaoNameLen {
		return errorsmod.Wrapf(ErrInvalidMetadata, "name must be 1..%d chars, got %d", MaxDaoNameLen, l)
	}
	if l := len(md.Description); l > MaxDaoDescriptionLen {
		return errorsmod.Wrapf(ErrInvalidMetadata, "description must be ≤ %d chars, got %d", MaxDaoDescriptionLen, l)
	}
	if l := len(md.ImageUrl); l > MaxDaoImageURLLen {
		return errorsmod.Wrapf(ErrInvalidMetadata, "image_url must be ≤ %d chars, got %d", MaxDaoImageURLLen, l)
	}
	for name, v := range map[string]string{
		"twitter":  md.Twitter,
		"discord":  md.Discord,
		"telegram": md.Telegram,
		"website":  md.Website,
		"other":    md.Other,
	} {
		if l := len(v); l > MaxDaoLinkLen {
			return errorsmod.Wrapf(ErrInvalidMetadata, "%s must be ≤ %d chars, got %d", name, MaxDaoLinkLen, l)
		}
	}
	return nil
}
