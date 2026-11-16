package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// GetDenomConfig returns the denom's config, or the defaults when none is stored.
func (k Keeper) GetDenomConfig(ctx sdk.Context, denom string) types.DenomConfig {
	bz := k.GetDenomPrefixStore(ctx, denom).Get([]byte(types.DenomConfigKey))
	if bz == nil {
		return types.DefaultDenomConfig()
	}

	var cfg types.DenomConfig
	k.cdc.MustUnmarshal(bz, &cfg)

	return cfg
}

// SetDenomConfig stores the denom's config. A config equal to the defaults deletes the entry,
// so the store holds no no-op records.
func (k Keeper) SetDenomConfig(ctx sdk.Context, denom string, cfg types.DenomConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	store := k.GetDenomPrefixStore(ctx, denom)
	if cfg.IsDefault() {
		store.Delete([]byte(types.DenomConfigKey))

		return nil
	}

	bz, err := k.cdc.Marshal(&cfg)
	if err != nil {
		return err
	}
	store.Set([]byte(types.DenomConfigKey), bz)

	return nil
}

// GetDenomCommunityPool returns the denom's community pool address, false when it has none.
//
// Every reader of a denom's community pool goes through here, so a stored address the bank refuses to send to
// (blocked) counts as no community pool: MsgSetDenomConfig refuses such an address, but one can become blocked after
// it was stored, and the denom's fees then follow the no-community-pool path instead of failing on the send.
func (k Keeper) GetDenomCommunityPool(ctx sdk.Context, denom string) (sdk.AccAddress, bool) {
	cfg := k.GetDenomConfig(ctx, denom)
	if cfg.CommunityPool == "" {
		return nil, false
	}

	addr, err := sdk.AccAddressFromBech32(cfg.CommunityPool)
	if err != nil {
		k.Logger().Error("invalid stored community pool address", "denom", denom, "error", err)

		return nil, false
	}

	if k.bankKeeper.BlockedAddr(addr) {
		return nil, false
	}

	return addr, true
}

// GetDenomFeeRates returns the denom's transfer fee and gas surcharge ratios, zeros when none are set.
func (k Keeper) GetDenomFeeRates(ctx sdk.Context, denom string) (transferFee, gasSurcharge math.LegacyDec) {
	cfg := k.GetDenomConfig(ctx, denom)

	return cfg.TransferFee, cfg.GasSurcharge
}
