package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	authority := sample.AccAddress()
	validParams := types.DefaultParams()

	// Sentinel invalid Params variant — same shape we use in the keeper-side
	// "valid authority but invalid params" test.
	invalidParams := types.DefaultParams()
	invalidParams.DaoCreationFeeDestination = "not_a_real_destination"

	tests := []struct {
		name    string
		msg     types.MsgUpdateParams
		wantErr bool
	}{
		{
			name: "valid",
			msg: types.MsgUpdateParams{
				Authority: authority,
				Params:    validParams,
			},
			wantErr: false,
		},
		{
			name: "invalid authority bech32",
			msg: types.MsgUpdateParams{
				Authority: "not-a-bech32",
				Params:    validParams,
			},
			wantErr: true,
		},
		{
			name: "valid authority but invalid params",
			msg: types.MsgUpdateParams{
				Authority: authority,
				Params:    invalidParams,
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
