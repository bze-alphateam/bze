package keeper

import (
	"context"
	"github.com/bze-alphateam/bze/x/burner/types"
)

type msgServer struct {
	Keeper
}

func (k msgServer) FundBurner(ctx context.Context, burner *types.MsgFundBurner) (*types.MsgFundBurnerResponse, error) {
	//TODO implement me
	panic("implement me")
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}
