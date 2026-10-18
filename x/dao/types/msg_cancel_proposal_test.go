package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgCancelProposal_ValidateBasic(t *testing.T) {
	proposer := sdk.AccAddress("dao_proposer________").String()

	for _, tc := range []struct {
		desc       string
		proposer   string
		proposalId uint64
		wantErr    error
	}{
		{desc: "valid", proposer: proposer, proposalId: 1},
		{desc: "invalid proposer", proposer: "bze1nope", proposalId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty proposer", proposer: "", proposalId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero proposal id", proposer: proposer, proposalId: 0, wantErr: types.ErrProposalNotFound},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := types.NewMsgCancelProposal(tc.proposer, tc.proposalId).ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
