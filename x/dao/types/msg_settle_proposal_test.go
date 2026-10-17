package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgSettleProposal_ValidateBasic(t *testing.T) {
	sender := sdk.AccAddress("dao_settler_________").String()

	for _, tc := range []struct {
		desc       string
		sender     string
		proposalId uint64
		wantErr    error
	}{
		{desc: "valid", sender: sender, proposalId: 1},
		{desc: "invalid sender", sender: "bze1nope", proposalId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty sender", sender: "", proposalId: 1, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero proposal id", sender: sender, proposalId: 0, wantErr: types.ErrProposalNotFound},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := types.NewMsgSettleProposal(tc.sender, tc.proposalId).ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
