package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

func TestNewMsgSetDenomAdmin(t *testing.T) {
	authority := sample.AccAddress()
	newAdmin := sample.AccAddress()
	denom := "factory/" + authority + "/token"

	msg := types.NewMsgSetDenomAdmin(authority, denom, newAdmin)

	require.Equal(t, authority, msg.Authority)
	require.Equal(t, denom, msg.Denom)
	require.Equal(t, newAdmin, msg.NewAdmin)
	require.Equal(t, "/bze.tokenfactory.MsgSetDenomAdmin", sdk.MsgTypeURL(msg))
}

func TestMsgSetDenomAdmin_ValidateBasic(t *testing.T) {
	authority := sample.AccAddress()
	newAdmin := sample.AccAddress()
	denom := "factory/" + authority + "/token"

	tests := []struct {
		name string
		msg  *types.MsgSetDenomAdmin
		err  error
	}{
		{name: "valid", msg: types.NewMsgSetDenomAdmin(authority, denom, newAdmin)},
		{name: "valid, new admin is the authority", msg: types.NewMsgSetDenomAdmin(authority, denom, authority)},
		{name: "invalid authority", msg: types.NewMsgSetDenomAdmin("invalid_address", denom, newAdmin), err: sdkerrors.ErrInvalidAddress},
		{name: "empty authority", msg: types.NewMsgSetDenomAdmin("", denom, newAdmin), err: sdkerrors.ErrInvalidAddress},
		{name: "empty denom", msg: types.NewMsgSetDenomAdmin(authority, "", newAdmin), err: types.ErrInvalidDenom},
		{name: "malformed denom", msg: types.NewMsgSetDenomAdmin(authority, "1bad denom!", newAdmin), err: types.ErrInvalidDenom},
		{name: "empty new admin", msg: types.NewMsgSetDenomAdmin(authority, denom, ""), err: sdkerrors.ErrInvalidAddress},
		{name: "malformed new admin", msg: types.NewMsgSetDenomAdmin(authority, denom, "invalid_address"), err: sdkerrors.ErrInvalidAddress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgSetDenomAdmin_EmptyNewAdminMessage(t *testing.T) {
	err := types.NewMsgSetDenomAdmin(sample.AccAddress(), "factory/x/token", "").ValidateBasic()
	require.ErrorContains(t, err, "new admin must not be empty")
}
