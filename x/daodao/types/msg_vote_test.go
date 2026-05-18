package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func TestMsgVote_ValidateBasic(t *testing.T) {
	voter := sample.AccAddress()

	cases := []struct {
		name    string
		msg     types.MsgVote
		wantErr bool
	}{
		{
			name:    "valid: YES",
			msg:     types.MsgVote{Voter: voter, DaoId: 1, ProposalId: 1, Option: types.VoteOption_VOTE_OPTION_YES},
			wantErr: false,
		},
		{
			name:    "valid: NO",
			msg:     types.MsgVote{Voter: voter, DaoId: 1, ProposalId: 1, Option: types.VoteOption_VOTE_OPTION_NO},
			wantErr: false,
		},
		{
			name:    "valid: ABSTAIN",
			msg:     types.MsgVote{Voter: voter, DaoId: 1, ProposalId: 1, Option: types.VoteOption_VOTE_OPTION_ABSTAIN},
			wantErr: false,
		},
		{
			name:    "invalid voter",
			msg:     types.MsgVote{Voter: "bad", DaoId: 1, ProposalId: 1, Option: types.VoteOption_VOTE_OPTION_YES},
			wantErr: true,
		},
		{
			name:    "dao_id zero",
			msg:     types.MsgVote{Voter: voter, DaoId: 0, ProposalId: 1, Option: types.VoteOption_VOTE_OPTION_YES},
			wantErr: true,
		},
		{
			name:    "proposal_id zero",
			msg:     types.MsgVote{Voter: voter, DaoId: 1, ProposalId: 0, Option: types.VoteOption_VOTE_OPTION_YES},
			wantErr: true,
		},
		{
			name:    "option UNSPECIFIED",
			msg:     types.MsgVote{Voter: voter, DaoId: 1, ProposalId: 1, Option: types.VoteOption_VOTE_OPTION_UNSPECIFIED},
			wantErr: true,
		},
		{
			name:    "option out of range",
			msg:     types.MsgVote{Voter: voter, DaoId: 1, ProposalId: 1, Option: types.VoteOption(99)},
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
