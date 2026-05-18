package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func TestMsgDeposit_ValidateBasic(t *testing.T) {
	depositor := sample.AccAddress()

	cases := []struct {
		name    string
		msg     types.MsgDeposit
		wantErr bool
	}{
		{
			name:    "valid",
			msg:     types.MsgDeposit{Depositor: depositor, DaoId: 1, ProposalId: 1, Amount: sdk.NewInt64Coin("ubze", 1)},
			wantErr: false,
		},
		{
			name:    "invalid depositor",
			msg:     types.MsgDeposit{Depositor: "bad", DaoId: 1, ProposalId: 1, Amount: sdk.NewInt64Coin("ubze", 1)},
			wantErr: true,
		},
		{
			name:    "dao_id zero",
			msg:     types.MsgDeposit{Depositor: depositor, DaoId: 0, ProposalId: 1, Amount: sdk.NewInt64Coin("ubze", 1)},
			wantErr: true,
		},
		{
			name:    "proposal_id zero",
			msg:     types.MsgDeposit{Depositor: depositor, DaoId: 1, ProposalId: 0, Amount: sdk.NewInt64Coin("ubze", 1)},
			wantErr: true,
		},
		{
			name:    "zero amount",
			msg:     types.MsgDeposit{Depositor: depositor, DaoId: 1, ProposalId: 1, Amount: sdk.NewInt64Coin("ubze", 0)},
			wantErr: true,
		},
		{
			name:    "empty denom",
			msg:     types.MsgDeposit{Depositor: depositor, DaoId: 1, ProposalId: 1, Amount: sdk.Coin{}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
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
