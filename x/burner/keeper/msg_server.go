package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
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

	params := k.GetParams(ctx)
	if err := k.checkIbcMoveCooldown(ctx, msg.Denom, params.IbcMoveCooldownBlocks); err != nil {
		return nil, err
	}

	toMove, err := k.capIbcMove(ctx, denomLockedBalance, params.IbcMoveMaxReserveRatio)
	if err != nil {
		return nil, err
	}

	if !k.tradeKeeper.CanSwapForNativeDenom(ctx, toMove) {
		return nil, errors.Wrap(types.ErrInvalidRequest, "cannot move the locked coins due to liquidity availability")
	}

	added, refunded, err := k.tradeKeeper.SafeModuleAddLiquidityWithNativeDenom(ctx, types.BlackHoleModuleName, sdk.NewCoins(toMove))
	if err != nil {
		return nil, errors.Wrap(err, "failed to move locked coins to liquidity pair")
	}

	if !added.IsAllPositive() {
		return nil, errors.Wrap(types.ErrInvalidRequest, "no liquidity was added")
	}

	// Send the refunded amount of native denom to the burner module
	// Explanation: the SafeModuleAddLiquidityWithNativeDenom method is swapping part of the coins provided
	// (toMove) to native denom.
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

	k.SetIbcMoveLock(ctx, types.IbcMoveLock{Denom: msg.Denom, LastMoveHeight: ctx.BlockHeight()})

	return &types.MsgMoveIbcLockedCoinsResponse{
		Added:    added.String(),
		Refunded: refunded.String(),
	}, nil
}

// checkIbcMoveCooldown refuses a move of denom while fewer than cooldownBlocks blocks have passed since its last
// successful move. A denom that was never moved has no lock and can be moved.
func (k msgServer) checkIbcMoveCooldown(ctx sdk.Context, denom string, cooldownBlocks uint64) error {
	lock, found := k.GetIbcMoveLock(ctx, denom)
	if !found {
		return nil
	}

	elapsed := ctx.BlockHeight() - lock.LastMoveHeight
	if elapsed < 0 || uint64(elapsed) < cooldownBlocks {
		return errors.Wrapf(
			types.ErrIbcMoveCooldown,
			"last move at height %d, next one allowed at height %d",
			lock.LastMoveHeight,
			lock.LastMoveHeight+int64(cooldownBlocks), //nolint:gosec // the cooldown is a validated block count
		)
	}

	return nil
}

// capIbcMove returns the part of the locked balance one move may add to the native pool of its denom: at most
// floor(maxReserveRatio x the pool's reserve of the denom). It refuses when there is no such pool or the cap is 0.
func (k msgServer) capIbcMove(ctx sdk.Context, locked sdk.Coin, maxReserveRatio math.LegacyDec) (sdk.Coin, error) {
	nativeDenom := k.tradeKeeper.GetParams(ctx).NativeDenom
	_, _, poolId := k.tradeKeeper.CreatePoolId(nativeDenom, locked.Denom)
	pool, found := k.tradeKeeper.GetLiquidityPool(ctx, poolId)
	if !found {
		return sdk.Coin{}, errors.Wrapf(types.ErrIbcMoveNoCapacity, "no liquidity pool %s", poolId)
	}

	reserve := pool.ReserveQuote
	if pool.Base == locked.Denom {
		reserve = pool.ReserveBase
	}

	if reserve.IsNil() || maxReserveRatio.IsNil() {
		return sdk.Coin{}, errors.Wrapf(types.ErrIbcMoveNoCapacity, "pool %s has no reserve to cap the move", poolId)
	}

	limit := maxReserveRatio.MulInt(reserve).TruncateInt()
	if !limit.IsPositive() {
		return sdk.Coin{}, errors.Wrapf(types.ErrIbcMoveNoCapacity, "pool %s reserve is too small to move any coins", poolId)
	}

	return sdk.NewCoin(locked.Denom, math.MinInt(locked.Amount, limit)), nil
}
