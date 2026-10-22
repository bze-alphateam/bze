package types

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	// MinPoolFee and MaxPoolFee bound a liquidity pool's swap fee (0.1% to 5%).
	MinPoolFee = math.LegacyNewDecWithPrec(1, 3)
	MaxPoolFee = math.LegacyNewDecWithPrec(5, 2)
)

// ValidatePoolFee checks a liquidity pool's swap fee: positive, at least MinPoolFee and at most MaxPoolFee.
// Shared by pool creation and MsgUpdatePools.
func ValidatePoolFee(fee math.LegacyDec) error {
	if !fee.IsPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, "fee must be positive")
	}

	if fee.LT(MinPoolFee) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, "fee must be at least 0.001 (0.1%)")
	}

	if fee.GT(MaxPoolFee) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, "fee must be at most 0.05 (5%)")
	}

	return nil
}

// ValidatePoolFeeDestination checks a liquidity pool's fee split: no negative share and the shares add up
// to exactly 1. Shared by pool creation and MsgUpdatePools.
func ValidatePoolFeeDestination(feeDest FeeDestination) error {
	//do not allow any of the destinations to be negative
	if feeDest.Treasury.IsNegative() || feeDest.Burner.IsNegative() || feeDest.Providers.IsNegative() {
		return ErrNegativeFeeDestination
	}

	//the sum of elements must be 1
	sum := feeDest.Burner.Add(feeDest.Treasury).Add(feeDest.Providers)
	if !sum.Equal(math.LegacyOneDec()) {
		return ErrInvalidFeeDestination
	}

	return nil
}
