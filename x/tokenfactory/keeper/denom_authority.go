package keeper

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) GetDenomAuthority(ctx sdk.Context, denom string) (types.DenomAuthority, error) {
	bz := k.GetDenomPrefixStore(ctx, denom).Get([]byte(types.DenomAuthorityMetadataKey))

	dAuth := types.DenomAuthority{}
	err := k.cdc.Unmarshal(bz, &dAuth)
	if err != nil {
		return types.DenomAuthority{}, err
	}

	return dAuth, nil
}

func (k Keeper) SetDenomAuthority(ctx sdk.Context, denom string, dAuth types.DenomAuthority) error {
	err := dAuth.Validate()
	if err != nil {
		return err
	}

	store := k.GetDenomPrefixStore(ctx, denom)
	bz, err := k.cdc.Marshal(&dAuth)
	if err != nil {
		return err
	}

	store.Set([]byte(types.DenomAuthorityMetadataKey), bz)
	return nil

}
