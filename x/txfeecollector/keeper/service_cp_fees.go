package keeper

import (
	"fmt"

	"github.com/bze-alphateam/bze/bzeutils"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ConvertCommunityPoolFees runs the community pool collector's pass over one page of txfeecollector_cp's balances
// (see readCollectorPage). Every coin reaches a community pool:
//   - native denom: funds the chain's community pool
//   - a denom with its own community pool: sent to that address, with a CommunityPoolFeePaidEvent
//   - LP shares: can never be swapped, sent to the burner module (which locks them)
//   - swappable for native denom: swapped, the result funds the chain's community pool
//   - anything else (too small to swap, no pool or a shallow one): kept in the collector until it can be swapped
//
// A failed send to a denom's community pool returns the error and the caller is expected to discard everything the
// pass wrote (EndBlock runs it through bzeutils.ApplyFuncIfNoError), the cursor move included. Such a send can only
// fail on a blocked recipient or a missing balance, neither of which should happen, so a failure is a bug: the coins
// stay in the collector, never rerouted, swapped or burned, until it is fixed.
func (k Keeper) ConvertCommunityPoolFees(ctx sdk.Context) error {
	moduleAddr := k.accountKeeper.GetModuleAddress(types.CpFeeCollector)
	page := k.readCollectorPage(ctx, types.CpFeeCollector, moduleAddr)
	if page.IsZero() {
		return nil
	}

	toFund := sdk.NewCoins()
	toSwap := sdk.NewCoins()
	toBurn := sdk.NewCoins()
	for _, c := range page {
		if !c.IsPositive() {
			continue
		}

		if k.tradeKeeper.IsNativeDenom(ctx, c.Denom) {
			toFund = toFund.Add(c)
			continue
		}

		if recipient, ok := k.denomCommunityPool(ctx, c.Denom); ok {
			if err := k.sendToDenomCommunityPool(ctx, recipient, c); err != nil {
				return err
			}
			continue
		}

		if bzeutils.IsLpTokenDenom(c.Denom) {
			toBurn = toBurn.Add(c)
			continue
		}

		if k.tradeKeeper.CanSwapForNativeDenom(ctx, c) {
			toSwap = toSwap.Add(c)
		}
		// otherwise it waits in the collector until it can be swapped
	}

	if !toSwap.IsZero() {
		swapped, err := k.tradeKeeper.ModuleSwapForNativeDenom(ctx, types.CpFeeCollector, toSwap)
		if err != nil {
			return err
		}
		toFund = toFund.Add(swapped)
	}

	if !toBurn.IsZero() {
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.CpFeeCollector, burnermoduletypes.ModuleName, toBurn)
		if err != nil {
			return err
		}
	}

	if toFund.IsZero() {
		return nil
	}

	return k.distrKeeper.FundCommunityPool(ctx, toFund, moduleAddr)
}

// denomCommunityPool returns the community pool the denom's fees are sent to, false when it has none.
func (k Keeper) denomCommunityPool(ctx sdk.Context, denom string) (sdk.AccAddress, bool) {
	if k.denomCpProvider == nil {
		return nil, false
	}

	return k.denomCpProvider.GetDenomCommunityPool(ctx, denom)
}

func (k Keeper) sendToDenomCommunityPool(ctx sdk.Context, recipient sdk.AccAddress, c sdk.Coin) error {
	err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.CpFeeCollector, recipient, sdk.NewCoins(c))
	if err != nil {
		return fmt.Errorf("failed to send %s to the denom community pool %s: %w", c, recipient, err)
	}

	return ctx.EventManager().EmitTypedEvent(&types.CommunityPoolFeePaidEvent{
		Denom:     c.Denom,
		Recipient: recipient.String(),
		Amount:    c.Amount.String(),
	})
}
