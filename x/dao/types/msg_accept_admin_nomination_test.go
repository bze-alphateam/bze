package types_test

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgAcceptAdminNomination_ValidateBasic(t *testing.T) {
	for _, tc := range []struct {
		desc    string
		nominee string
		daoId   uint64
		wantErr error
	}{
		{desc: "valid", nominee: execAdmin.String(), daoId: 1},
		{desc: "invalid nominee", nominee: "bze1nope", daoId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty nominee", daoId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero dao id", nominee: execAdmin.String(), wantErr: types.ErrDaoNotFound},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := types.NewMsgAcceptAdminNomination(tc.nominee, tc.daoId).ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
