package types_test

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgWithdrawAdminNomination_ValidateBasic(t *testing.T) {
	for _, tc := range []struct {
		desc      string
		authority string
		daoId     uint64
		wantErr   error
	}{
		{desc: "valid", authority: execAdmin.String(), daoId: 1},
		{desc: "invalid authority", authority: "bze1nope", daoId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty authority", daoId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero dao id", authority: execAdmin.String(), wantErr: types.ErrDaoNotFound},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := types.NewMsgWithdrawAdminNomination(tc.authority, tc.daoId).ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
