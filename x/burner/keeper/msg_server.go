package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/bze-alphateam/bze/bzeutils"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/burner/types"
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

func (k msgServer) FundBurner(goCtx context.Context, msg *types.MsgFundBurner) (*types.MsgFundBurnerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	amount, err := sdk.ParseCoinsNormalized(msg.Amount)
	if err != nil {
		return nil, err
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

	return &types.MsgFundBurnerResponse{}, nil
}

func (k msgServer) MoveIbcLockedCoins(goCtx context.Context, msg *types.MsgMoveIbcLockedCoins) (*types.MsgMoveIbcLockedCoinsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg == nil {
		return nil, errors.Wrap(types.ErrInvalidRequest, "invalid message")
	}

	//defensive programming: add this check in case we ever think about modifying the ValidateBasic that allows only
	//IBC denoms in this message atm.
	if bzeutils.IsLpTokenDenom(msg.Denom) {
		return nil, errors.Wrap(types.ErrInvalidRequest, "cannot burn LP tokens")
	}

	if ok := k.bankKeeper.HasSupply(ctx, msg.Denom); !ok {
		return nil, errors.Wrap(types.ErrInvalidRequest, "denom does not exist")
	}

	lockAccount := k.accountKeeper.GetModuleAccount(ctx, types.BlackHoleModuleName)
	if lockAccount == nil {
		return nil, errors.Wrap(types.ErrInvalidRequest, "could not get lock account")
	}

	denomLockedBalance := k.bankKeeper.GetBalance(ctx, lockAccount.GetAddress(), msg.Denom)
	if !denomLockedBalance.IsPositive() {
		return nil, errors.Wrap(types.ErrInvalidRequest, "no coins to move for this denom")
	}

	if !k.tradeKeeper.CanSwapForNativeDenom(ctx, denomLockedBalance) {
		return nil, errors.Wrap(types.ErrInvalidRequest, "cannot move the locked coins due to liquidity availability")
	}

	added, refunded, err := k.tradeKeeper.ModuleAddLiquidityWithNativeDenom(ctx, types.BlackHoleModuleName, sdk.NewCoins(denomLockedBalance))
	if err != nil {
		return nil, errors.Wrap(err, "failed to move locked coins to liquidity pair")
	}

	if !added.IsAllPositive() {
		return nil, errors.Wrap(types.ErrInvalidRequest, "no liquidity was added")
	}

	// Send the refunded amount of native denom to the burner module
	// Explanation: the ModuleAddLiquidityWithNativeDenom method is swapping part of the coins provided
	// (denomLockedBalance) to native denom.
	// Then using the provided denom + native denom resulted after swap it adds liquidity to their LP.
	// Some amount might be left, and it's refunded to the caller module (in this case BlackHoleModuleName)
	// We leave the refunded amount of this msg.Denom in the module account,
	// but the native coin (BZE) should be sent to the burner module.
	for _, coin := range refunded {
		if !coin.IsPositive() {
			continue
		}

		if !k.tradeKeeper.IsNativeDenom(ctx, coin.Denom) {
			continue
		}

		err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.BlackHoleModuleName, types.ModuleName, sdk.NewCoins(coin))
		if err != nil {
			return nil, errors.Wrap(err, "failed to send refunded native coins to burner module")
		}

		//we can break, we're searching only for the native coins to send to the burner module
		break
	}

	return &types.MsgMoveIbcLockedCoinsResponse{
		Added:    added.String(),
		Refunded: refunded.String(),
	}, nil
}
