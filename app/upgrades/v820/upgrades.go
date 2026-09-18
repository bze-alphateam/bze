package v820

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// UpgradeName is the name validators use in the software-upgrade proposal for v8.2.0.
const UpgradeName = "v8.2.0"

// BankKeeper is the slice of the bank keeper the v8.2.0 handler needs.
type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// AccountKeeper is the slice of the account keeper the v8.2.0 handler needs.
type AccountKeeper interface {
	GetModuleAddress(moduleName string) sdk.AccAddress
}

// TxfeecollectorKeeper is the slice of the x/txfeecollector keeper the v8.2.0 handler
// needs to write the inbound transfer block.
type TxfeecollectorKeeper interface {
	GetParams(ctx context.Context) txfeecollectortypes.Params
	SetParams(ctx context.Context, params txfeecollectortypes.Params) error
}

func CreateUpgradeHandler(
	cfg module.Configurator,
	mm *module.Manager,
	bank BankKeeper,
	acc AccountKeeper,
	txfeecollector TxfeecollectorKeeper,
) upgradetypes.UpgradeHandler {

	return func(c context.Context, _plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		// RunMigrations triggers module-level migrations based on ConsensusVersion
		// changes: rewards v4->v5 sets the Denom Rewards parameter defaults and
		// txfeecollector v2->v3 sets the empty BlockedIbcInbound default. No store
		// keys are added or removed, so no store loader is needed.
		newVm, err := mm.RunMigrations(ctx, cfg, vm)
		if err != nil {
			return newVm, err
		}

		if err := ApplyWindDown(ctx, bank, acc, txfeecollector); err != nil {
			return newVm, err
		}

		return newVm, nil
	}
}

// ApplyWindDown runs the Noble USDC wind-down for the chain the context belongs to.
// Chains without an entry in the table are left alone.
func ApplyWindDown(ctx sdk.Context, bank BankKeeper, acc AccountKeeper, txfeecollector TxfeecollectorKeeper) error {
	windDown, found := GetWindDown(ctx.ChainID())
	if !found {
		ctx.Logger().Info("no noble usdc wind-down configured for this chain, skipping", "chain_id", ctx.ChainID())

		return nil
	}

	if err := blockInboundTransfers(ctx, txfeecollector, windDown); err != nil {
		return err
	}

	transferBlackHoleLpShares(ctx, bank, acc, windDown)

	return nil
}

// blockInboundTransfers writes the blocked (channel, base denom) pairs into the
// x/txfeecollector params, leaving the other parameters untouched.
//
// An error here fails the upgrade on purpose: the values come from a hardcoded table
// that the package tests validate, so the only way to get one is a programming error
// that must not reach a running chain silently. Governance can still edit the param
// afterwards through MsgUpdateParams.
func blockInboundTransfers(ctx sdk.Context, txfeecollector TxfeecollectorKeeper, windDown WindDown) error {
	if len(windDown.BlockedInbound) == 0 {
		ctx.Logger().Info("no inbound ibc transfers to block, skipping")

		return nil
	}

	params := txfeecollector.GetParams(ctx)
	params.BlockedIbcInbound = windDown.BlockedInbound
	if err := params.Validate(); err != nil {
		return err
	}

	if err := txfeecollector.SetParams(ctx, params); err != nil {
		return err
	}

	for _, blocked := range windDown.BlockedInbound {
		ctx.Logger().Info("blocked inbound ibc transfer", "channel", blocked.ChannelId, "denom", blocked.BaseDenom)
	}

	return nil
}

// transferBlackHoleLpShares moves the black hole's whole balance of a single LP
// denomination to the admin address. Every other balance of the black hole - the other
// pools' LP shares and the locked coins - is untouched.
//
// It never returns an error: a failed value transfer must not brick the upgrade. When
// the transfer does not happen the shares simply stay where they are today, which is
// the pre-upgrade state, and the log line says why.
func transferBlackHoleLpShares(ctx sdk.Context, bank BankKeeper, acc AccountKeeper, windDown WindDown) {
	if windDown.LpDenom == "" {
		ctx.Logger().Info("no black hole lp shares to transfer, skipping")

		return
	}

	blackHole := acc.GetModuleAddress(burnertypes.BlackHoleModuleName)
	if blackHole == nil {
		ctx.Logger().Error("could not transfer black hole lp shares: black hole module address not found")

		return
	}

	balance := bank.GetBalance(ctx, blackHole, windDown.LpDenom)
	if !balance.Amount.IsPositive() {
		ctx.Logger().Info("black hole holds no lp shares of this denom, skipping", "denom", windDown.LpDenom)

		return
	}

	recipient, err := sdk.AccAddressFromBech32(windDown.AdminAddress)
	if err != nil {
		ctx.Logger().Error("could not transfer black hole lp shares: invalid admin address", "address", windDown.AdminAddress, "error", err)

		return
	}

	if err := bank.SendCoinsFromModuleToAccount(ctx, burnertypes.BlackHoleModuleName, recipient, sdk.NewCoins(balance)); err != nil {
		ctx.Logger().Error("could not transfer black hole lp shares", "denom", windDown.LpDenom, "amount", balance.Amount.String(), "error", err)

		return
	}

	// x/bank emits its own transfer event for the coins that just moved, so the upgrade
	// step adds only the log line naming why they moved.
	ctx.Logger().Info(
		"transferred black hole lp shares to the admin address",
		"denom", windDown.LpDenom,
		"amount", balance.Amount.String(),
		"recipient", windDown.AdminAddress,
	)
}
