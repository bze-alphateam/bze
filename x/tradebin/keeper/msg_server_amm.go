package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"fmt"
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
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	base, quote, poolId := k.CreatePoolId(msg.Base, msg.Quote)
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

	rBase, rQuote, err := k.getProvidedReserves(base, quote, msg.InitialBase, msg.InitialQuote)
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
	//initial LP is forever locked - that's why we don't send the minted tokens anywhere

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
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, err.Error())
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
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, err.Error())
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
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, err.Error())
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

func (k msgServer) getProvidedReserves(baseDenom, quoteDenom string, baseAmt, quoteAmt math.Int) (baseCoin, quoteCoin sdk.Coin, err error) {
	baseCoin = sdk.NewCoin(baseDenom, baseAmt)
	quoteCoin = sdk.NewCoin(quoteDenom, quoteAmt)
	if !baseCoin.IsValid() || !quoteCoin.IsValid() {
		err = errors.Wrap(sdkerrors.ErrInvalidCoins, "invalid reserve")
		return
	}

	if !baseCoin.IsPositive() || !quoteCoin.IsPositive() {
		err = errors.Wrap(sdkerrors.ErrInvalidCoins, "non positive reserve provided")
		return
	}

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
	//the sum of elements must be 1
	if !feeDest.Burner.Add(feeDest.Treasury).Add(feeDest.Providers).Equal(math.LegacyNewDecWithPrec(1, 0)) {
		return types.ErrInvalidFeeDestination
	}

	//do not allow any of the destinations to be negative
	if feeDest.Treasury.IsNegative() || feeDest.Burner.IsNegative() || feeDest.Providers.IsNegative() {
		return types.ErrNegativeFeeDestination
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

func (k msgServer) mintDepositLpTokens(ctx sdk.Context, baseAmount, quoteAmount, poolBaseReserve, poolQuoteReserve *math.Int, lp *types.LiquidityPool) (mintedLp sdk.Coin, err error) {
	lpSupply := k.bankKeeper.GetSupply(ctx, lp.GetLpDenom())
	if !lpSupply.IsPositive() {
		return mintedLp, errors.Wrapf(types.ErrInvalidDenom, "could not find supply for pool %s", lp.GetId())
	}

	baseRatio := math.LegacyNewDecFromInt(*baseAmount).Quo(math.LegacyNewDecFromInt(*poolBaseReserve))
	quoteRatio := math.LegacyNewDecFromInt(*quoteAmount).Quo(math.LegacyNewDecFromInt(*poolQuoteReserve))

	var mintRatio math.LegacyDec
	if baseRatio.LT(quoteRatio) {
		mintRatio = baseRatio
	} else {
		mintRatio = quoteRatio
	}

	tokensToMint := mintRatio.Mul(math.LegacyNewDecFromInt(lpSupply.Amount)).TruncateInt()
	if !tokensToMint.IsPositive() {
		return mintedLp, errors.Wrap(sdkerrors.ErrInvalidCoins, "resulted LP shares is not positive")
	}

	mintedLp = sdk.NewCoin(lp.GetLpDenom(), tokensToMint)
	// Mint the LP tokens
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(mintedLp))
	if err != nil {
		return mintedLp, errors.Wrapf(err, "could not mint liquidity pool tokens %s", lp.GetId())
	}

	return mintedLp, nil
}

func (k msgServer) swapTokens(ctx sdk.Context, input sdk.Coin, pool *types.LiquidityPool, userAddress sdk.AccAddress) (output sdk.Coin, err error) {
	if !pool.HasDenom(input.Denom) {
		return output, fmt.Errorf("denom %s does not exist in pool %s", input.Denom, pool.GetId())
	}

	realInput, fee := k.calculateSwapInputAndFee(input, pool)
	feeToPool, err := k.collectSwapFee(ctx, fee, pool)
	if err != nil {
		return output, err
	}

	inputReserve, outputReserve := pool.GetReservesCoinsByDenom(input.Denom)

	//output_reserve x real_input (the input - fee)
	prod := math.LegacyNewDecFromInt(outputReserve.Amount.Mul(realInput.Amount))

	//input_reserve + real_input (the input - fee)
	quo := math.LegacyNewDecFromInt(inputReserve.Amount.Add(realInput.Amount))
	if !quo.IsPositive() || !prod.IsPositive() {
		return output, fmt.Errorf("non positive product or quotient on swap tokens")
	}

	outputAmount := prod.Quo(quo).TruncateInt()
	output = sdk.NewCoin(outputReserve.Denom, outputAmount)
	//add the part of the fee that should remain in the LP (as LP Reward to LP providers)
	err = pool.ChangeReserves(realInput.Add(feeToPool), output)
	if err != nil {
		return output, err
	}

	k.SetLiquidityPool(ctx, *pool)

	//emit event and call order executed hooks
	k.onSwapSuccess(ctx, pool, userAddress, input, output)

	return output, nil
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

// collectSwapFee - calculates the distribution of the fee, and it returns the part of the fee that should be added to
// LP (for LP Providers rewards)
func (k msgServer) collectSwapFee(ctx sdk.Context, fee sdk.Coin, pool *types.LiquidityPool) (sdk.Coin, error) {
	if !fee.IsPositive() {
		//return 0 coin
		return sdk.NewCoin(fee.Denom, math.ZeroInt()), nil
	}

	feeDec := math.LegacyNewDecFromInt(fee.Amount)
	treasury := sdk.NewCoin(fee.Denom, feeDec.Mul(pool.FeeDest.Treasury).TruncateInt())
	if treasury.IsPositive() {
		fee = fee.Sub(treasury)
		moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
		err := k.distrKeeper.FundCommunityPool(ctx, sdk.NewCoins(treasury), moduleAcc.GetAddress())
		if err != nil {
			return sdk.Coin{}, err
		}

		if !fee.IsPositive() {

			return sdk.NewCoin(fee.Denom, math.ZeroInt()), nil
		}
	}

	burner := sdk.NewCoin(fee.Denom, feeDec.Mul(pool.FeeDest.Burner).TruncateInt())
	if burner.IsPositive() {
		fee = fee.Sub(burner)
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, burnermoduletypes.ModuleName, sdk.NewCoins(burner))
		if err != nil {
			return sdk.Coin{}, err
		}
	}

	//just to make sure it's never negative (thinking that truncating the Dec might result in 0 or even negative value)
	if !fee.IsPositive() {
		//return 0 coin
		return sdk.NewCoin(fee.Denom, math.ZeroInt()), nil
	}

	return fee, nil
}

func (k msgServer) calculateSwapInputAndFee(input sdk.Coin, pool *types.LiquidityPool) (remainingInput, fee sdk.Coin) {
	feeAmount := math.LegacyNewDecFromInt(input.Amount).Mul(pool.Fee).TruncateInt()
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
