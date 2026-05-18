package keeper

import (
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// msgServer embeds *Keeper (pointer) so post-construction keeper mutations
// — most notably Epic 5's SetMsgRouter from app/app.go after baseapp is
// built — are visible to dispatched handlers. Keeping it by value would
// let SetMsgRouter mutate one copy while the msgServer kept stale state.
type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper. Pass a pointer so the msg server and any
// other shared references stay in sync with subsequent mutators
// (SetMsgRouter, etc.).
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}
