package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/guardian/types"
)

func TestMsgHaltChain_ValidateBasic(t *testing.T) {
	signer := sdk.AccAddress("signer______________").String()
	for _, tc := range []struct {
		name    string
		msg     types.MsgHaltChain
		wantErr error
	}{
		{"valid", types.MsgHaltChain{Signer: signer, Reason: "incident in progress"}, nil},
		{"longest reason", types.MsgHaltChain{Signer: signer, Reason: strings.Repeat("a", types.MaxHaltReasonLength)}, nil},
		{"empty reason", types.MsgHaltChain{Signer: signer}, types.ErrInvalidHaltReason},
		{"reason too long", types.MsgHaltChain{Signer: signer, Reason: strings.Repeat("a", types.MaxHaltReasonLength+1)}, types.ErrInvalidHaltReason},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}

	require.Error(t, (&types.MsgHaltChain{Signer: "not-an-address", Reason: "x"}).ValidateBasic())
	require.Error(t, (&types.MsgHaltChain{Reason: "x"}).ValidateBasic())
}
