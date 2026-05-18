package types_test

import (
	"strings"
	"testing"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// validProposalDeposit is the InitialDeposit threaded through "valid"
// table-test cases. After Epic 4 InitialDeposit is required to be a valid
// Coin (Validate rejects empty denom). Zero amount with non-empty denom is
// valid at the message layer; the keeper enforces non-member vs member
// amount rules statefully.
func validProposalDeposit() sdk.Coin {
	return sdk.NewInt64Coin("ubze", 0)
}

func TestMsgCreateProposal_ValidateBasic(t *testing.T) {
	proposer := sample.AccAddress()

	cases := []struct {
		name    string
		msg     types.MsgCreateProposal
		wantErr bool
	}{
		{
			name: "valid: minimal proposal",
			msg: types.MsgCreateProposal{
				Proposer: proposer, DaoId: 1, Title: "t",
				InitialDeposit: validProposalDeposit(),
			},
			wantErr: false,
		},
		{
			name: "valid: title at cap",
			msg: types.MsgCreateProposal{
				Proposer: proposer, DaoId: 1, Title: strings.Repeat("x", types.MaxProposalTitleLen),
				InitialDeposit: validProposalDeposit(),
			},
			wantErr: false,
		},
		{
			name: "valid: description at cap",
			msg: types.MsgCreateProposal{
				Proposer: proposer, DaoId: 1, Title: "t",
				Description:    strings.Repeat("x", types.MaxProposalDescriptionLen),
				InitialDeposit: validProposalDeposit(),
			},
			wantErr: false,
		},
		{
			name: "invalid proposer bech32",
			msg: types.MsgCreateProposal{
				Proposer: "not-bech32", DaoId: 1, Title: "t",
				InitialDeposit: validProposalDeposit(),
			},
			wantErr: true,
		},
		{
			name: "dao_id zero",
			msg: types.MsgCreateProposal{
				Proposer: proposer, DaoId: 0, Title: "t",
				InitialDeposit: validProposalDeposit(),
			},
			wantErr: true,
		},
		{
			name: "empty title",
			msg: types.MsgCreateProposal{
				Proposer: proposer, DaoId: 1, Title: "",
				InitialDeposit: validProposalDeposit(),
			},
			wantErr: true,
		},
		{
			name: "title over cap",
			msg: types.MsgCreateProposal{
				Proposer: proposer, DaoId: 1, Title: strings.Repeat("x", types.MaxProposalTitleLen+1),
				InitialDeposit: validProposalDeposit(),
			},
			wantErr: true,
		},
		{
			name: "description over cap",
			msg: types.MsgCreateProposal{
				Proposer: proposer, DaoId: 1, Title: "t",
				Description:    strings.Repeat("x", types.MaxProposalDescriptionLen+1),
				InitialDeposit: validProposalDeposit(),
			},
			wantErr: true,
		},
		{
			name: "nil msgs entry rejected",
			msg: types.MsgCreateProposal{
				Proposer: proposer, DaoId: 1, Title: "t",
				Msgs:           []*cdctypes.Any{nil},
				InitialDeposit: validProposalDeposit(),
			},
			wantErr: true,
		},
		{
			// Epic 4: InitialDeposit must be a valid Coin. Empty denom
			// (zero-value Coin) is rejected.
			name: "initial_deposit empty denom rejected",
			msg: types.MsgCreateProposal{
				Proposer: proposer, DaoId: 1, Title: "t",
				// InitialDeposit intentionally left as zero-value Coin.
			},
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
