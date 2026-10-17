package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgVote_ValidateBasic(t *testing.T) {
	voter := sdk.AccAddress("dao_voter___________").String()

	for _, tc := range []struct {
		desc       string
		voter      string
		proposalId uint64
		option     types.VoteOption
		wantErr    error
	}{
		{desc: "yes", voter: voter, proposalId: 1, option: types.VoteOption_YES},
		{desc: "no", voter: voter, proposalId: 1, option: types.VoteOption_NO},
		{desc: "abstain", voter: voter, proposalId: 1, option: types.VoteOption_ABSTAIN},
		{desc: "invalid voter", voter: "bze1nope", proposalId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty voter", voter: "", proposalId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero proposal id", voter: voter, proposalId: 0, wantErr: types.ErrProposalNotFound},
		{desc: "unknown option", voter: voter, proposalId: 1, option: types.VoteOption(3), wantErr: types.ErrInvalidVote},
		{desc: "negative option", voter: voter, proposalId: 1, option: types.VoteOption(-1), wantErr: types.ErrInvalidVote},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := types.NewMsgVote(tc.voter, tc.proposalId, tc.option).ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
