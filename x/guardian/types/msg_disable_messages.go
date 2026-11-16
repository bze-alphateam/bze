package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MaxDisableMessages is the most message types one MsgDisableMessages can
// disable.
const MaxDisableMessages = 100

var _ sdk.Msg = &MsgDisableMessages{}

// protectedMsgTypePrefixes are the message families that can never be
// disabled through x/guardian: governance, the circuit breaker itself (its
// reset is how governance re-enables a message), software upgrades and the
// guardian's own messages. Disabling any of them could lock governance or the
// guardian out.
var protectedMsgTypePrefixes = []string{
	"/cosmos.gov.",
	"/cosmos.circuit.",
	"/cosmos.upgrade.",
	"/bze.guardian.",
}

// protectedMsgTypeURLs are the x/dao messages a DAO needs to decide and act
// (the guardian may be a DAO): submit a proposal, vote on it and settle it.
var protectedMsgTypeURLs = map[string]struct{}{
	"/bze.dao.MsgSubmitProposal": {},
	"/bze.dao.MsgVote":           {},
	"/bze.dao.MsgSettleProposal": {},
}

// IsProtectedMsgTypeURL reports whether a message type can never be disabled
// through x/guardian.
func IsProtectedMsgTypeURL(msgTypeURL string) bool {
	if _, ok := protectedMsgTypeURLs[msgTypeURL]; ok {
		return true
	}

	for _, prefix := range protectedMsgTypePrefixes {
		if strings.HasPrefix(msgTypeURL, prefix) {
			return true
		}
	}

	return false
}

// ValidateBasic does a sanity check on the provided data. The keeper runs it
// too: a message executed by x/gov or x/authz does not pass through the
// transaction decoder.
func (m *MsgDisableMessages) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}

	if len(m.MsgTypeUrls) == 0 {
		return errorsmod.Wrap(ErrInvalidMsgTypeURLs, "at least one message type url is required")
	}

	if len(m.MsgTypeUrls) > MaxDisableMessages {
		return errorsmod.Wrapf(ErrInvalidMsgTypeURLs, "at most %d message type urls, got %d", MaxDisableMessages, len(m.MsgTypeUrls))
	}

	seen := make(map[string]struct{}, len(m.MsgTypeUrls))
	for _, url := range m.MsgTypeUrls {
		if !strings.HasPrefix(url, "/") || len(url) == 1 {
			return errorsmod.Wrapf(ErrInvalidMsgTypeURLs, "%q is not a type url (/package.Msg)", url)
		}

		if _, dup := seen[url]; dup {
			return errorsmod.Wrapf(ErrInvalidMsgTypeURLs, "duplicate %s", url)
		}
		seen[url] = struct{}{}

		if IsProtectedMsgTypeURL(url) {
			return errorsmod.Wrapf(ErrProtectedMsgType, "%s", url)
		}
	}

	return nil
}
