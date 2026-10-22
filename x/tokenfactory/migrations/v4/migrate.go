package v4

import (
	"strings"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// Migrate sets the mintable flag on every denom authority record: denoms that still have an admin
// stay mintable, renounced denoms (empty admin) become non-mintable for good.
// It returns how many denoms ended up mintable and non-mintable.
func Migrate(store storetypes.KVStore, cdc codec.BinaryCodec) (mintable, nonMintable int, err error) {
	denomsStore := prefix.NewStore(store, []byte(types.DenomsPrefixKey+types.KeySeparator))
	authoritySuffix := types.KeySeparator + types.DenomAuthorityMetadataKey

	// collect first: writing while an iterator is open is not allowed on every store
	type record struct {
		key   []byte
		dAuth types.DenomAuthority
	}
	var records []record

	iterator := denomsStore.Iterator(nil, nil)
	for ; iterator.Valid(); iterator.Next() {
		if !strings.HasSuffix(string(iterator.Key()), authoritySuffix) {
			continue
		}

		var dAuth types.DenomAuthority
		if err = cdc.Unmarshal(iterator.Value(), &dAuth); err != nil {
			_ = iterator.Close()
			return 0, 0, err
		}

		records = append(records, record{key: iterator.Key(), dAuth: dAuth})
	}
	if err = iterator.Close(); err != nil {
		return 0, 0, err
	}

	for _, r := range records {
		r.dAuth.Mintable = r.dAuth.GetAdmin() != ""
		if r.dAuth.Mintable {
			mintable++
		} else {
			nonMintable++
		}

		bz, err := cdc.Marshal(&r.dAuth)
		if err != nil {
			return 0, 0, err
		}
		denomsStore.Set(r.key, bz)
	}

	return mintable, nonMintable, nil
}
