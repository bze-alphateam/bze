package guardian_test

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	guardian "github.com/bze-alphateam/bze/x/guardian/module"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

func TestValidateGenesis_MalformedJSONIsRefused(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	err := guardian.AppModuleBasic{}.ValidateGenesis(cdc, nil, json.RawMessage(`{"params": `))
	require.ErrorContains(t, err, "failed to unmarshal guardian genesis state")
}

func TestValidateGenesis_InvalidStateIsRefusedAndDefaultPasses(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	basic := guardian.AppModuleBasic{}

	require.NoError(t, basic.ValidateGenesis(cdc, nil, basic.DefaultGenesis(cdc)))

	bad := cdc.MustMarshalJSON(&types.GenesisState{Params: types.NewParams("nope")})
	require.ErrorContains(t, basic.ValidateGenesis(cdc, nil, bad), "invalid guardian address (nope)")
}
