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
// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}


