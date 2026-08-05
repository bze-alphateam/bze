package types

import (
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgCreateBoost(t *testing.T) {
	creator := sample.AccAddress()

	msg := NewMsgCreateBoost(creator, "000000000001", "ubze", "1000", "30")

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, "000000000001", msg.RewardId)
	require.Equal(t, "ubze", msg.Denom)
	require.Equal(t, "1000", msg.DailyAmount)
	require.Equal(t, "30", msg.Days)
}

func TestMsgCreateBoost_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()

	tests := []struct {
		name string
		msg  MsgCreateBoost
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgCreateBoost{
				Creator:     "invalid_address",
				RewardId:    "000000000001",
				Denom:       "ubze",
				DailyAmount: "1000",
				Days:        "30",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty reward_id",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "",
				Denom:       "ubze",
				DailyAmount: "1000",
				Days:        "30",
			},
			err: ErrInvalidRewardId,
		},
		{
			name: "empty denom",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "000000000001",
				Denom:       "",
				DailyAmount: "1000",
				Days:        "30",
			},
			err: ErrInvalidBoostDenom,
		},
		{
			name: "non-numeric daily_amount",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "000000000001",
				Denom:       "ubze",
				DailyAmount: "abc",
				Days:        "30",
			},
			err: ErrInvalidAmount,
		},
		{
			name: "zero daily_amount",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "000000000001",
				Denom:       "ubze",
				DailyAmount: "0",
				Days:        "30",
			},
			err: ErrInvalidAmount,
		},
		{
			name: "negative daily_amount",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "000000000001",
				Denom:       "ubze",
				DailyAmount: "-5",
				Days:        "30",
			},
			err: ErrInvalidAmount,
		},
		{
			name: "non-numeric days",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "000000000001",
				Denom:       "ubze",
				DailyAmount: "1000",
				Days:        "xyz",
			},
			err: ErrInvalidBoostDays,
		},
		{
			name: "zero days",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "000000000001",
				Denom:       "ubze",
				DailyAmount: "1000",
				Days:        "0",
			},
			err: ErrInvalidBoostDays,
		},
		{
			name: "days too large",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "000000000001",
				Denom:       "ubze",
				DailyAmount: "1000",
				Days:        "36501",
			},
			err: ErrInvalidBoostDays,
		},
		{
			name: "valid plain denom",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "000000000001",
				Denom:       "ubze",
				DailyAmount: "1000",
				Days:        "30",
			},
		},
		{
			name: "valid ibc denom",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "000000000001",
				Denom:       "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2",
				DailyAmount: "1000",
				Days:        "30",
			},
		},
		{
			name: "valid factory denom",
			msg: MsgCreateBoost{
				Creator:     validCreator,
				RewardId:    "000000000001",
				Denom:       "factory/bze1abc/sub",
				DailyAmount: "1000",
				Days:        "1",
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
