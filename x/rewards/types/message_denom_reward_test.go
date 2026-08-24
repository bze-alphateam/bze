package types

import (
	"cosmossdk.io/math"
	"strconv"
	"testing"

	"github.com/bze-alphateam/bze/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgCreateDenomReward_ValidateBasic(t *testing.T) {
	creator := sample.AccAddress()
	tests := []struct {
		name string
		msg  MsgCreateDenomReward
		err  error
	}{
		{name: "invalid creator", msg: MsgCreateDenomReward{Creator: "invalid", Denom: "ubze"}, err: sdkerrors.ErrInvalidAddress},
		{name: "empty creator", msg: MsgCreateDenomReward{Creator: "", Denom: "ubze"}, err: sdkerrors.ErrInvalidAddress},
		{name: "empty denom", msg: MsgCreateDenomReward{Creator: creator, Denom: ""}, err: ErrInvalidStakingDenom},
		{name: "invalid denom char", msg: MsgCreateDenomReward{Creator: creator, Denom: "!"}, err: ErrInvalidStakingDenom},
		{name: "denom too short", msg: MsgCreateDenomReward{Creator: creator, Denom: "a"}, err: ErrInvalidStakingDenom},
		{name: "denom leading digit", msg: MsgCreateDenomReward{Creator: creator, Denom: "1abc"}, err: ErrInvalidStakingDenom},
		{name: "valid", msg: MsgCreateDenomReward{Creator: creator, Denom: "ubze"}},
		{name: "valid uppercase denom", msg: MsgCreateDenomReward{Creator: creator, Denom: "UBZE"}},
		{name: "valid ibc denom", msg: MsgCreateDenomReward{Creator: creator, Denom: "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"}},
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

func TestMsgJoinDenomReward_ValidateBasic(t *testing.T) {
	creator := sample.AccAddress()
	tests := []struct {
		name string
		msg  MsgJoinDenomReward
		err  error
	}{
		{name: "invalid creator", msg: MsgJoinDenomReward{Creator: "invalid", Denom: "ubze", Amount: math.NewInt(100)}, err: sdkerrors.ErrInvalidAddress},
		{name: "empty denom", msg: MsgJoinDenomReward{Creator: creator, Denom: "", Amount: math.NewInt(100)}, err: ErrInvalidStakingDenom},
		{name: "invalid denom char", msg: MsgJoinDenomReward{Creator: creator, Denom: "!", Amount: math.NewInt(100)}, err: ErrInvalidStakingDenom},
		{name: "denom too short", msg: MsgJoinDenomReward{Creator: creator, Denom: "a", Amount: math.NewInt(100)}, err: ErrInvalidStakingDenom},
		{name: "denom leading digit", msg: MsgJoinDenomReward{Creator: creator, Denom: "1abc", Amount: math.NewInt(100)}, err: ErrInvalidStakingDenom},
		{name: "unset amount", msg: MsgJoinDenomReward{Creator: creator, Denom: "ubze"}, err: ErrInvalidAmount},
		{name: "negative amount", msg: MsgJoinDenomReward{Creator: creator, Denom: "ubze", Amount: math.NewInt(-1)}, err: ErrInvalidAmount},
		{name: "zero amount", msg: MsgJoinDenomReward{Creator: creator, Denom: "ubze", Amount: math.NewInt(0)}, err: ErrInvalidAmount},
		{name: "valid", msg: MsgJoinDenomReward{Creator: creator, Denom: "ubze", Amount: math.NewInt(100)}},
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

func TestMsgExitDenomReward_ValidateBasic(t *testing.T) {
	creator := sample.AccAddress()
	tests := []struct {
		name string
		msg  MsgExitDenomReward
		err  error
	}{
		{name: "invalid creator", msg: MsgExitDenomReward{Creator: "invalid", Denom: "ubze"}, err: sdkerrors.ErrInvalidAddress},
		{name: "empty denom", msg: MsgExitDenomReward{Creator: creator, Denom: ""}, err: ErrInvalidStakingDenom},
		{name: "invalid denom char", msg: MsgExitDenomReward{Creator: creator, Denom: "!"}, err: ErrInvalidStakingDenom},
		{name: "denom too short", msg: MsgExitDenomReward{Creator: creator, Denom: "a"}, err: ErrInvalidStakingDenom},
		{name: "denom leading digit", msg: MsgExitDenomReward{Creator: creator, Denom: "1abc"}, err: ErrInvalidStakingDenom},
		{name: "valid", msg: MsgExitDenomReward{Creator: creator, Denom: "ubze"}},
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

func TestMsgClaimDenomRewards_ValidateBasic(t *testing.T) {
	creator := sample.AccAddress()
	tests := []struct {
		name string
		msg  MsgClaimDenomRewards
		err  error
	}{
		{name: "invalid creator", msg: MsgClaimDenomRewards{Creator: "invalid", Denom: "ubze"}, err: sdkerrors.ErrInvalidAddress},
		{name: "empty denom", msg: MsgClaimDenomRewards{Creator: creator, Denom: ""}, err: ErrInvalidStakingDenom},
		{name: "invalid denom char", msg: MsgClaimDenomRewards{Creator: creator, Denom: "!"}, err: ErrInvalidStakingDenom},
		{name: "denom too short", msg: MsgClaimDenomRewards{Creator: creator, Denom: "a"}, err: ErrInvalidStakingDenom},
		{name: "denom leading digit", msg: MsgClaimDenomRewards{Creator: creator, Denom: "1abc"}, err: ErrInvalidStakingDenom},
		{name: "valid", msg: MsgClaimDenomRewards{Creator: creator, Denom: "ubze"}},
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

func TestMsgCreateDenomRewardSchedule_ValidateBasic(t *testing.T) {
	creator := sample.AccAddress()
	tooLong := strconv.Itoa(HundredYearsInDays + 1)
	tests := []struct {
		name string
		msg  MsgCreateDenomRewardSchedule
		err  error
	}{
		{name: "invalid creator", msg: MsgCreateDenomRewardSchedule{Creator: "invalid", Denom: "ubze", PrizeDenom: "uprize", DailyAmount: math.NewInt(100), Duration: "30"}, err: sdkerrors.ErrInvalidAddress},
		{name: "empty denom", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "", PrizeDenom: "uprize", DailyAmount: math.NewInt(100), Duration: "30"}, err: ErrInvalidStakingDenom},
		{name: "invalid denom char", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "!", PrizeDenom: "uprize", DailyAmount: math.NewInt(100), Duration: "30"}, err: ErrInvalidStakingDenom},
		{name: "denom too short", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "a", PrizeDenom: "uprize", DailyAmount: math.NewInt(100), Duration: "30"}, err: ErrInvalidStakingDenom},
		{name: "denom leading digit", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "1abc", PrizeDenom: "uprize", DailyAmount: math.NewInt(100), Duration: "30"}, err: ErrInvalidStakingDenom},
		{name: "empty prize denom", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "", DailyAmount: math.NewInt(100), Duration: "30"}, err: ErrInvalidPrizeDenom},
		{name: "invalid prize denom char", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "!", DailyAmount: math.NewInt(100), Duration: "30"}, err: ErrInvalidPrizeDenom},
		{name: "prize denom too short", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "u", DailyAmount: math.NewInt(100), Duration: "30"}, err: ErrInvalidPrizeDenom},
		{name: "prize denom leading digit", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "1abc", DailyAmount: math.NewInt(100), Duration: "30"}, err: ErrInvalidPrizeDenom},
		{name: "unset daily amount", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "uprize", Duration: "30"}, err: ErrInvalidAmount},
		{name: "zero daily amount", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "uprize", DailyAmount: math.NewInt(0), Duration: "30"}, err: ErrInvalidAmount},
		{name: "non-numeric duration", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "uprize", DailyAmount: math.NewInt(100), Duration: "x"}, err: ErrInvalidDuration},
		{name: "zero duration", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "uprize", DailyAmount: math.NewInt(100), Duration: "0"}, err: ErrInvalidDuration},
		{name: "duration too long", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "uprize", DailyAmount: math.NewInt(100), Duration: tooLong}, err: ErrInvalidDuration},
		{name: "valid", msg: MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "uprize", DailyAmount: math.NewInt(100), Duration: "30"}},
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

func TestMsgUpdateDenomRewardSchedule_ValidateBasic(t *testing.T) {
	creator := sample.AccAddress()
	tooLong := strconv.Itoa(HundredYearsInDays + 1)
	tests := []struct {
		name string
		msg  MsgUpdateDenomRewardSchedule
		err  error
	}{
		{name: "invalid creator", msg: MsgUpdateDenomRewardSchedule{Creator: "invalid", Denom: "ubze", ScheduleId: "000000000001", Duration: "30"}, err: sdkerrors.ErrInvalidAddress},
		{name: "empty denom", msg: MsgUpdateDenomRewardSchedule{Creator: creator, Denom: "", ScheduleId: "000000000001", Duration: "30"}, err: ErrInvalidStakingDenom},
		{name: "invalid denom char", msg: MsgUpdateDenomRewardSchedule{Creator: creator, Denom: "!", ScheduleId: "000000000001", Duration: "30"}, err: ErrInvalidStakingDenom},
		{name: "denom too short", msg: MsgUpdateDenomRewardSchedule{Creator: creator, Denom: "a", ScheduleId: "000000000001", Duration: "30"}, err: ErrInvalidStakingDenom},
		{name: "denom leading digit", msg: MsgUpdateDenomRewardSchedule{Creator: creator, Denom: "1abc", ScheduleId: "000000000001", Duration: "30"}, err: ErrInvalidStakingDenom},
		{name: "empty schedule id", msg: MsgUpdateDenomRewardSchedule{Creator: creator, Denom: "ubze", ScheduleId: "", Duration: "30"}, err: ErrInvalidScheduleId},
		{name: "non-numeric duration", msg: MsgUpdateDenomRewardSchedule{Creator: creator, Denom: "ubze", ScheduleId: "000000000001", Duration: "x"}, err: ErrInvalidDuration},
		{name: "zero duration", msg: MsgUpdateDenomRewardSchedule{Creator: creator, Denom: "ubze", ScheduleId: "000000000001", Duration: "0"}, err: ErrInvalidDuration},
		{name: "duration too long", msg: MsgUpdateDenomRewardSchedule{Creator: creator, Denom: "ubze", ScheduleId: "000000000001", Duration: tooLong}, err: ErrInvalidDuration},
		{name: "valid", msg: MsgUpdateDenomRewardSchedule{Creator: creator, Denom: "ubze", ScheduleId: "000000000001", Duration: "30"}},
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

func TestMsgDistributeDenomRewards_ValidateBasic(t *testing.T) {
	creator := sample.AccAddress()
	tests := []struct {
		name string
		msg  MsgDistributeDenomRewards
		err  error
	}{
		{name: "invalid creator", msg: MsgDistributeDenomRewards{Creator: "invalid", Denom: "ubze", PrizeDenom: "uprize", Amount: math.NewInt(100)}, err: sdkerrors.ErrInvalidAddress},
		{name: "empty denom", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "", PrizeDenom: "uprize", Amount: math.NewInt(100)}, err: ErrInvalidStakingDenom},
		{name: "invalid denom char", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "!", PrizeDenom: "uprize", Amount: math.NewInt(100)}, err: ErrInvalidStakingDenom},
		{name: "denom too short", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "a", PrizeDenom: "uprize", Amount: math.NewInt(100)}, err: ErrInvalidStakingDenom},
		{name: "denom leading digit", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "1abc", PrizeDenom: "uprize", Amount: math.NewInt(100)}, err: ErrInvalidStakingDenom},
		{name: "empty prize denom", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "ubze", PrizeDenom: "", Amount: math.NewInt(100)}, err: ErrInvalidPrizeDenom},
		{name: "invalid prize denom char", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "ubze", PrizeDenom: "!", Amount: math.NewInt(100)}, err: ErrInvalidPrizeDenom},
		{name: "prize denom too short", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "ubze", PrizeDenom: "u", Amount: math.NewInt(100)}, err: ErrInvalidPrizeDenom},
		{name: "prize denom leading digit", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "ubze", PrizeDenom: "1abc", Amount: math.NewInt(100)}, err: ErrInvalidPrizeDenom},
		{name: "valid uppercase prize denom", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "ubze", PrizeDenom: "UPRIZE", Amount: math.NewInt(100)}},
		{name: "unset amount", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "ubze", PrizeDenom: "uprize"}, err: ErrInvalidAmount},
		{name: "zero amount", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "ubze", PrizeDenom: "uprize", Amount: math.NewInt(0)}, err: ErrInvalidAmount},
		{name: "valid", msg: MsgDistributeDenomRewards{Creator: creator, Denom: "ubze", PrizeDenom: "uprize", Amount: math.NewInt(100)}},
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

// TestNewMsgDenomReward_Constructors sanity-checks the message constructors.
func TestNewMsgDenomReward_Constructors(t *testing.T) {
	creator := sample.AccAddress()

	require.Equal(t, &MsgCreateDenomReward{Creator: creator, Denom: "ubze"}, NewMsgCreateDenomReward(creator, "ubze"))
	require.Equal(t, &MsgJoinDenomReward{Creator: creator, Denom: "ubze", Amount: math.NewInt(100)}, NewMsgJoinDenomReward(creator, "ubze", math.NewInt(100)))
	require.Equal(t, &MsgExitDenomReward{Creator: creator, Denom: "ubze"}, NewMsgExitDenomReward(creator, "ubze"))
	require.Equal(t, &MsgClaimDenomRewards{Creator: creator, Denom: "ubze"}, NewMsgClaimDenomRewards(creator, "ubze"))
	require.Equal(t, &MsgCreateDenomRewardSchedule{Creator: creator, Denom: "ubze", PrizeDenom: "uprize", DailyAmount: math.NewInt(100), Duration: "30"}, NewMsgCreateDenomRewardSchedule(creator, "ubze", "uprize", math.NewInt(100), "30"))
	require.Equal(t, &MsgUpdateDenomRewardSchedule{Creator: creator, Denom: "ubze", ScheduleId: "000000000001", Duration: "30"}, NewMsgUpdateDenomRewardSchedule(creator, "ubze", "000000000001", "30"))
	require.Equal(t, &MsgDistributeDenomRewards{Creator: creator, Denom: "ubze", PrizeDenom: "uprize", Amount: math.NewInt(100)}, NewMsgDistributeDenomRewards(creator, "ubze", "uprize", math.NewInt(100)))
}
