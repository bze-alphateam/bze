package types_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestRegisterInterfaces_RegistersMsgUpdateParams(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgUpdateParams")
	require.NoError(t, err)
	require.IsType(t, &types.MsgUpdateParams{}, msg)
	require.Implements(t, (*sdk.Msg)(nil), msg)
}
