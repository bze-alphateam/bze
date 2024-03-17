package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateStakingReward{}, "rewards/CreateStakingReward", nil)
	cdc.RegisterConcrete(&MsgUpdateStakingReward{}, "rewards/UpdateStakingReward", nil)
	cdc.RegisterConcrete(&MsgCreateTradingReward{}, "rewards/CreateTradingReward", nil)
	cdc.RegisterConcrete(&MsgJoinStaking{}, "rewards/JoinStaking", nil)
	cdc.RegisterConcrete(&MsgExitStaking{}, "rewards/ExitStaking", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateStakingReward{},
		&MsgUpdateStakingReward{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateTradingReward{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgJoinStaking{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgExitStaking{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
