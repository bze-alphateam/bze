package keeper

import (
	"github.com/bze-alphateam/bze/x/dao/types"
)

// msgServer embeds the keeper by pointer, not by value, so wiring done on the
// app's keeper after the msg server is registered (the message router, the
// x/rewards hooks) is visible to every message handler.
type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper. The keeper is shared, not copied (see msgServer).
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}
