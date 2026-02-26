package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestNewMsgCreateStakingReward(t *testing.T) {
	creator := sample.AccAddress()
	prizeAmount := math.NewInt(1000)
	prizeDenom := "utoken"
	stakingDenom := "ustake"
	duration := "30"
	minStake := math.NewInt(100)
	lock := "7"

	msg := NewMsgCreateStakingReward(creator, prizeAmount, prizeDenom, stakingDenom, duration, minStake, lock)

	require.Equal(t, creator, msg.Creator)
	require.Equal(t, prizeAmount, msg.PrizeAmount)
	require.Equal(t, prizeDenom, msg.PrizeDenom)
	require.Equal(t, stakingDenom, msg.StakingDenom)
	require.Equal(t, duration, msg.Duration)
	require.Equal(t, minStake, msg.MinStake)
	require.Equal(t, lock, msg.Lock)
}

func TestMsgCreateStakingReward_ValidateBasic(t *testing.T) {
	validCreator := sample.AccAddress()
	validPrizeAmount := math.NewInt(1000)
	validPrizeDenom := "utoken"
	validStakingDenom := "ustake"
	validDuration := "30"
	validMinStake := math.NewInt(100)
	validLock := "7"

	tests := []struct {
		name string
		msg  MsgCreateStakingReward
		err  error
	}{
		{
			name: "invalid creator address",
			msg: MsgCreateStakingReward{
				Creator:      "invalid_address",
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty creator address",
			msg: MsgCreateStakingReward{
				Creator:      "",
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid prize amount - negative",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  math.NewInt(-100),
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: ErrInvalidAmount,
		},
		{
			name: "invalid prize amount - zero",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  math.ZeroInt(),
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: ErrInvalidAmount,
		},
		{
			name: "invalid prize denom - empty",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   "",
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: ErrInvalidPrizeDenom,
		},
		{
			name: "invalid staking denom - empty",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: "",
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: ErrInvalidStakingDenom,
		},
		{
			name: "invalid min stake - negative",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     math.NewInt(-100),
				Lock:         validLock,
			},
			err: ErrInvalidMinStake,
		},
		{
			name: "invalid duration - empty",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     "",
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - not a number",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     "invalid",
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - zero",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     "0",
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - negative",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     "-10",
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid duration - above maximum",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     "36501",
				MinStake:     validMinStake,
				Lock:         validLock,
			},
			err: ErrInvalidDuration,
		},
		{
			name: "invalid lock - empty",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         "",
			},
			err: ErrInvalidLockingTime,
		},
		{
			name: "invalid lock - not a number",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         "invalid",
			},
			err: ErrInvalidLockingTime,
		},
		{
			name: "invalid lock - negative",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         "-10",
			},
			err: ErrInvalidLockingTime,
		},
		{
			name: "invalid lock - above maximum",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         "3651",
			},
			err: ErrInvalidLockingTime,
		},
		{
			name: "valid message - typical values",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   validPrizeDenom,
				StakingDenom: validStakingDenom,
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         validLock,
			},
		},
		{
			name: "valid message - minimum values",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  math.NewInt(1),
				PrizeDenom:   "u",
				StakingDenom: "s",
				Duration:     "1",
				MinStake:     math.ZeroInt(),
				Lock:         "0",
			},
		},
		{
			name: "valid message - maximum values",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  math.NewInt(999999999999),
				PrizeDenom:   "longutoken",
				StakingDenom: "longustake",
				Duration:     "36500",
				MinStake:     math.NewInt(999999999999),
				Lock:         "3650",
			},
		},
		{
			name: "valid message - same prize and staking denom",
			msg: MsgCreateStakingReward{
				Creator:      validCreator,
				PrizeAmount:  validPrizeAmount,
				PrizeDenom:   "utoken",
				StakingDenom: "utoken",
				Duration:     validDuration,
				MinStake:     validMinStake,
				Lock:         validLock,
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

func TestMsgCreateStakingReward_ToStakingReward(t *testing.T) {
	validCreator := sample.AccAddress()
	validMsg := MsgCreateStakingReward{
		Creator:      validCreator,
		PrizeAmount:  math.NewInt(1000),
		PrizeDenom:   "utoken",
		StakingDenom: "ustake",
		Duration:     "30",
		MinStake:     math.NewInt(100),
		Lock:         "7",
	}

	sr, err := validMsg.ToStakingReward()
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1000), sr.PrizeAmount)
	require.Equal(t, "utoken", sr.PrizeDenom)
	require.Equal(t, "ustake", sr.StakingDenom)
	require.Equal(t, uint64(100), sr.MinStake)
	require.Equal(t, uint32(30), sr.Duration)
	require.Equal(t, uint32(7), sr.Lock)
	require.Equal(t, math.ZeroInt(), sr.StakedAmount)
	require.Equal(t, math.LegacyZeroDec(), sr.DistributedStake)
}
