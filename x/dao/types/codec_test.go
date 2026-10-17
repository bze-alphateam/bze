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

func TestRegisterInterfaces_RegistersMsgWithdrawAdminNomination(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgWithdrawAdminNomination")
	require.NoError(t, err)
	require.IsType(t, &types.MsgWithdrawAdminNomination{}, msg)
}

func TestRegisterInterfaces_RegistersMsgSubmitProposal(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgSubmitProposal")
	require.NoError(t, err)
	require.IsType(t, &types.MsgSubmitProposal{}, msg)
}

func TestRegisterInterfaces_RegistersMsgVote(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgVote")
	require.NoError(t, err)
	require.IsType(t, &types.MsgVote{}, msg)
}

func TestRegisterInterfaces_RegistersMsgCancelProposal(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgCancelProposal")
	require.NoError(t, err)
	require.IsType(t, &types.MsgCancelProposal{}, msg)
}

func TestRegisterInterfaces_RegistersMsgSettleProposal(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgSettleProposal")
	require.NoError(t, err)
	require.IsType(t, &types.MsgSettleProposal{}, msg)
}

func TestRegisterInterfaces_RegistersMsgCleanupProposal(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgCleanupProposal")
	require.NoError(t, err)
	require.IsType(t, &types.MsgCleanupProposal{}, msg)
}

func TestRegisterInterfaces_RegistersMsgUpdateDaoConfig(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgUpdateDaoConfig")
	require.NoError(t, err)
	require.IsType(t, &types.MsgUpdateDaoConfig{}, msg)
}

func TestRegisterInterfaces_RegistersMsgUpdateMetadata(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msg, err := registry.Resolve("/bze.dao.MsgUpdateMetadata")
	require.NoError(t, err)
	require.IsType(t, &types.MsgUpdateMetadata{}, msg)
}
