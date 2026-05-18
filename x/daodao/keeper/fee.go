package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// payCreationFee deducts Params.dao_creation_fee from `creator` and routes
// it according to Params.dao_creation_fee_destination:
//
//   - "burner"         → SendCoinsFromAccountToModule(creator, burner.ModuleName)
//   - "community_pool" → distrKeeper.FundCommunityPool(amount, creator)
//
// Caller must ensure params.DaoCreationFee.Amount > 0 before calling this.
// Insufficient balance bubbles up as ErrInsufficientCreationFee.
func (k Keeper) payCreationFee(ctx context.Context, creator sdk.AccAddress, params types.Params) error {
	fee := params.DaoCreationFee
	feeCoins := sdk.NewCoins(fee)
	if feeCoins.IsZero() {
		return nil
	}

	// Use SpendableCoins so vesting / otherwise-locked balances are excluded
	// from the check. That way the user gets our nice ErrInsufficientCreationFee
	// instead of a lower-level bank error from SendCoinsFromAccountToModule
	// later on.
	spendable := k.bankKeeper.SpendableCoins(ctx, creator).AmountOf(fee.Denom)
	if spendable.LT(fee.Amount) {
		return errorsmod.Wrapf(types.ErrInsufficientCreationFee,
			"creator %s has %s%s spendable but fee is %s", creator.String(), spendable.String(), fee.Denom, fee.String())
	}

	switch params.DaoCreationFeeDestination {
	case types.FeeDestinationBurnerModule:
		// Deposit into the burner module account. The burner module's own
		// processing eventually burns the coins.
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creator, burnertypes.ModuleName, feeCoins); err != nil {
			return fmt.Errorf("send fee to burner: %w", err)
		}
	case types.FeeDestinationCommunityPool:
		if err := k.distrKeeper.FundCommunityPool(ctx, feeCoins, creator); err != nil {
			return fmt.Errorf("fund community pool: %w", err)
		}
	default:
		// Should be unreachable — Params.Validate rejects unknown destinations.
		return fmt.Errorf("unknown fee destination %q", params.DaoCreationFeeDestination)
	}
	return nil
}
