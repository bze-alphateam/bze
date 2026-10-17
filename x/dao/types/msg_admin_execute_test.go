package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

var (
	execAdmin = sdk.AccAddress("dao_admin___________")
	execDao   = sdk.AccAddress("dao_account_________")
)

func treasurySend() sdk.Msg {
	return banktypes.NewMsgSend(execDao, execAdmin, sdk.NewCoins(sdk.NewInt64Coin("ubze", 1)))
}

func TestMsgAdminExecute_ValidateBasic(t *testing.T) {
	for _, tc := range []struct {
		desc    string
		admin   string
		daoId   uint64
		msgs    []sdk.Msg
		wantErr error
	}{
		{desc: "valid", admin: execAdmin.String(), daoId: 1, msgs: []sdk.Msg{treasurySend()}},
		{desc: "invalid admin", admin: "bze1nope", daoId: 1, msgs: []sdk.Msg{treasurySend()}, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty admin", admin: "", daoId: 1, msgs: []sdk.Msg{treasurySend()}, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero dao id", admin: execAdmin.String(), daoId: 0, msgs: []sdk.Msg{treasurySend()}, wantErr: types.ErrDaoNotFound},
		{desc: "no msgs", admin: execAdmin.String(), daoId: 1, wantErr: types.ErrInvalidMsgs},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			msg, err := types.NewMsgAdminExecute(tc.admin, tc.daoId, tc.msgs)
			require.NoError(t, err)

			err = msg.ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// An Any the codec has not unpacked holds no sdk.Msg.
func TestMsgAdminExecute_ValidateBasic_UnpackedAnyRejected(t *testing.T) {
	msg := &types.MsgAdminExecute{
		Admin: execAdmin.String(),
		DaoId: 1,
		Msgs:  []*codectypes.Any{{TypeUrl: "/cosmos.bank.v1beta1.MsgSend"}},
	}

	require.ErrorIs(t, msg.ValidateBasic(), types.ErrInvalidMsgs)
}

// The inner msgs survive a codec round trip: MsgAdminExecute unpacks its Anys.
func TestMsgAdminExecute_UnpacksMsgsAfterDecoding(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	msg, err := types.NewMsgAdminExecute(execAdmin.String(), 1, []sdk.Msg{treasurySend()})
	require.NoError(t, err)
	packed, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	bz, err := cdc.MarshalJSON(packed)
	require.NoError(t, err)

	var decoded codectypes.Any
	require.NoError(t, cdc.UnmarshalJSON(bz, &decoded))
	var got sdk.Msg
	require.NoError(t, cdc.UnpackAny(&decoded, &got))

	msgs, err := got.(*types.MsgAdminExecute).UnpackMsgs()
	require.NoError(t, err)
	require.Equal(t, []sdk.Msg{treasurySend()}, msgs)
}
