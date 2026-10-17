package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgCleanupProposal_ValidateBasic(t *testing.T) {
	sender := sdk.AccAddress("dao_cleaner_________").String()

	for _, tc := range []struct {
		desc       string
		sender     string
		proposalId uint64
		limit      uint64
		wantErr    error
	}{
		{desc: "valid", sender: sender, proposalId: 1, limit: 10},
		{desc: "zero limit means the param", sender: sender, proposalId: 1},
		{desc: "limit above any param", sender: sender, proposalId: 1, limit: ^uint64(0)},
		{desc: "invalid sender", sender: "bze1nope", proposalId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty sender", sender: "", proposalId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero proposal id", sender: sender, proposalId: 0, wantErr: types.ErrProposalNotFound},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := types.NewMsgCleanupProposal(tc.sender, tc.proposalId, tc.limit).ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestProposalStatus_IsFinished(t *testing.T) {
	for status, finished := range map[types.ProposalStatus]bool{
		types.ProposalStatus_VOTING:           false,
		types.ProposalStatus_EXECUTION_FAILED: false,
		types.ProposalStatus_REJECTED:         true,
		types.ProposalStatus_EXECUTED:         true,
		types.ProposalStatus_EXPIRED:          true,
		types.ProposalStatus_CANCELLED:        true,
	} {
		require.Equal(t, finished, status.IsFinished(), status.String())
	}
}
