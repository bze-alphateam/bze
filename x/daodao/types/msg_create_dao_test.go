package types_test

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/testutil/sample"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

func validMetadata() types.DaoMetadata {
	return types.DaoMetadata{Name: "dao"}
}

// staticCfg builds a MsgCreateDao voting_config with a single STATIC member.
// Test-only helper for ValidateBasic table tests.
func staticCfg(addr string) *types.MsgCreateDao_Static {
	return &types.MsgCreateDao_Static{
		Static: &types.StaticVotingConfig{
			Members: []types.StaticMember{{Address: addr, Weight: 1}},
		},
	}
}

// validMsgGov is the GovernanceConfig threaded into "valid" MsgCreateDao
// table cases. Epic 3 makes governance required; tests that previously
// expected ValidateBasic to pass must supply a real config.
func validMsgGov() types.GovernanceConfig {
	return types.GovernanceConfig{
		ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITHOUT_QUORUM,
		ThresholdBps: 5_000,
		QuorumBps:    0,
		VotingPeriod: 24 * time.Hour,
		AllowRevote:  true,
	}
}

// validMsgDeposit is the DepositConfig threaded into "valid" MsgCreateDao
// table cases. Epic 4 makes deposit required.
func validMsgDeposit() types.DepositConfig {
	return types.DepositConfig{
		MinDeposit:         sdk.NewInt64Coin("ubze", 1),
		DepositPeriod:      7 * 24 * time.Hour,
		ForfeitDestination: types.ForfeitDestination_FORFEIT_DESTINATION_TREASURY,
		VotingRefundPolicy: types.RefundPolicy_REFUND_POLICY_ON_PASS,
	}
}

func TestMsgCreateDao_ValidateBasic(t *testing.T) {
	creator := sample.AccAddress()
	admin := sample.AccAddress()

	tests := []struct {
		name    string
		msg     types.MsgCreateDao
		wantErr bool
	}{
		{
			name: "valid: no admin (defaults to creator)",
			msg: types.MsgCreateDao{
				Creator:      creator,
				Metadata:     validMetadata(),
				VotingConfig: staticCfg(creator),
				Governance:   validMsgGov(),
				Deposit:      validMsgDeposit(),
			},
			wantErr: false,
		},
		{
			name: "valid: explicit admin",
			msg: types.MsgCreateDao{
				Creator:      creator,
				Admin:        admin,
				Metadata:     validMetadata(),
				VotingConfig: staticCfg(creator),
				Governance:   validMsgGov(),
				Deposit:      validMsgDeposit(),
			},
			wantErr: false,
		},
		{
			// Epic 3 makes governance required at creation.
			name: "missing governance config is rejected",
			msg: types.MsgCreateDao{
				Creator:      creator,
				Metadata:     validMetadata(),
				VotingConfig: staticCfg(creator),
			},
			wantErr: true,
		},
		{
			// Invalid governance is also rejected. Use a quorum-bps that
			// exceeds the cap.
			name: "invalid governance config is rejected",
			msg: types.MsgCreateDao{
				Creator:      creator,
				Metadata:     validMetadata(),
				VotingConfig: staticCfg(creator),
				Governance: types.GovernanceConfig{
					ApprovalRule: types.ApprovalRule_APPROVAL_RULE_WITH_QUORUM,
					ThresholdBps: 5_000,
					QuorumBps:    9_999, // exceeds MaxQuorumBps = 8500
					VotingPeriod: 24 * time.Hour,
				},
			},
			wantErr: true,
		},
		{
			name: "missing voting_config",
			msg: types.MsgCreateDao{
				Creator:  creator,
				Metadata: validMetadata(),
			},
			wantErr: true,
		},
		{
			name: "reward_staked at creation is rejected",
			msg: types.MsgCreateDao{
				Creator:  creator,
				Metadata: validMetadata(),
				VotingConfig: &types.MsgCreateDao_RewardStaked{
					RewardStaked: &types.RewardStakedVotingConfig{RewardId: "00000000-0000-0000-0000-000000000001"},
				},
			},
			wantErr: true,
		},
		{
			name: "static with empty member list",
			msg: types.MsgCreateDao{
				Creator:      creator,
				Metadata:     validMetadata(),
				VotingConfig: &types.MsgCreateDao_Static{Static: &types.StaticVotingConfig{}},
			},
			wantErr: true,
		},
		{
			name: "static with zero weight",
			msg: types.MsgCreateDao{
				Creator:  creator,
				Metadata: validMetadata(),
				VotingConfig: &types.MsgCreateDao_Static{
					Static: &types.StaticVotingConfig{
						Members: []types.StaticMember{{Address: creator, Weight: 0}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "static with duplicate addresses",
			msg: types.MsgCreateDao{
				Creator:  creator,
				Metadata: validMetadata(),
				VotingConfig: &types.MsgCreateDao_Static{
					Static: &types.StaticVotingConfig{
						Members: []types.StaticMember{
							{Address: creator, Weight: 1},
							{Address: creator, Weight: 2},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "static with bad-bech32 member",
			msg: types.MsgCreateDao{
				Creator:  creator,
				Metadata: validMetadata(),
				VotingConfig: &types.MsgCreateDao_Static{
					Static: &types.StaticVotingConfig{
						Members: []types.StaticMember{{Address: "not-a-bech32", Weight: 1}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid creator bech32",
			msg: types.MsgCreateDao{
				Creator:      "not-a-bech32",
				Metadata:     validMetadata(),
				VotingConfig: staticCfg(creator),
			},
			wantErr: true,
		},
		{
			name: "invalid admin bech32",
			msg: types.MsgCreateDao{
				Creator:      creator,
				Admin:        "not-a-bech32",
				Metadata:     validMetadata(),
				VotingConfig: staticCfg(creator),
			},
			wantErr: true,
		},
		{
			name: "empty name",
			msg: types.MsgCreateDao{
				Creator:      creator,
				Metadata:     types.DaoMetadata{Name: ""},
				VotingConfig: staticCfg(creator),
			},
			wantErr: true,
		},
		{
			name: "name over cap",
			msg: types.MsgCreateDao{
				Creator: creator,
				Metadata: types.DaoMetadata{
					Name: strings.Repeat("x", types.MaxDaoNameLen+1),
				},
			},
			wantErr: true,
		},
		{
			name: "description over cap",
			msg: types.MsgCreateDao{
				Creator: creator,
				Metadata: types.DaoMetadata{
					Name:        "ok",
					Description: strings.Repeat("x", types.MaxDaoDescriptionLen+1),
				},
			},
			wantErr: true,
		},
		{
			name: "image_url over cap",
			msg: types.MsgCreateDao{
				Creator: creator,
				Metadata: types.DaoMetadata{
					Name:     "ok",
					ImageUrl: strings.Repeat("x", types.MaxDaoImageURLLen+1),
				},
			},
			wantErr: true,
		},
		{
			name: "twitter over cap",
			msg: types.MsgCreateDao{
				Creator: creator,
				Metadata: types.DaoMetadata{
					Name:    "ok",
					Twitter: strings.Repeat("x", types.MaxDaoLinkLen+1),
				},
			},
			wantErr: true,
		},
		{
			name: "discord over cap",
			msg: types.MsgCreateDao{
				Creator: creator,
				Metadata: types.DaoMetadata{
					Name:    "ok",
					Discord: strings.Repeat("x", types.MaxDaoLinkLen+1),
				},
			},
			wantErr: true,
		},
		{
			name: "telegram over cap",
			msg: types.MsgCreateDao{
				Creator: creator,
				Metadata: types.DaoMetadata{
					Name:     "ok",
					Telegram: strings.Repeat("x", types.MaxDaoLinkLen+1),
				},
			},
			wantErr: true,
		},
		{
			name: "website over cap",
			msg: types.MsgCreateDao{
				Creator: creator,
				Metadata: types.DaoMetadata{
					Name:    "ok",
					Website: strings.Repeat("x", types.MaxDaoLinkLen+1),
				},
			},
			wantErr: true,
		},
		{
			name: "other over cap",
			msg: types.MsgCreateDao{
				Creator: creator,
				Metadata: types.DaoMetadata{
					Name:  "ok",
					Other: strings.Repeat("x", types.MaxDaoLinkLen+1),
				},
			},
			wantErr: true,
		},
		{
			name: "every field at exactly its cap is valid",
			msg: types.MsgCreateDao{
				Creator: creator,
				Metadata: types.DaoMetadata{
					Name:        strings.Repeat("x", types.MaxDaoNameLen),
					Description: strings.Repeat("x", types.MaxDaoDescriptionLen),
					ImageUrl:    strings.Repeat("x", types.MaxDaoImageURLLen),
					Twitter:     strings.Repeat("x", types.MaxDaoLinkLen),
					Discord:     strings.Repeat("x", types.MaxDaoLinkLen),
					Telegram:    strings.Repeat("x", types.MaxDaoLinkLen),
					Website:     strings.Repeat("x", types.MaxDaoLinkLen),
					Other:       strings.Repeat("x", types.MaxDaoLinkLen),
				},
				VotingConfig: staticCfg(creator),
				Governance:   validMsgGov(),
				Deposit:      validMsgDeposit(),
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
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
