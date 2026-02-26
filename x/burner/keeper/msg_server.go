package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/bze-alphateam/bze/x/burner/types"
	v2types "github.com/bze-alphateam/bze/x/burner/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) FundBurner(goCtx context.Context, msg *v2types.MsgFundBurner) (*v2types.MsgFundBurnerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	amount := msg.Amount
	if !amount.IsAllPositive() {
		return nil, errors.Wrap(types.ErrInvalidBurnAmount, "provided amounts are not positive")
	}

	burnable, exchangeable, lockable := k.filterCoinsToBurn(ctx, amount)
	toBurnerModule := burnable.Add(exchangeable...)
	total := toBurnerModule.Add(lockable...)
	if total.IsZero() {
		return nil, errors.Wrap(types.ErrInvalidBurnAmount, "provided amounts can not be burned, locked or exchanged")
	}

	creatorAccount, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	if lockable.IsAllPositive() {
		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAccount, types.BlackHoleModuleName, lockable)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to send coins to locker")
		}
	}

	if toBurnerModule.IsAllPositive() {
		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAccount, types.ModuleName, toBurnerModule)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to send coins to burner module")
		}
	}

	err = ctx.EventManager().EmitTypedEvent(&types.FundBurnerEvent{From: msg.Creator, Amount: total.String()})
	if err != nil {
		ctx.Logger().Error("failed to emit fund burner event", "error", err)
	}

	return &v2types.MsgFundBurnerResponse{}, nil
}
