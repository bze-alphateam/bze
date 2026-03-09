package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/bzeutils"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateLiquidityPool(goCtx context.Context, msg *types.MsgCreateLiquidityPool) (*types.MsgCreateLiquidityPoolResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	creatorAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	initialBase := msg.InitialBase
	initialQuote := msg.InitialQuote
	base, quote, poolId := k.CreatePoolId(msg.Base, msg.Quote)
	//if the base and quote are different from in the message, it means that CreatePoolId has reversed them in
	//alphabetical order. In this case we also have to reverse the provided amounts
	if base != msg.Base {
		initialBase, initialQuote = initialQuote, initialBase
	}

	err = k.validateMarketAssets(ctx, base, quote)
	if err != nil {
		return nil, err
	}

	err = k.validatePoolId(ctx, poolId)
	if err != nil {
		return nil, err
	}

	fee, feeDest, err := k.parseValidPoolFees(msg)
	if err != nil {
		return nil, err
	}

	rBase, rQuote, err := k.getProvidedReserves(base, quote, initialBase, initialQuote)
	if err != nil {
		return nil, err
	}

	lp := types.LiquidityPool{
		Id:           poolId,
		Base:         base,
		Quote:        quote,
		LpDenom:      k.getPoolDenom(poolId),
		Creator:      msg.Creator,
		Fee:          fee,
		FeeDest:      &feeDest,
		ReserveBase:  math.ZeroInt(),
		ReserveQuote: math.ZeroInt(),
		Stable:       msg.Stable,
	}

	if msg.Stable {
		//TODO: implement stable swap
		return nil, sdkerrors.ErrNotSupported
	}

	err = k.payMarketCreateFee(ctx, creatorAcc)
	if err != nil {
		return nil, err
	}

	//capture the initial reserves
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAcc, types.ModuleName, sdk.NewCoins(rBase, rQuote))
	if err != nil {
		return nil, err
	}

	//create the denom first
	k.createLpDenom(ctx, &lp)
	//mint initial LP Tokens
	lpTokens, err := k.mintInitialLpTokens(ctx, rBase, rQuote, &lp)
	if err != nil {
		return nil, err
	}

	if !lpTokens.IsPositive() {
		return nil, errors.Wrap(sdkerrors.ErrInvalidCoins, "resulted LP tokens must be positive")
	}
	//initial LP is forever locked
	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, burnermoduletypes.BlackHoleModuleName, sdk.NewCoins(lpTokens))
	if err != nil {
		return nil, errors.Wrap(err, "failed to send initial LP tokens to burner black hole")
	}

	k.SetLiquidityPool(ctx, lp)
	//emit LP Created event
	err = ctx.EventManager().EmitTypedEvent(
		&types.PoolCreatedEvent{
			Creator: msg.Creator,
			Base:    base,
			Quote:   quote,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgCreateLiquidityPoolResponse{
		Id: poolId,
	}, nil
}

func (k msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	creatorAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
	}

	pool, found := k.GetLiquidityPool(ctx, msg.GetPoolId())
	if !found {
		return nil, errors.Wrapf(types.ErrMarketNotFound, "pool %s not found", msg.GetPoolId())
	}

	lpSupply := k.bankKeeper.GetSupply(ctx, pool.GetLpDenom())
	if !lpSupply.IsPositive() {
		return nil, errors.Wrapf(types.ErrInvalidDenom, "could not find supply for pool %s", pool.GetId())
	}

	toRemove := sdk.NewCoin(pool.GetLpDenom(), msg.LpTokens)
	if !toRemove.IsPositive() {
		return nil, fmt.Errorf("provided LP tokens is not positive %s", toRemove)
	}

	//capture user LP tokens
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAcc, types.ModuleName, sdk.NewCoins(toRemove))
	if err != nil {
		return nil, errors.Wrap(err, "failed to send LP Tokens to module account")
	}

	userShare := math.LegacyNewDecFromInt(toRemove.Amount).Quo(math.LegacyNewDecFromInt(lpSupply.Amount))
	base := math.LegacyNewDecFromInt(pool.ReserveBase).Mul(userShare).TruncateInt()
	quote := math.LegacyNewDecFromInt(pool.ReserveQuote).Mul(userShare).TruncateInt()

	// Validate minimum amounts
	if base.LT(msg.MinBase) {
		return nil, errors.Wrapf(types.ErrResultedAmountTooLow, "base amount too low, got %s, minimum %s", base, msg.MinBase.String())
	}

	if quote.LT(msg.MinQuote) {
		return nil, errors.Wrapf(types.ErrResultedAmountTooLow, "quote amount too low, got %s, minimum %s", quote, msg.MinQuote.String())
	}

	//burn lp tokens
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(toRemove))
	if err != nil {
		return nil, errors.Wrap(err, "failed to burn LP Tokens")
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAcc, sdk.NewCoins(sdk.NewCoin(pool.GetBase(), base), sdk.NewCoin(pool.GetQuote(), quote)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to send resulted coins to user account")
	}

	pool.ReserveBase = pool.ReserveBase.Sub(base)
	pool.ReserveQuote = pool.ReserveQuote.Sub(quote)

	k.SetLiquidityPool(ctx, pool)

	//emit liquidity removed event
	err = ctx.EventManager().EmitTypedEvent(
		&types.LiquidityRemovedEvent{
			Creator:     msg.Creator,
			BaseAmount:  base,
			QuoteAmount: quote,
			PoolId:      pool.GetId(),
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgRemoveLiquidityResponse{
		Base:  base,
		Quote: quote,
	}, nil
}

func (k msgServer) AddLiquidity(goCtx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	creatorAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
	}

	pool, found := k.GetLiquidityPool(ctx, msg.GetPoolId())
	if !found {
		return nil, errors.Wrapf(types.ErrMarketNotFound, "pool %s not found", msg.GetPoolId())
	}

	poolBaseReserve := pool.ReserveBase
	poolQuoteReserve := pool.ReserveQuote
	if poolBaseReserve.IsZero() || poolQuoteReserve.IsZero() {
		//pools should not be empty, they are created with a desired price
		return nil, errors.Wrap(sdkerrors.ErrInvalidCoins, "pool is empty")
	}

	optimalBase, optimalQuote, err := k.BalanceProvidedAmounts(msg.BaseAmount, msg.QuoteAmount, pool.ReserveBase, pool.ReserveQuote)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to calculate provided amounts")
	}

	//create user paid coins
	paidBase, paidQuote, err := k.getProvidedReserves(pool.GetBase(), pool.GetQuote(), optimalBase, optimalQuote)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create optimal reserves with the provided coins")
	}

	//capture user paid coins
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAcc, types.ModuleName, sdk.NewCoins(paidBase, paidQuote))
	if err != nil {
		return nil, errors.Wrap(err, "failed to send liquidity coins to module account")
	}

	//mint LP tokens
	minted, err := k.mintDepositLpTokens(ctx, &optimalBase, &optimalQuote, &poolBaseReserve, &poolQuoteReserve, &pool)
	if err != nil {
		return nil, err
	}

	if minted.Amount.LT(msg.MinLpTokens) {
		return nil, errors.Wrapf(types.ErrResultedAmountTooLow, "could not mint the minimum expected lp tokens. Minted %d, expected minimum: %s", minted.Amount.Uint64(), msg.MinLpTokens.String())
	}

	//send LP tokens to user
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creatorAcc, sdk.NewCoins(minted))
	if err != nil {
		return nil, errors.Wrap(err, "could not send lp tokens to creator")
	}

	//increment pool reserves
	pool.ReserveBase = poolBaseReserve.Add(optimalBase)
	pool.ReserveQuote = poolQuoteReserve.Add(optimalQuote)

	k.SetLiquidityPool(ctx, pool)

	//emit liquidity added event
	err = ctx.EventManager().EmitTypedEvent(
		&types.LiquidityAddedEvent{
			Creator:      msg.Creator,
			BaseAmount:   optimalBase,
			QuoteAmount:  optimalQuote,
			MintedAmount: minted.Amount,
			PoolId:       pool.GetId(),
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgAddLiquidityResponse{
		MintedAmount: minted.Amount,
	}, nil
}

func (k msgServer) MultiSwap(goCtx context.Context, msg *types.MsgMultiSwap) (*types.MsgMultiSwapResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	creatorAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrInvalidAddress, err.Error())
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

	inputCoin := *ic
	for _, pool := range pools {
		//use the result as outputCoin for next pool swap
		swapResult, err := k.swapTokens(ctx, inputCoin, &pool)
		if err != nil {
			return nil, errors.Wrapf(types.ErrInvalidPoolSwap, "swap failed on pool %s: %s", pool.GetId(), err.Error())
		}

		//emit event and call order executed hooks
		k.onSwapSuccess(ctx, &pool, creatorAcc, inputCoin, swapResult)

		//modify input coin with the resulted coins from the swap to be used as input on the next pool in this slice
		inputCoin = swapResult
	}

	//the final output coin is the result of the last swap from the list of pools
	outputCoin := inputCoin

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

	//capture market taker trading fee
	_, err = k.captureTradingFees(ctx, creatorAcc, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pay trading fees")
	}

	return &types.MsgMultiSwapResponse{
		Output: outputCoin,
	}, nil
}

func (k msgServer) createLpDenom(ctx sdk.Context, lp *types.LiquidityPool) {
	denomMetaData := banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    lp.GetLpDenom(),
				Exponent: 0,
			},
			{
				Denom:    k.getPoolScaledDenom(lp.GetId()),
				Exponent: sharesScaleExponent,
			},
		},
		Base:    lp.GetLpDenom(),
		Display: k.getPoolScaledDenom(lp.GetId()),
	}

	k.bankKeeper.SetDenomMetaData(ctx, denomMetaData)
}

func (k msgServer) mintInitialLpTokens(ctx sdk.Context, baseCoin, quoteCoin sdk.Coin, lp *types.LiquidityPool) (lpTokens sdk.Coin, err error) {
	// Calculate base * quote
	product := math.LegacyNewDecFromInt(baseCoin.Amount).Mul(math.LegacyNewDecFromInt(quoteCoin.Amount))
	// Calculate sqrt(base * quote)
	sqrtProduct, err := product.ApproxSqrt()
	if err != nil {
		return
	}

	// Scale by a multiplier to preserve precision
	multiplier := math.LegacyNewDec(10).Power(sharesScaleExponent)
	scaledLiquidity := sqrtProduct.Mul(multiplier).TruncateInt()
	if !scaledLiquidity.IsPositive() {
		err = errors.Wrap(sdkerrors.ErrInvalidCoins, "initial liquidity provided is too low to mint LP tokens")
		return
	}

	//create the LP coin
	lpTokens = sdk.NewCoin(lp.GetLpDenom(), scaledLiquidity)
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(lpTokens))
	if err != nil {
		return
	}

	lp.ReserveBase = baseCoin.Amount
	lp.ReserveQuote = quoteCoin.Amount

	return
}

func (k msgServer) validateMarketAssets(ctx sdk.Context, base, quote string) error {
	if base == quote {
		return errors.Wrap(types.ErrInvalidDenom, "base and quote must be different")
	}

	if !k.bankKeeper.HasSupply(ctx, base) || !k.bankKeeper.HasSupply(ctx, quote) {
		return types.ErrDenomHasNoSupply
	}

	return nil
}

func (k msgServer) validatePoolId(ctx sdk.Context, poolId string) error {
	_, exists := k.GetLiquidityPool(ctx, poolId)
	if exists {
		return types.ErrMarketAlreadyExists
	}

	return nil
}

func (k msgServer) validateFeeDestination(feeDest *types.FeeDestination) error {
	//do not allow any of the destinations to be negative
	if feeDest.Treasury.IsNegative() || feeDest.Burner.IsNegative() || feeDest.Providers.IsNegative() {
		return types.ErrNegativeFeeDestination
	}

	//the sum of elements must be 1
	one := math.LegacyNewDec(1)
	sum := feeDest.Burner.Add(feeDest.Treasury).Add(feeDest.Providers)
	if !sum.Equal(one) {
		return types.ErrInvalidFeeDestination
	}

	return nil
}

func (k msgServer) validateFee(fee *math.LegacyDec) error {
	if !fee.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "fee must be positive")
	}

	if fee.LT(math.LegacyNewDecWithPrec(1, 3)) {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "fee must be at least 0.001 (0.1%)")
	}

	if fee.GT(math.LegacyNewDecWithPrec(5, 2)) {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "fee must be at most 0.05 (5%)")
	}

	return nil
}

func (k msgServer) parseValidPoolFees(msg *types.MsgCreateLiquidityPool) (fee math.LegacyDec, feeDest types.FeeDestination, err error) {
	fee, err = math.LegacyNewDecFromStr(msg.Fee)
	if err != nil {
		err = errors.Wrap(sdkerrors.ErrInvalidCoins, err.Error())
		return
	}

	err = k.validateFee(&fee)
	if err != nil {
		return
	}

	feeDest, err = msg.ParseFeeDestination()
	if err != nil {
		err = errors.Wrap(types.ErrInvalidFeeDestination, err.Error())
		return
	}

	err = k.validateFeeDestination(&feeDest)
	if err != nil {
		return
	}

	return
}

func (k msgServer) onSwapSuccess(ctx sdk.Context, pool *types.LiquidityPool, userAddress sdk.AccAddress, input, output sdk.Coin) {
	err := ctx.EventManager().EmitTypedEvent(
		&types.SwapEvent{
			Creator: userAddress.String(),
			PoolId:  pool.GetId(),
			In:      input,
			Out:     output,
		},
	)

	if err != nil {
		k.Logger().Error(err.Error())
	}

	//order hooks should be called with the amount traded of the base denomination
	baseAmount := input.Amount
	if pool.GetBase() == output.Denom {
		baseAmount = output.Amount
	}

	//call hooks for the filled order
	for _, h := range k.GetOnOrderFillHooks() {
		wrappedFn := func(ctx sdk.Context) error {
			h(ctx, pool.GetId(), baseAmount.String(), userAddress.String())

			return nil
		}

		err = bzeutils.ApplyFuncIfNoError(ctx, wrappedFn)
		if err != nil {
			k.Logger().Error(err.Error())
		}
	}
}

func (k msgServer) getRoutesPools(ctx sdk.Context, msg *types.MsgMultiSwap) (pools []types.LiquidityPool, err error) {
	if len(msg.Routes) == 0 {
		return nil, fmt.Errorf("msg does not contain any routes")
	}

	tempMap := make(map[string]struct{})
	for _, route := range msg.Routes {
		//this should not happen because it's already validated in ValidateBasic
		if _, ok := tempMap[route]; ok {
			return nil, fmt.Errorf("route %s is duplicated", route)
		}

		p, ok := k.GetLiquidityPool(ctx, route)
		if !ok {
			//stop if any pool is missing
			return nil, fmt.Errorf("pool %s not found", route)
		}

		pools = append(pools, p)
		tempMap[route] = struct{}{}
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
