package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// MaxPercent is the highest percent a quota can hold.
const MaxPercent = 100

var _ sdk.Msg = &MsgTightenRateLimit{}

// ValidateBasic does a sanity check on the provided data. The keeper runs it
// too: a message executed by x/gov or x/authz does not pass through the
// transaction decoder, and a negative percent would pass the "not above the
// current quota" check.
func (m *MsgTightenRateLimit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errorsmod.Wrap(err, "invalid signer address")
	}

	if m.Denom == "" {
		return errorsmod.Wrap(ErrInvalidQuota, "denom is empty")
	}

	if err := host.ChannelIdentifierValidator(m.ChannelId); err != nil {
		return errorsmod.Wrap(ErrInvalidQuota, err.Error())
	}

	if err := validatePercent("max_percent_send", m.MaxPercentSend); err != nil {
		return err
	}

	return validatePercent("max_percent_recv", m.MaxPercentRecv)
}

func validatePercent(name string, percent math.Int) error {
	if percent.IsNil() {
		return errorsmod.Wrapf(ErrInvalidQuota, "%s is not set", name)
	}

	if percent.IsNegative() || percent.GT(math.NewInt(MaxPercent)) {
		return errorsmod.Wrapf(ErrInvalidQuota, "%s must be between 0 and %d, got %s", name, MaxPercent, percent)
	}

	return nil
}
