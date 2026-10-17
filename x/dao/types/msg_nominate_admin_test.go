package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgNominateAdmin_ValidateBasic(t *testing.T) {
	nominee := sdk.AccAddress("dao_successor_______").String()

	for _, tc := range []struct {
		desc      string
		authority string
		daoId     uint64
		nominee   string
		wantErr   error
	}{
		{desc: "valid", authority: execAdmin.String(), daoId: 1, nominee: nominee},
		{desc: "empty nominee hands the dao to its voters", authority: execAdmin.String(), daoId: 1},
		{desc: "invalid authority", authority: "bze1nope", daoId: 1, nominee: nominee, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty authority", daoId: 1, nominee: nominee, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero dao id", authority: execAdmin.String(), nominee: nominee, wantErr: types.ErrDaoNotFound},
		{desc: "invalid nominee", authority: execAdmin.String(), daoId: 1, nominee: "bze1nope", wantErr: types.ErrInvalidNominee},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := types.NewMsgNominateAdmin(tc.authority, tc.daoId, tc.nominee).ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
