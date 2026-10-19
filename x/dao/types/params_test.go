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
	require.Equal(t, uint64(5), p.MaxSrPerDao)
	require.Equal(t, uint32(5), p.MinElectorateLockDays)
	require.Equal(t, uint64(128), p.MaxStaticMembers)
	require.Equal(t, uint64(16), p.MaxMsgsPerProposal)
	require.Equal(t, uint64(20_000), p.MaxProposalBytes)
	require.Equal(t, uint64(10), p.MaxOpenProposals)
	require.Equal(t, uint64(140), p.MaxTitleLen)
	require.Equal(t, uint64(5_000), p.MaxSummaryLen)
	require.Equal(t, time.Hour, p.MinVotingPeriod)
	require.Equal(t, 90*24*time.Hour, p.MaxVotingPeriod)
	require.Equal(t, 14*24*time.Hour, p.ExecutionWindow)
	require.Equal(t, uint64(100), p.CleanupBatchSize)
	require.Equal(t, sdk.NewInt64Coin("ubze", 50_000_000_000), p.DaoCreationFee)
	require.Equal(t, types.FeeDestinationBurner, p.DaoCreationFeeDestination)
	require.Equal(t, 24*time.Hour, p.MinExpediteDelay)
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
		{desc: "one staking reward per dao", mutate: func(p *types.Params) { p.MaxSrPerDao = 1 }},
		{desc: "zero staking rewards per dao", mutate: func(p *types.Params) { p.MaxSrPerDao = 0 }, wantErr: "max_sr_per_dao"},
		{desc: "one day min lock", mutate: func(p *types.Params) { p.MinElectorateLockDays = 1 }},
		{desc: "zero min lock", mutate: func(p *types.Params) { p.MinElectorateLockDays = 0 }, wantErr: "min_electorate_lock_days"},
		{desc: "one msg per proposal", mutate: func(p *types.Params) { p.MaxMsgsPerProposal = 1 }},
		{desc: "zero msgs per proposal", mutate: func(p *types.Params) { p.MaxMsgsPerProposal = 0 }, wantErr: "max_msgs_per_proposal"},
		{desc: "one proposal byte", mutate: func(p *types.Params) { p.MaxProposalBytes = 1 }},
		{desc: "zero proposal bytes", mutate: func(p *types.Params) { p.MaxProposalBytes = 0 }, wantErr: "max_proposal_bytes"},
		{desc: "one open proposal", mutate: func(p *types.Params) { p.MaxOpenProposals = 1 }},
		{desc: "zero open proposals", mutate: func(p *types.Params) { p.MaxOpenProposals = 0 }, wantErr: "max_open_proposals"},
		{desc: "one title byte", mutate: func(p *types.Params) { p.MaxTitleLen = 1 }},
		{desc: "zero title bytes", mutate: func(p *types.Params) { p.MaxTitleLen = 0 }, wantErr: "max_title_len"},
		{desc: "zero summary bytes", mutate: func(p *types.Params) { p.MaxSummaryLen = 0 }},
		{desc: "zero execution window", mutate: func(p *types.Params) { p.ExecutionWindow = 0 }},
		{desc: "negative execution window", mutate: func(p *types.Params) { p.ExecutionWindow = -time.Second }, wantErr: "execution_window"},
		{desc: "one ballot per cleanup", mutate: func(p *types.Params) { p.CleanupBatchSize = 1 }},
		{desc: "zero cleanup batch", mutate: func(p *types.Params) { p.CleanupBatchSize = 0 }, wantErr: "cleanup_batch_size"},
		{desc: "zero expedite delay", mutate: func(p *types.Params) { p.MinExpediteDelay = 0 }},
		{desc: "negative expedite delay", mutate: func(p *types.Params) { p.MinExpediteDelay = -time.Second }, wantErr: "min_expedite_delay"},
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
