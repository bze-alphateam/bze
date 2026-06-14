package v3

import (
	"cosmossdk.io/store/prefix"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrate reads existing params, sets defaults for the new CwDeployFee fields,
// and writes the updated params back to the store.
func Migrate(
	_ sdk.Context,
	store prefix.Store,
	cdc codec.BinaryCodec,
) error {
	var params types.Params

	bz := store.Get(types.ParamsKey)
	if bz != nil {
		cdc.MustUnmarshal(bz, &params)
	} else {
		params = types.DefaultParams()
	}

	// Set defaults for the new CosmWasm fee fields. The deploy fee is the real
	// anti-spam gate (it controls bytecode upload on a permissionless chain),
	// so it's set high at 50,000 BZE (50,000,000,000 ubze). Instantiation only
	// runs already-uploaded, already-paid-for code and is gas-metered, so it's
	// kept low at 10 BZE (10,000,000 ubze). Both route to the same destination
	// configured by CwDeployFeeDestination.
	params.CwDeployFeeDestination = types.DefaultCwDeployFeeDestination
	params.CwDeployFee = sdk.NewCoins(sdk.NewInt64Coin("ubze", 50000000000))
	params.CwInstantiateFee = sdk.NewCoins(sdk.NewInt64Coin("ubze", 10000000))

	if err := params.Validate(); err != nil {
		return err
	}

	bz = cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)

	return nil
}
