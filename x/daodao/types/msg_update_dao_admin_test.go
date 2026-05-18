package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func TestMsgUpdateDaoAdmin_ValidateBasic(t *testing.T) {
	authority := sample.AccAddress()
	newAdmin := sample.AccAddress()

	tests := []struct {
		name    string
		msg     types.MsgUpdateDaoAdmin
		wantErr bool
	}{
		{
			name: "valid",
			msg: types.MsgUpdateDaoAdmin{
				Authority: authority,
				DaoId:     1,
				NewAdmin:  newAdmin,
			},
			wantErr: false,
		},
		{
			name: "invalid authority bech32",
			msg: types.MsgUpdateDaoAdmin{
				Authority: "not-a-bech32",
				DaoId:     1,
				NewAdmin:  newAdmin,
			},
			wantErr: true,
		},
		{
			name: "invalid new_admin bech32",
			msg: types.MsgUpdateDaoAdmin{
				Authority: authority,
				DaoId:     1,
				NewAdmin:  "not-a-bech32",
			},
			wantErr: true,
		},
		{
			name: "new_admin equals authority (self-nomination rejected)",
			msg: types.MsgUpdateDaoAdmin{
				Authority: authority,
				DaoId:     1,
				NewAdmin:  authority,
			},
			wantErr: true,
		},
		{
			name: "dao_id zero",
			msg: types.MsgUpdateDaoAdmin{
				Authority: authority,
				DaoId:     0,
				NewAdmin:  newAdmin,
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
