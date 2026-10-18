package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateDao{},
		&MsgAdminExecute{},
		&MsgNominateAdmin{},
		&MsgAcceptAdminNomination{},
		&MsgWithdrawAdminNomination{},
		&MsgSubmitProposal{},
		&MsgVote{},
		&MsgCancelProposal{},
		&MsgSettleProposal{},
		&MsgCleanupProposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
