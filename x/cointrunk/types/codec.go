package types

import (
	"github.com/bze-alphateam/bze/x/cointrunk/v1types"
	v2types "github.com/bze-alphateam/bze/x/cointrunk/v2types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAddArticle{},
		&MsgPayPublisherRespect{},
		&MsgAcceptDomain{},
		&MsgSavePublisher{},
		&MsgUpdateParams{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&v2types.MsgPayPublisherRespect{},
	)
	registry.RegisterInterface(
		"bze.cointrunk.v1.AcceptedDomainProposal",
		(*v1beta1.Content)(nil),
		&v1types.AcceptedDomainProposal{},
	)

	registry.RegisterInterface(
		"bze.cointrunk.v1.PublisherProposal",
		(*v1beta1.Content)(nil),
		&v1types.PublisherProposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
