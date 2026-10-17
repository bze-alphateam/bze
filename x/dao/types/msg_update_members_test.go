package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgUpdateMembers_ValidateBasic(t *testing.T) {
	authority := sdk.AccAddress("dao_authority_______").String()
	a := sdk.AccAddress("dao_member_a________").String()
	b := sdk.AccAddress("dao_member_b________").String()
	c := sdk.AccAddress("dao_member_c________").String()

	for _, tc := range []struct {
		desc    string
		msg     *types.MsgUpdateMembers
		wantErr error
		errText string
	}{
		{desc: "add and remove", msg: types.NewMsgUpdateMembers(authority, 1, []string{a, b}, []string{c})},
		{desc: "add only", msg: types.NewMsgUpdateMembers(authority, 1, []string{a}, nil)},
		{desc: "remove only", msg: types.NewMsgUpdateMembers(authority, 1, nil, []string{a})},
		{desc: "bad authority", msg: types.NewMsgUpdateMembers("bad", 1, []string{a}, nil), wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero dao id", msg: types.NewMsgUpdateMembers(authority, 0, []string{a}, nil), wantErr: types.ErrDaoNotFound},
		{desc: "nothing to change", msg: types.NewMsgUpdateMembers(authority, 1, nil, nil), wantErr: types.ErrInvalidMembers, errText: "nothing to add or remove"},
		{desc: "invalid added address", msg: types.NewMsgUpdateMembers(authority, 1, []string{a, "bze1nope"}, nil), wantErr: types.ErrInvalidMembers, errText: "invalid add address bze1nope"},
		{desc: "invalid removed address", msg: types.NewMsgUpdateMembers(authority, 1, nil, []string{""}), wantErr: types.ErrInvalidMembers, errText: "invalid remove address"},
		{desc: "duplicate added address", msg: types.NewMsgUpdateMembers(authority, 1, []string{a, b, a}, nil), wantErr: types.ErrInvalidMembers, errText: "duplicate add address"},
		{desc: "duplicate removed address in another case", msg: types.NewMsgUpdateMembers(authority, 1, nil, []string{a, strings.ToUpper(a)}), wantErr: types.ErrInvalidMembers, errText: "duplicate remove address"},
		{desc: "address both added and removed", msg: types.NewMsgUpdateMembers(authority, 1, []string{a, b}, []string{strings.ToUpper(b)}), wantErr: types.ErrInvalidMembers, errText: "is both added and removed"},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
			require.ErrorContains(t, err, tc.errText)
		})
	}
}
