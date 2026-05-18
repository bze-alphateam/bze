package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func TestMsgAcceptDaoAdmin_ValidateBasic(t *testing.T) {
	newAdmin := sample.AccAddress()

	tests := []struct {
		name    string
		msg     types.MsgAcceptDaoAdmin
		wantErr bool
	}{
		{
			name: "valid",
			msg: types.MsgAcceptDaoAdmin{
				NewAdmin: newAdmin,
				DaoId:    1,
			},
			wantErr: false,
		},
		{
			name: "invalid new_admin bech32",
			msg: types.MsgAcceptDaoAdmin{
				NewAdmin: "not-a-bech32",
				DaoId:    1,
			},
			wantErr: true,
		},
		{
			name: "dao_id zero",
			msg: types.MsgAcceptDaoAdmin{
				NewAdmin: newAdmin,
				DaoId:    0,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
