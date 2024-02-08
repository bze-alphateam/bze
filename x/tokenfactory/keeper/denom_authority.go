package keeper

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) setDenomAuthority(ctx sdk.Context, denom string, dAuth types.DenomAuthority) error {
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
