package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	for _, tc := range []struct {
		desc      string
		authority string
		wantErr   bool
	}{
		{desc: "valid authority", authority: sample.AccAddress()},
		{desc: "empty authority", authority: "", wantErr: true},
		{desc: "malformed authority", authority: "invalid_address", wantErr: true},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			msg := types.MsgUpdateParams{Authority: tc.authority, Params: types.DefaultParams()}
			err := msg.ValidateBasic()
			if tc.wantErr {
				require.ErrorContains(t, err, "invalid authority address")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgUpdateParams_ValidateBasic_RejectsInvalidParams(t *testing.T) {
	params := types.DefaultParams()
	params.DaoCreationFeeDestination = "nowhere"

	msg := types.MsgUpdateParams{Authority: sample.AccAddress(), Params: params}

	require.ErrorContains(t, msg.ValidateBasic(), "dao_creation_fee_destination")
}
