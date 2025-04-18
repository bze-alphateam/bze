package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"fmt"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) MultiSwap(goCtx context.Context, msg *types.MsgMultiSwap) (*types.MsgMultiSwapResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	creatorAcc := msg.GetCreatorAcc()
	if creatorAcc == nil {
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, "invalid creator address")
	}

	ic, moc, err := k.getMessageCoins(msg)
	if err != nil {
		//should never happen, errors are handled in ValidateBasic
		return nil, errors.Wrapf(types.ErrInvalidOrderAmount, "invalid coins provided (%s)", err.Error())
	}

	pools, err := k.getRoutesPools(ctx, msg)
	if err != nil {
		return nil, errors.Wrapf(types.ErrInvalidRoutes, "invalid pools (%s)", err.Error())
	}

	//capture user input coins
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAcc, types.ModuleName, sdk.NewCoins(*ic))
	if err != nil {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidCoins, "could not capture user input coins %s", err.Error())
	}

	outputCoin := *ic
	for _, pool := range pools {
		//use the result as outputCoin for next pool swap
		outputCoin, err = k.swapTokens(ctx, outputCoin, &pool, creatorAcc)
		if err != nil {
			return nil, errors.Wrapf(types.ErrInvalidPoolSwap, "swap failed on pool %s: %s", pool.GetId(), err.Error())
		}
	}

	//last outputCoin should be expected output
	if outputCoin.Denom != moc.Denom {
		return nil, errors.Wrapf(types.ErrInvalidPoolSwap, "expected %s output, got %s", moc.Denom, outputCoin.Denom)
	}

	//check the minimum expected output
	if outputCoin.Amount.LT(moc.Amount) {
		return nil, errors.Wrapf(types.ErrResultedAmountTooLow, "expected minimum %s, got %s", moc.String(), outputCoin.String())
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAcc, sdk.NewCoins(outputCoin))
	if err != nil {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidCoins, "could not send bought coins %s", err.Error())
	}

	return &types.MsgMultiSwapResponse{
		Output: outputCoin,
	}, nil
}

func (k msgServer) swapTokens(ctx sdk.Context, input sdk.Coin, pool *types.LiquidityPool, userAddress sdk.AccAddress) (output sdk.Coin, err error) {
	if !pool.HasDenom(input.Denom) {
		return output, fmt.Errorf("denom %s does not exist in pool %s", input.Denom, pool.GetId())
	}

	realInput, fee := k.calculateSwapInputAndFee(input, pool)
	err = k.collectSwapFee(ctx, fee, pool)
	if err != nil {
		return output, err
	}

	inputReserve, outputReserve := pool.GetReservesCoinsByDenom(input.Denom)

	//output_reserve x real_input (the input - fee)
	prod := sdk.NewDecFromInt(outputReserve.Amount.Mul(realInput.Amount))

	//input_reserve + real_input (the input - fee)
	quo := sdk.NewDecFromInt(inputReserve.Amount.Add(realInput.Amount))
	if !quo.IsPositive() || !prod.IsPositive() {
		return output, fmt.Errorf("non positive product or quotient on swap tokens")
	}

	outputAmount := prod.Quo(quo).TruncateInt()
	output = sdk.NewCoin(outputReserve.Denom, outputAmount)
	err = pool.ChangeReserves(realInput, output)
	if err != nil {
		return output, err
	}

	k.SetLiquidityPool(ctx, *pool)

	err = ctx.EventManager().EmitTypedEvent(
		&types.SwapEvent{
			Creator: userAddress.String(),
			PoolId:  pool.GetId(),
			In:      input,
			Out:     output,
		},
	)

	if err != nil {
		k.Logger(ctx).Error(err.Error())
	}

	return output, nil
}

func (k msgServer) collectSwapFee(ctx sdk.Context, fee sdk.Coin, pool *types.LiquidityPool) error {
	//string treasury = community pool
	//string burner = burner module
	//string providers = add to LP reserve directly
	//string liquidity = add LP periodically with the tokens captured by this part.

	return nil
}

func (k msgServer) calculateSwapInputAndFee(input sdk.Coin, pool *types.LiquidityPool) (remainingInput, fee sdk.Coin) {
	feeAmount := sdk.NewDecFromInt(input.Amount).Mul(pool.Fee).TruncateInt()
	rAmount := input.Amount.Sub(feeAmount)

	return sdk.NewCoin(input.GetDenom(), rAmount), sdk.NewCoin(input.GetDenom(), feeAmount)
}

func (k msgServer) getRoutesPools(ctx sdk.Context, msg *types.MsgMultiSwap) (pools []types.LiquidityPool, err error) {
	if len(msg.Routes) == 0 {
		return nil, fmt.Errorf("msg does not contain any routes")
	}

	for _, route := range msg.Routes {
		p, ok := k.GetLiquidityPool(ctx, route)
		if !ok {
			//stop if any pool is missing
			return nil, fmt.Errorf("pool %s not found", route)
		}

		pools = append(pools, p)
	}

	return pools, nil
}

// getMessageCoins - returns the input coin and minimum output coins of the message.
// it should never return an error as the same validations are handled in ValidateBasic
func (k msgServer) getMessageCoins(msg *types.MsgMultiSwap) (*sdk.Coin, *sdk.Coin, error) {
	ic := msg.GetInput()
	if !ic.IsValid() {
		return nil, nil, fmt.Errorf("invalid input")
	}

	moc := msg.GetMinOutput()
	if !moc.IsValid() {
		return nil, nil, fmt.Errorf("invalid minimum output")
	}

	if !ic.IsPositive() || !moc.IsPositive() {
		return nil, nil, errors.Wrap(sdkerrors.ErrInvalidCoins, "invalid input or min output")
	}

	return &ic, &moc, nil
}
