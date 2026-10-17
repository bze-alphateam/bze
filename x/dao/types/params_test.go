package types_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func TestDefaultParams_AreValid(t *testing.T) {
	p := types.DefaultParams()

	require.NoError(t, p.Validate())
	require.Equal(t, uint64(128), p.MaxStaticMembers)
	require.Equal(t, time.Hour, p.MinVotingPeriod)
	require.Equal(t, 90*24*time.Hour, p.MaxVotingPeriod)
	require.Equal(t, sdk.NewInt64Coin("ubze", 50_000_000_000), p.DaoCreationFee)
	require.Equal(t, types.FeeDestinationBurner, p.DaoCreationFeeDestination)
}

func TestParams_Validate(t *testing.T) {
	for _, tc := range []struct {
		desc    string
		mutate  func(p *types.Params)
		wantErr string
	}{
		{desc: "defaults", mutate: func(p *types.Params) {}},
		{desc: "community pool destination", mutate: func(p *types.Params) { p.DaoCreationFeeDestination = types.FeeDestinationCommunityPool }},
		{desc: "zero creation fee is free", mutate: func(p *types.Params) { p.DaoCreationFee = sdk.NewInt64Coin("ubze", 0) }},
		{desc: "min equals max period", mutate: func(p *types.Params) { p.MaxVotingPeriod = p.MinVotingPeriod }},
		{desc: "one static member", mutate: func(p *types.Params) { p.MaxStaticMembers = 1 }},
		{desc: "zero static members", mutate: func(p *types.Params) { p.MaxStaticMembers = 0 }, wantErr: "max_static_members"},
		{desc: "zero min period", mutate: func(p *types.Params) { p.MinVotingPeriod = 0 }, wantErr: "min_voting_period"},
		{desc: "negative min period", mutate: func(p *types.Params) { p.MinVotingPeriod = -time.Second }, wantErr: "min_voting_period"},
		{desc: "max below min", mutate: func(p *types.Params) { p.MaxVotingPeriod = p.MinVotingPeriod - time.Second }, wantErr: "max_voting_period"},
		{desc: "negative fee", mutate: func(p *types.Params) { p.DaoCreationFee = sdk.Coin{Denom: "ubze", Amount: math.NewInt(-1)} }, wantErr: "dao_creation_fee"},
		{desc: "invalid fee denom", mutate: func(p *types.Params) { p.DaoCreationFee = sdk.Coin{Denom: "1x", Amount: math.NewInt(1)} }, wantErr: "dao_creation_fee"},
		{desc: "nil fee amount", mutate: func(p *types.Params) { p.DaoCreationFee = sdk.Coin{Denom: "ubze"} }, wantErr: "dao_creation_fee"},
		{desc: "empty destination", mutate: func(p *types.Params) { p.DaoCreationFeeDestination = "" }, wantErr: "dao_creation_fee_destination"},
		{desc: "unknown destination", mutate: func(p *types.Params) { p.DaoCreationFeeDestination = "treasury" }, wantErr: "dao_creation_fee_destination"},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			p := types.DefaultParams()
			tc.mutate(&p)

			err := p.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
