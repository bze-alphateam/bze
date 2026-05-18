package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func TestMsgUpdateMembers_ValidateBasic(t *testing.T) {
	auth := sample.AccAddress()
	a := sample.AccAddress()
	b := sample.AccAddress()

	tests := []struct {
		name    string
		msg     types.MsgUpdateMembers
		wantErr bool
	}{
		{
			name: "valid: add only",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     1,
				Add:       []types.StaticMember{{Address: a, Weight: 1}},
			},
			wantErr: false,
		},
		{
			name: "valid: remove only",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     1,
				Remove:    []string{a},
			},
			wantErr: false,
		},
		{
			name: "valid: mixed add + remove",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     1,
				Add:       []types.StaticMember{{Address: a, Weight: 1}},
				Remove:    []string{b},
			},
			wantErr: false,
		},
		{
			name: "empty msg (no add, no remove)",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     1,
			},
			wantErr: true,
		},
		{
			name: "bad authority",
			msg: types.MsgUpdateMembers{
				Authority: "not-a-bech32",
				DaoId:     1,
				Add:       []types.StaticMember{{Address: a, Weight: 1}},
			},
			wantErr: true,
		},
		{
			name: "dao_id zero",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     0,
				Add:       []types.StaticMember{{Address: a, Weight: 1}},
			},
			wantErr: true,
		},
		{
			name: "add: bad bech32",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     1,
				Add:       []types.StaticMember{{Address: "not-a-bech32", Weight: 1}},
			},
			wantErr: true,
		},
		{
			name: "add: zero weight",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     1,
				Add:       []types.StaticMember{{Address: a, Weight: 0}},
			},
			wantErr: true,
		},
		{
			name: "add: duplicate addresses",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     1,
				Add: []types.StaticMember{
					{Address: a, Weight: 1},
					{Address: a, Weight: 2},
				},
			},
			wantErr: true,
		},
		{
			name: "remove: bad bech32",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     1,
				Remove:    []string{"not-a-bech32"},
			},
			wantErr: true,
		},
		{
			name: "remove: duplicate addresses",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     1,
				Remove:    []string{a, a},
			},
			wantErr: true,
		},
		{
			name: "add and remove overlap",
			msg: types.MsgUpdateMembers{
				Authority: auth,
				DaoId:     1,
				Add:       []types.StaticMember{{Address: a, Weight: 1}},
				Remove:    []string{a},
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
