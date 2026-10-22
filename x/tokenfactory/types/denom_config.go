package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Caps of the denom config rates. Changing them needs a chain upgrade.
var (
	MaxTransferFee  = math.LegacyNewDecWithPrec(5, 2) // 0.05
	MaxGasSurcharge = math.LegacyNewDec(2)
)

// DefaultDenomConfig is the config of a denom without a stored entry:
// no community pool and zero rates. The rates are set explicitly because a
// LegacyDec read from an empty field is nil and panics on use.
func DefaultDenomConfig() DenomConfig {
	return DenomConfig{
		CommunityPool: "",
		TransferFee:   math.LegacyZeroDec(),
		GasSurcharge:  math.LegacyZeroDec(),
	}
}

// IsDefault reports whether the config equals the defaults.
func (c DenomConfig) IsDefault() bool {
	return c.CommunityPool == "" &&
		!c.TransferFee.IsNil() && c.TransferFee.IsZero() &&
		!c.GasSurcharge.IsNil() && c.GasSurcharge.IsZero()
}

// Validate checks the community pool address and that both rates are set and within their caps.
func (c DenomConfig) Validate() error {
	if c.CommunityPool != "" {
		if _, err := sdk.AccAddressFromBech32(c.CommunityPool); err != nil {
			return ErrInvalidDenomConfig.Wrapf("invalid community pool address: %s", err)
		}
	}

	if err := validateRate("transfer fee", c.TransferFee, MaxTransferFee); err != nil {
		return err
	}

	return validateRate("gas surcharge", c.GasSurcharge, MaxGasSurcharge)
}

func validateRate(name string, rate, maxRate math.LegacyDec) error {
	if rate.IsNil() {
		return ErrInvalidDenomConfig.Wrapf("%s is required", name)
	}
	if rate.IsNegative() {
		return ErrInvalidDenomConfig.Wrapf("%s cannot be negative: %s", name, rate)
	}
	if rate.GT(maxRate) {
		return ErrInvalidDenomConfig.Wrapf("%s %s is above the max %s", name, rate, maxRate)
	}

	return nil
}
