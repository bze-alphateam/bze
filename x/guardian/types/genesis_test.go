package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

func TestGenesis_Validate(t *testing.T) {
	require.NoError(t, types.DefaultGenesis().Validate())
	require.Empty(t, types.DefaultGenesis().Params.Guardian)

	good := types.GenesisState{Params: types.NewParams(sdk.AccAddress("guardian_address____").String())}
	require.NoError(t, good.Validate())

	bad := types.GenesisState{Params: types.NewParams("not-an-address")}
	require.Error(t, bad.Validate())
}
