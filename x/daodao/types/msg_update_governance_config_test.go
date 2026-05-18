package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func validGovForMsg() types.GovernanceConfig {
	return types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
		ThresholdBps: 5_000,
		VotingPeriod: time.Hour,
	}
}

func TestMsgUpdateGovernanceConfig_ValidateBasic(t *testing.T) {
	auth := sample.AccAddress()

	cases := []struct {
		name    string
		msg     types.MsgUpdateGovernanceConfig
		wantErr bool
	}{
		{
			name:    "valid",
			msg:     types.MsgUpdateGovernanceConfig{Authority: auth, DaoId: 1, Governance: validGovForMsg()},
			wantErr: false,
		},
		{
			name:    "bad authority",
			msg:     types.MsgUpdateGovernanceConfig{Authority: "not-bech32", DaoId: 1, Governance: validGovForMsg()},
			wantErr: true,
		},
		{
			name:    "dao_id zero",
			msg:     types.MsgUpdateGovernanceConfig{Authority: auth, DaoId: 0, Governance: validGovForMsg()},
			wantErr: true,
		},
		{
			name: "threshold over cap",
			msg: types.MsgUpdateGovernanceConfig{Authority: auth, DaoId: 1, Governance: types.GovernanceConfig{
				ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
				ThresholdBps: 9_999,
				VotingPeriod: time.Hour,
			}},
			wantErr: true,
		},
		{
			name: "WITHOUT_QUORUM with non-zero quorum",
			msg: types.MsgUpdateGovernanceConfig{Authority: auth, DaoId: 1, Governance: types.GovernanceConfig{
				ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
				ThresholdBps: 5_000,
				QuorumBps:    1,
				VotingPeriod: time.Hour,
			}},
			wantErr: true,
		},
		{
			name: "voting_period below floor",
			msg: types.MsgUpdateGovernanceConfig{Authority: auth, DaoId: 1, Governance: types.GovernanceConfig{
				ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
				ThresholdBps: 5_000,
				VotingPeriod: 30 * time.Minute,
			}},
			wantErr: true,
		},
	}
	for _, tc := range cases {
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
