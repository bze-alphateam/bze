package keeper

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) SetDenomBranding(ctx sdk.Context, denom string, branding types.DenomBranding) error {
	err := branding.Validate()
	if err != nil {
		return err
	}

	store := k.GetBrandingsPrefixStore(ctx)
	bz, err := k.cdc.Marshal(&branding)
	if err != nil {
		return err
	}

	store.Set([]byte(denom), bz)

	return nil
}

func (k Keeper) GetDenomBranding(ctx sdk.Context, denom string) (types.DenomBranding, bool) {
	bz := k.GetBrandingsPrefixStore(ctx).Get([]byte(denom))
	if bz == nil {
		return types.DenomBranding{}, false
	}

	var branding types.DenomBranding
	k.cdc.MustUnmarshal(bz, &branding)

	return branding, true
}

func (k Keeper) RemoveDenomBranding(ctx sdk.Context, denom string) {
	k.GetBrandingsPrefixStore(ctx).Delete([]byte(denom))
}

func (k Keeper) GetAllDenomBrandings(ctx sdk.Context) []types.DenomBrandingRecord {
	records := make([]types.DenomBrandingRecord, 0)
	iterator := k.GetBrandingsPrefixStore(ctx).Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var branding types.DenomBranding
		k.cdc.MustUnmarshal(iterator.Value(), &branding)
		records = append(records, types.DenomBrandingRecord{
			Denom:    string(iterator.Key()),
			Branding: &branding,
		})
	}

	return records
}
