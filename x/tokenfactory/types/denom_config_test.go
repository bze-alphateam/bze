package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

func TestDefaultDenomConfig(t *testing.T) {
	cfg := types.DefaultDenomConfig()
	require.Empty(t, cfg.CommunityPool)
	require.False(t, cfg.TransferFee.IsNil())
	require.False(t, cfg.GasSurcharge.IsNil())
	require.True(t, cfg.TransferFee.IsZero())
	require.True(t, cfg.GasSurcharge.IsZero())
	require.True(t, cfg.IsDefault())
	require.NoError(t, cfg.Validate())
}

func TestDenomConfig_IsDefault(t *testing.T) {
	cp := sample.AccAddress()
	tests := []struct {
		name string
		cfg  types.DenomConfig
		want bool
	}{
		{name: "defaults", cfg: types.DefaultDenomConfig(), want: true},
		{name: "community pool set", cfg: types.DenomConfig{CommunityPool: cp, TransferFee: math.LegacyZeroDec(), GasSurcharge: math.LegacyZeroDec()}},
		{name: "transfer fee set", cfg: types.DenomConfig{TransferFee: math.LegacyNewDecWithPrec(1, 3), GasSurcharge: math.LegacyZeroDec()}},
		{name: "gas surcharge set", cfg: types.DenomConfig{TransferFee: math.LegacyZeroDec(), GasSurcharge: math.LegacyOneDec()}},
		{name: "nil rates are not the defaults", cfg: types.DenomConfig{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.cfg.IsDefault())
		})
	}
}

func TestDenomConfig_Validate(t *testing.T) {
	cp := sample.AccAddress()
	zero := math.LegacyZeroDec()
	tests := []struct {
		name  string
		cfg   types.DenomConfig
		valid bool
	}{
		{name: "defaults", cfg: types.DefaultDenomConfig(), valid: true},
		{name: "full config", cfg: types.DenomConfig{CommunityPool: cp, TransferFee: math.LegacyNewDecWithPrec(1, 3), GasSurcharge: math.LegacyNewDecWithPrec(5, 1)}, valid: true},
		{name: "rates at the caps", cfg: types.DenomConfig{CommunityPool: cp, TransferFee: types.MaxTransferFee, GasSurcharge: types.MaxGasSurcharge}, valid: true},
		{name: "invalid community pool", cfg: types.DenomConfig{CommunityPool: "notanaddress", TransferFee: zero, GasSurcharge: zero}},
		{name: "missing transfer fee", cfg: types.DenomConfig{GasSurcharge: zero}},
		{name: "missing gas surcharge", cfg: types.DenomConfig{TransferFee: zero}},
		{name: "negative transfer fee", cfg: types.DenomConfig{TransferFee: math.LegacyNewDecWithPrec(-1, 3), GasSurcharge: zero}},
		{name: "negative gas surcharge", cfg: types.DenomConfig{TransferFee: zero, GasSurcharge: math.LegacyNewDecWithPrec(-1, 3)}},
		{name: "transfer fee above the cap", cfg: types.DenomConfig{TransferFee: math.LegacyMustNewDecFromStr("0.050000000000000001"), GasSurcharge: zero}},
		{name: "gas surcharge above the cap", cfg: types.DenomConfig{TransferFee: zero, GasSurcharge: math.LegacyMustNewDecFromStr("2.000000000000000001")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, types.ErrInvalidDenomConfig)
		})
	}
}
