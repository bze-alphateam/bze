package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateStakingReward{},
		&MsgUpdateStakingReward{},
		&MsgJoinStaking{},
		&MsgExitStaking{},
		&MsgClaimStakingRewards{},
		&MsgDistributeStakingRewards{},
		&MsgCreateTradingReward{},
		&MsgActivateTradingReward{},
		&MsgDeleteStakingReward{},
		&MsgCreateDenomReward{},
		&MsgJoinDenomReward{},
		&MsgExitDenomReward{},
		&MsgClaimDenomRewards{},
		&MsgCreateDenomRewardSchedule{},
		&MsgUpdateDenomRewardSchedule{},
		&MsgDistributeDenomRewards{},
		&MsgUpdateParams{},
	)
	// this line is used by starport scaffolding # 3
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
