package types

import (
	"github.com/bze-alphateam/bze/x/cointrunk/v1types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddArticle{},
		&MsgPayPublisherRespect{},
		&MsgAcceptDomain{},
		&MsgSavePublisher{},
		&MsgUpdateParams{},
		&v1types.AcceptedDomainProposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
