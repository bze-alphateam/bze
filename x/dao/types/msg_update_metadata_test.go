package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgUpdateMetadata_ValidateBasic(t *testing.T) {
	authority := sdk.AccAddress("dao_authority_______").String()

	for _, tc := range []struct {
		desc    string
		msg     *types.MsgUpdateMetadata
		wantErr error
	}{
		{desc: "valid", msg: types.NewMsgUpdateMetadata(authority, 1, "Bee Guild", "pays for hives")},
		{desc: "empty description", msg: types.NewMsgUpdateMetadata(authority, 1, "Bee Guild", "")},
		{desc: "longest name and description", msg: types.NewMsgUpdateMetadata(authority, 1, strings.Repeat("n", types.MaxNameLength), strings.Repeat("d", types.MaxDescriptionLength))},
		{desc: "bad authority", msg: types.NewMsgUpdateMetadata("bad", 1, "Bee Guild", ""), wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero dao id", msg: types.NewMsgUpdateMetadata(authority, 0, "Bee Guild", ""), wantErr: types.ErrDaoNotFound},
		{desc: "empty name", msg: types.NewMsgUpdateMetadata(authority, 1, "", ""), wantErr: types.ErrInvalidMetadata},
		{desc: "name too long", msg: types.NewMsgUpdateMetadata(authority, 1, strings.Repeat("n", types.MaxNameLength+1), ""), wantErr: types.ErrInvalidMetadata},
		{desc: "description too long", msg: types.NewMsgUpdateMetadata(authority, 1, "Bee Guild", strings.Repeat("d", types.MaxDescriptionLength+1)), wantErr: types.ErrInvalidMetadata},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
