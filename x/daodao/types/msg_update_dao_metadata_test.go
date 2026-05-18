package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func TestMsgUpdateDaoMetadata_ValidateBasic(t *testing.T) {
	authority := sample.AccAddress()

	tests := []struct {
		name    string
		msg     types.MsgUpdateDaoMetadata
		wantErr bool
	}{
		{
			name: "valid",
			msg: types.MsgUpdateDaoMetadata{
				Authority: authority,
				DaoId:     1,
				Metadata:  validMetadata(),
			},
			wantErr: false,
		},
		{
			name: "invalid authority bech32",
			msg: types.MsgUpdateDaoMetadata{
				Authority: "not-a-bech32",
				DaoId:     1,
				Metadata:  validMetadata(),
			},
			wantErr: true,
		},
		{
			name: "dao_id zero",
			msg: types.MsgUpdateDaoMetadata{
				Authority: authority,
				DaoId:     0,
				Metadata:  validMetadata(),
			},
			wantErr: true,
		},
		{
			name: "empty name in metadata",
			msg: types.MsgUpdateDaoMetadata{
				Authority: authority,
				DaoId:     1,
				Metadata:  types.DaoMetadata{Name: ""},
			},
			wantErr: true,
		},
		{
			name: "metadata description over cap",
			msg: types.MsgUpdateDaoMetadata{
				Authority: authority,
				DaoId:     1,
				Metadata: types.DaoMetadata{
					Name:        "ok",
					Description: strings.Repeat("x", types.MaxDaoDescriptionLen+1),
				},
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
