package types

import (
    
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
	&MsgCreateMarket{},
)
registry.RegisterImplementations((*sdk.Msg)(nil),
	&MsgCreateOrder{},
)
registry.RegisterImplementations((*sdk.Msg)(nil),
	&MsgCancelOrder{},
)
registry.RegisterImplementations((*sdk.Msg)(nil),
	&MsgFillOrders{},
)
registry.RegisterImplementations((*sdk.Msg)(nil),
	&MsgCreateLiquidityPool{},
)
registry.RegisterImplementations((*sdk.Msg)(nil),
	&MsgAddLiquidity{},
)
registry.RegisterImplementations((*sdk.Msg)(nil),
	&MsgRemoveLiquidity{},
)
registry.RegisterImplementations((*sdk.Msg)(nil),
	&MsgMultiSwap{},
)
// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}


