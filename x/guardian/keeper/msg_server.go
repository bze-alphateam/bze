package keeper

import (
	"github.com/bze-alphateam/bze/x/guardian/types"
)

// msgServer embeds the keeper by pointer, not by value, so the rate-limiting
// keeper wired after the msg server is registered is visible to the handlers.
type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper. The keeper is shared, not copied (see msgServer).
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}
