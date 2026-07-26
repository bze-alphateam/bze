package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgSetDenomBranding_ValidateBasic(t *testing.T) {
	denom := "factory/" + sample.AccAddress() + "/token"
	tests := []struct {
		name string
		msg  MsgSetDenomBranding
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgSetDenomBranding{
				Creator:  "invalid_address",
				Denom:    denom,
				Branding: validBranding(),
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty denom",
			msg: MsgSetDenomBranding{
				Creator:  sample.AccAddress(),
				Denom:    "",
				Branding: validBranding(),
			},
			err: ErrInvalidDenom,
		},
		{
			name: "nil branding is a valid clear request",
			msg: MsgSetDenomBranding{
				Creator: sample.AccAddress(),
				Denom:   denom,
			},
		},
		{
			name: "empty branding is a valid clear request",
			msg: MsgSetDenomBranding{
				Creator:  sample.AccAddress(),
				Denom:    denom,
				Branding: &DenomBranding{},
			},
		},
		{
			name: "partial branding is rejected",
			msg: MsgSetDenomBranding{
				Creator:  sample.AccAddress(),
				Denom:    denom,
				Branding: &DenomBranding{Font: "inter"},
			},
			err: ErrInvalidBranding,
		},
		{
			name: "invalid colour is rejected",
			msg: MsgSetDenomBranding{
				Creator: sample.AccAddress(),
				Denom:   denom,
				Branding: func() *DenomBranding {
					b := validBranding()
					b.Light.Primary = "blue"
					return b
				}(),
			},
			err: ErrInvalidBranding,
		},
		{
			name: "invalid font slug is rejected",
			msg: MsgSetDenomBranding{
				Creator: sample.AccAddress(),
				Denom:   denom,
				Branding: func() *DenomBranding {
					b := validBranding()
					b.Font = "Comic Sans"
					return b
				}(),
			},
			err: ErrInvalidBranding,
		},
		{
			name: "valid full package",
			msg: MsgSetDenomBranding{
				Creator:  sample.AccAddress(),
				Denom:    denom,
				Branding: validBranding(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
