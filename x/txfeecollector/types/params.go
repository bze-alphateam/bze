package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

const DefaultMaxBalanceIterations = uint64(100)

// DefaultBlockedIbcInbound returns the default value of the BlockedIbcInbound param:
// no inbound transfer is blocked. Entries are added by governance through
// MsgUpdateParams (or, for the Noble USDC wind-down, by the v8.2.0 upgrade handler).
//
// It is nil rather than an empty slice so that default params round-trip through the
// store unchanged: protobuf decodes an absent repeated field as nil.
func DefaultBlockedIbcInbound() []BlockedIbcTransfer {
	return nil
}

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(validatorMinGasFee sdk.DecCoin, maxBalanceIterations uint64, blockedIbcInbound []BlockedIbcTransfer) Params {
	return Params{
		ValidatorMinGasFee:   validatorMinGasFee,
		MaxBalanceIterations: maxBalanceIterations,
		BlockedIbcInbound:    blockedIbcInbound,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		sdk.NewDecCoinFromDec("ubze", sdkmath.LegacyNewDecWithPrec(1, 2)), // 0.01ubze
		DefaultMaxBalanceIterations,
		DefaultBlockedIbcInbound(),
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateValidatorMinGasFee(p.ValidatorMinGasFee); err != nil {
		return err
	}

	if err := validateMaxBalanceIterations(p.MaxBalanceIterations); err != nil {
		return err
	}

	if err := validateBlockedIbcInbound(p.BlockedIbcInbound); err != nil {
		return err
	}

	return nil
}

// IsInboundBlocked reports whether an inbound ICS-20 packet carrying baseDenom and
// received on channelID must be refused. Both values are compared exactly as they
// appear on the packet: channelID is the packet's destination channel and baseDenom
// is the denomination as written by the sending chain.
func (p Params) IsInboundBlocked(channelID, baseDenom string) bool {
	for _, blocked := range p.BlockedIbcInbound {
		if blocked.ChannelId == channelID && blocked.BaseDenom == baseDenom {
			return true
		}
	}

	return false
}

func validateValidatorMinGasFee(i interface{}) error {
	v, ok := i.(sdk.DecCoin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.Denom != "ubze" {
		return fmt.Errorf("validator min gas fee denom must be ubze")
	}

	if v.Amount.IsNegative() {
		return fmt.Errorf("validator min gas fee amount cannot be negative: %s", v.Amount)
	}

	if !v.IsValid() {
		return fmt.Errorf("invalid validator min gas fee: %s", v)
	}

	return nil
}

func validateMaxBalanceIterations(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("max balance iterations must be greater than 0")
	}

	return nil
}

func validateBlockedIbcInbound(i interface{}) error {
	v, ok := i.([]BlockedIbcTransfer)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	seen := make(map[string]struct{}, len(v))
	for _, blocked := range v {
		if err := host.ChannelIdentifierValidator(blocked.ChannelId); err != nil {
			return fmt.Errorf("invalid blocked ibc inbound channel id %q: %w", blocked.ChannelId, err)
		}

		if err := sdk.ValidateDenom(blocked.BaseDenom); err != nil {
			return fmt.Errorf("invalid blocked ibc inbound base denom %q: %w", blocked.BaseDenom, err)
		}

		key := blocked.ChannelId + "/" + blocked.BaseDenom
		if _, found := seen[key]; found {
			return fmt.Errorf("duplicate blocked ibc inbound entry: %s", key)
		}
		seen[key] = struct{}{}
	}

	return nil
}
