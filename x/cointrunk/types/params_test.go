package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
)

func TestParams_Validate_PublisherRespectTaxBounds(t *testing.T) {
	validCoin := sdk.NewInt64Coin("ubze", 1000)

	tests := []struct {
		desc  string
		tax   math.LegacyDec
		valid bool
	}{
		{
			desc:  "valid tax - 20%",
			tax:   math.LegacyMustNewDecFromStr("0.2"),
			valid: true,
		},
		{
			desc:  "valid tax - smallest positive",
			tax:   math.LegacyMustNewDecFromStr("0.000000000000000001"),
			valid: true,
		},
		{
			desc:  "valid tax - just under 1",
			tax:   math.LegacyMustNewDecFromStr("0.999999999999999999"),
			valid: true,
		},
		{
			desc:  "invalid tax - zero",
			tax:   math.LegacyZeroDec(),
			valid: false,
		},
		{
			desc:  "invalid tax - negative",
			tax:   math.LegacyMustNewDecFromStr("-0.1"),
			valid: false,
		},
		{
			desc:  "invalid tax - exactly 1",
			tax:   math.LegacyOneDec(),
			valid: false,
		},
		{
			desc:  "invalid tax - greater than 1",
			tax:   math.LegacyMustNewDecFromStr("1.5"),
			valid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			params := types.NewParams(
				5,
				validCoin,
				types.PublisherRespectParams{
					Tax:   tc.tax,
					Denom: "ubze",
				},
			)

			err := params.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
