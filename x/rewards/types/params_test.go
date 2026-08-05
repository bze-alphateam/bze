package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestDefaultParams_Boost(t *testing.T) {
	p := DefaultParams()

	require.Equal(t, DefaultCreateBoostFee, p.CreateBoostFee)
	require.Equal(t, DefaultMaxBoostsPerReward, p.MaxBoostsPerReward)
	require.Equal(t, uint32(10), p.MaxBoostsPerReward)
	require.NoError(t, p.Validate())
}

func TestParams_Validate_Boost(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(p *Params)
		wantErr bool
	}{
		{
			name:    "defaults are valid",
			mutate:  func(p *Params) {},
			wantErr: false,
		},
		{
			name:    "invalid create boost fee (empty denom)",
			mutate:  func(p *Params) { p.CreateBoostFee = sdk.Coin{Denom: "", Amount: math.NewInt(1)} },
			wantErr: true,
		},
		{
			name:    "invalid create boost fee (negative amount)",
			mutate:  func(p *Params) { p.CreateBoostFee = sdk.Coin{Denom: "ubze", Amount: math.NewInt(-1)} },
			wantErr: true,
		},
		{
			name:    "zero max boosts per reward",
			mutate:  func(p *Params) { p.MaxBoostsPerReward = 0 },
			wantErr: true,
		},
		{
			name:    "custom valid boost params",
			mutate:  func(p *Params) { p.CreateBoostFee = sdk.NewInt64Coin("ubze", 1); p.MaxBoostsPerReward = 1 },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := DefaultParams()
			tt.mutate(&p)
			err := p.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
