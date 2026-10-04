package types_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

func TestMsgHaltDenoms_ValidateBasic(t *testing.T) {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	tooMany := make([]string, types.MaxDenomsPerHaltMsg+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("udenom%d", i)
	}

	tests := []struct {
		name      string
		authority string
		denoms    []string
		wantErr   error
	}{
		{"valid", authority, []string{"uusdc", "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"}, nil},
		{"any bech32 authority passes the stateless check", sdk.AccAddress("someone____________").String(), []string{"uusdc"}, nil},
		{"max denoms", authority, tooMany[:types.MaxDenomsPerHaltMsg], nil},
		{"invalid authority", "not-an-address", []string{"uusdc"}, nil},
		{"no denoms", authority, nil, types.ErrInvalidDenom},
		{"duplicate", authority, []string{"uusdc", "uusdc"}, types.ErrInvalidDenom},
		{"malformed denom", authority, []string{"1bad"}, types.ErrInvalidDenom},
		{"too many", authority, tooMany, types.ErrInvalidDenom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			halt := types.NewMsgHaltDenoms(tc.authority, tc.denoms)
			unhalt := types.NewMsgUnhaltDenoms(tc.authority, tc.denoms)
			for _, msg := range []sdk.HasValidateBasic{halt, unhalt} {
				err := msg.ValidateBasic()
				switch {
				case tc.name == "invalid authority":
					require.ErrorContains(t, err, "invalid authority address")
				case tc.wantErr == nil:
					require.NoError(t, err)
				default:
					require.ErrorIs(t, err, tc.wantErr)
				}
			}
		})
	}
}
