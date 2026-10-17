package types_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func daoConfig() types.DaoConfig {
	return types.DaoConfig{
		MinQuorum:         math.LegacyMustNewDecFromStr("0.4"),
		Threshold:         math.LegacyMustNewDecFromStr("0.5"),
		VotingPeriod:      72 * time.Hour,
		ExpeditedPeriod:   24 * time.Hour,
		ExpediteQuorum:    math.LegacyMustNewDecFromStr("0.6"),
		ExpediteThreshold: math.LegacyMustNewDecFromStr("0.66"),
	}
}

func TestMsgUpdateDaoConfig_ValidateBasic(t *testing.T) {
	authority := sdk.AccAddress("dao_account_________").String()
	badRatio := daoConfig()
	badRatio.Threshold = math.LegacyZeroDec()
	badPeriods := daoConfig()
	badPeriods.ExpeditedPeriod = badPeriods.VotingPeriod + time.Second

	for _, tc := range []struct {
		desc      string
		authority string
		daoId     uint64
		config    types.DaoConfig
		wantErr   error
	}{
		{desc: "valid", authority: authority, daoId: 1, config: daoConfig()},
		{desc: "invalid authority", authority: "bze1nope", daoId: 1, config: daoConfig(), wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty authority", authority: "", daoId: 1, config: daoConfig(), wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "zero dao id", authority: authority, daoId: 0, config: daoConfig(), wantErr: types.ErrDaoNotFound},
		{desc: "invalid ratio", authority: authority, daoId: 1, config: badRatio, wantErr: types.ErrInvalidConfig},
		{desc: "expedited longer than voting", authority: authority, daoId: 1, config: badPeriods, wantErr: types.ErrInvalidConfig},
		{desc: "unset config", authority: authority, daoId: 1, wantErr: types.ErrInvalidConfig},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := types.NewMsgUpdateDaoConfig(tc.authority, tc.daoId, tc.config).ValidateBasic()
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestDaoConfig_ChangedFields(t *testing.T) {
	coin := func(amount int64) *sdk.Coin {
		c := sdk.NewInt64Coin("ubze", amount)
		return &c
	}

	for _, tc := range []struct {
		desc   string
		mutate func(c *types.DaoConfig)
		want   []string
	}{
		{desc: "identical", mutate: func(c *types.DaoConfig) {}, want: []string{}},
		{desc: "equal decimal", mutate: func(c *types.DaoConfig) { c.MinQuorum = math.LegacyNewDecWithPrec(40, 2) }, want: []string{}},
		{desc: "expedite fields", mutate: func(c *types.DaoConfig) {
			c.ExpediteThreshold = math.LegacyOneDec()
			c.AllowExpedite = true
		}, want: []string{"allow_expedite", "expedite_threshold"}},
		{desc: "cost set", mutate: func(c *types.DaoConfig) { c.ProposalCost = coin(1) }, want: []string{"proposal_cost"}},
		{desc: "zero cost differs from none", mutate: func(c *types.DaoConfig) { c.ProposalCost = coin(0) }, want: []string{"proposal_cost"}},
		{desc: "deposit denom", mutate: func(c *types.DaoConfig) {
			c.ProposalDeposit = &sdk.Coin{Denom: "uvdl", Amount: math.NewInt(5)}
		}, want: []string{"proposal_deposit"}},
		{desc: "policy and destinations", mutate: func(c *types.DaoConfig) {
			c.DepositRefundPolicy = types.DepositRefundPolicy_ONLY_PASSED
			c.CostDestination = types.FundsDestination_TREASURY
		}, want: []string{"deposit_refund_policy", "cost_destination"}},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			next := daoConfig()
			tc.mutate(&next)
			require.Equal(t, tc.want, daoConfig().ChangedFields(next))
		})
	}

	// coins compare by value, not by pointer
	a, b := daoConfig(), daoConfig()
	a.ProposalDeposit, b.ProposalDeposit = coin(5), coin(5)
	require.Empty(t, a.ChangedFields(b))
	b.ProposalDeposit = coin(6)
	require.Equal(t, []string{"proposal_deposit"}, a.ChangedFields(b))
}
