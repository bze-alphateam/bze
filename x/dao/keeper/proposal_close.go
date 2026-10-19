package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// leaveVoting undoes what a proposal in voting holds: the pin on each of its
// sources (I4) and its open marker. Every path that ends VOTING (cancel,
// settlement) calls it exactly once per proposal.
func (k Keeper) leaveVoting(ctx sdk.Context, proposal types.Proposal) {
	for _, source := range proposal.PinnedElectorate.Sources() {
		k.unpinSource(ctx, source)
	}

	k.removeOpenMarker(ctx, proposal)
}

// releaseDeposit sends the proposal's escrowed deposit out of the module
// account: back to the proposer on a refund, otherwise to the
// kept_deposit_destination of the pinned config. A proposal without a deposit
// releases nothing.
func (k Keeper) releaseDeposit(ctx sdk.Context, proposal types.Proposal, refund bool) error {
	deposit := proposal.PinnedDeposit
	if deposit == nil || !deposit.IsPositive() {
		return nil
	}

	coins := sdk.NewCoins(*deposit)
	var (
		destination sdk.AccAddress
		err         error
	)
	switch {
	case refund:
		destination = sdk.MustAccAddressFromBech32(proposal.Proposer)
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, destination, coins)
	case proposal.PinnedConfig.KeptDepositDestination == types.FundsDestination_TREASURY:
		destination = types.DaoAddress(proposal.DaoId)
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, destination, coins)
	default:
		destination = authtypes.NewModuleAddress(txfeecollectortypes.BurnerFeeCollector)
		err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, txfeecollectortypes.BurnerFeeCollector, coins)
	}

	if err != nil {
		return errorsmod.Wrapf(err, "failed to release the deposit of proposal %d", proposal.Id)
	}

	return ctx.EventManager().EmitTypedEvent(&types.EventDepositReleased{
		Id:          proposal.Id,
		Refunded:    refund,
		Destination: destination.String(),
		Amount:      deposit.String(),
	})
}
