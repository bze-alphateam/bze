package keeper

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
)

// msgServer embeds the keeper by pointer, not by value: the app registers the msg server inside
// appBuilder.Build (RegisterServices) and wires hooks (Keeper.SetHooks) only afterwards, so a
// by-value copy taken at registration time would never see them and every message handler would
// run with nil hooks forever. Sharing the pointer makes hooks registered at any point visible to
// JoinStaking, ExitStaking and DeleteStakingReward.
type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper. The keeper is shared, not copied (see msgServer).
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}
