package keeper

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// msgServer embeds the keeper by pointer, not by value: the app registers the msg server inside
// appBuilder.Build (RegisterServices) and wires the order-fill hooks (Keeper.SetOnOrderFillHooks)
// only afterwards, so a by-value copy taken at registration time would never see them and the AMM
// swap path would run with no hooks forever. Sharing the pointer makes hooks registered at any
// point visible to every message handler, the same way EndBlock's ProcessingEngine sees them.
type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper. The keeper is shared, not copied (see msgServer).
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}
