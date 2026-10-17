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

func TestRegisterInterfaces_RegistersMsgAdminExecute(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgAdminExecute")
	require.NoError(t, err)
	require.IsType(t, &types.MsgAdminExecute{}, msg)
}

func TestRegisterInterfaces_RegistersMsgNominateAdmin(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgNominateAdmin")
	require.NoError(t, err)
	require.IsType(t, &types.MsgNominateAdmin{}, msg)
}

func TestRegisterInterfaces_RegistersMsgAcceptAdminNomination(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgAcceptAdminNomination")
	require.NoError(t, err)
	require.IsType(t, &types.MsgAcceptAdminNomination{}, msg)
}
