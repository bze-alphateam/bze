package types

import (
	"github.com/bze-alphateam/bze/x/burner/v1types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgFundBurner{},
		&MsgStartRaffle{},
		&MsgJoinRaffle{},
		&MsgMoveIbcLockedCoins{},
	)

	registry.RegisterInterface(
		"bze.burner.v1.BurnCoinsProposal",
		(*v1beta1.Content)(nil),
		&v1types.BurnCoinsProposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
