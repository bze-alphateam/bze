package types

import (
	errorsmod "cosmossdk.io/errors"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg                            = &MsgCreateProposal{}
	_ cdctypes.UnpackInterfacesMessage   = &MsgCreateProposal{}
)

// ValidateBasic performs stateless validation of MsgCreateProposal.
//
// Stateless rules:
//   - proposer is valid bech32.
//   - dao_id is non-zero.
//   - title is 1..MaxProposalTitleLen chars.
//   - description is 0..MaxProposalDescriptionLen chars.
//   - len(msgs) <= MaxMsgsPerProposal (Params-bounded; ValidateBasic only
//     reflects the loose "must decode" check — the keeper enforces the
//     param-driven cardinality cap because Params are not available here).
//   - each `msg` entry's cached value is non-nil (interface registry must
//     have already unpacked it via UnpackInterfaces, which the SDK pipeline
//     calls automatically on incoming txs).
//   - initial_deposit (Epic 4) is a structurally valid Coin (sdk.Coin.Validate
//     accepts denom + non-negative amount). amount == 0 is allowed at this
//     layer; the keeper rejects amount=0 for non-member proposers.
//
// Stateful checks deferred to the keeper:
//   - DAO exists and resolves a backend.
//   - proposer has Power(dao, proposer) > 0 (member-only submission in
//     Epic 3; relaxed in Epic 4 for deposit-attached non-members).
//   - msgs cardinality vs Params.max_msgs_per_proposal.
//   - msgs signers (Epic 5 dispatch validates that each msg signer ==
//     dao.account_address).
//   - initial_deposit.denom matches dao.deposit.min_deposit.denom.
//   - For non-member proposers, initial_deposit.amount >= min_deposit.amount.
func (m *MsgCreateProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "proposer: %s", err.Error())
	}
	if m.DaoId == 0 {
		return errorsmod.Wrap(ErrDaoNotFound, "dao_id must be non-zero")
	}
	if l := len(m.Title); l == 0 || l > MaxProposalTitleLen {
		return errorsmod.Wrapf(ErrInvalidProposalContent,
			"title must be 1..%d chars, got %d", MaxProposalTitleLen, l)
	}
	if l := len(m.Description); l > MaxProposalDescriptionLen {
		return errorsmod.Wrapf(ErrInvalidProposalContent,
			"description must be <= %d chars, got %d", MaxProposalDescriptionLen, l)
	}
	// Epic 4: initial_deposit must be structurally valid. sdk.Coin.Validate
	// accepts amount=0 with a valid denom; that's intentional — members may
	// submit with zero deposit, and the denom-vs-min_deposit check happens
	// in the keeper where the DAO config is available.
	if err := m.InitialDeposit.Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidDepositAmount, "initial_deposit: %s", err.Error())
	}
	// Every Any must have unpacked to a non-nil sdk.Msg. ValidateBasic
	// cannot check Params.max_msgs_per_proposal, but a wildly oversized
	// msgs slice can still be rejected with a defensive sanity cap of 256
	// here — anything beyond that is clearly malformed and lets us short-
	// circuit the keeper.
	const sanityMsgsCap = 256
	if len(m.Msgs) > sanityMsgsCap {
		return errorsmod.Wrapf(ErrInvalidProposalContent,
			"msgs has %d entries; sanity cap is %d (keeper enforces Params.max_msgs_per_proposal)",
			len(m.Msgs), sanityMsgsCap)
	}
	for i, anyMsg := range m.Msgs {
		if anyMsg == nil {
			return errorsmod.Wrapf(ErrInvalidProposalContent, "msgs[%d] is nil", i)
		}
		// The SDK tx pipeline calls UnpackInterfaces before ValidateBasic;
		// after that, GetCachedValue is non-nil for a successfully decoded
		// Any. A nil here means the wire bytes did not match any registered
		// concrete type — reject.
		if anyMsg.GetCachedValue() == nil {
			return errorsmod.Wrapf(ErrInvalidProposalContent,
				"msgs[%d] is not a registered sdk.Msg (TypeUrl=%q)", i, anyMsg.TypeUrl)
		}
		if _, ok := anyMsg.GetCachedValue().(sdk.Msg); !ok {
			return errorsmod.Wrapf(ErrInvalidProposalContent,
				"msgs[%d] decoded to %T which is not sdk.Msg", i, anyMsg.GetCachedValue())
		}
	}
	return nil
}

// UnpackInterfaces walks the Any-typed `msgs` slice and asks the registry
// to populate each Any's cached value. Required for codec round-trips and
// for ValidateBasic's sdk.Msg type check above. Invoked by the SDK's
// codec.UnpackInterfaces machinery on transaction decode.
func (m *MsgCreateProposal) UnpackInterfaces(unpacker cdctypes.AnyUnpacker) error {
	for _, anyMsg := range m.Msgs {
		var inner sdk.Msg
		if err := unpacker.UnpackAny(anyMsg, &inner); err != nil {
			return err
		}
	}
	return nil
}
