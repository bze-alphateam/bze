package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterInterfaces registers the daodao module's message types on the SDK
// interface registry. Called once at module wiring time.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateDao{},
		&MsgUpdateDaoMetadata{},
		&MsgUpdateDaoAdmin{},
		&MsgAcceptDaoAdmin{},
		&MsgUpdateMembers{},
		// Epic 3 messages.
		&MsgCreateProposal{},
		&MsgVote{},
		&MsgUpdateGovernanceConfig{},
		// Epic 4 messages.
		&MsgDeposit{},
		&MsgUpdateDepositConfig{},
		// Epic 5 messages.
		&MsgExecuteProposal{},
		&MsgRenounceAdmin{},
		&MsgUpdateVotingBackend{},
		// Epic 6 messages.
		&MsgCreatePoll{},
		&MsgVoteOnPoll{},
		&MsgDepositOnPoll{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
