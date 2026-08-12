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
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateStakingReward{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgJoinStaking{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgExitStaking{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgClaimStakingRewards{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDistributeStakingRewards{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateTradingReward{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgActivateTradingReward{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDeleteStakingReward{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateDenomReward{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgJoinDenomReward{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgExitDenomReward{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgClaimDenomRewards{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateDenomRewardSchedule{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateDenomRewardSchedule{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDistributeDenomRewards{},
	)
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
