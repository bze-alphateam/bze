package keeper

import (
	"context"
	"cosmossdk.io/errors"
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (k msgServer) CreateLiquidityPool(goCtx context.Context, msg *types.MsgCreateLiquidityPool) (*types.MsgCreateLiquidityPoolResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	creatorAcc := msg.GetCreatorAcc()
	if creatorAcc == nil {
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, "invalid creator address")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	base, quote, poolId := k.CreatePoolId(msg.Base, msg.Quote)
	err := k.validateMarketAssets(ctx, base, quote)
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
		ReserveBase:  sdk.ZeroInt(),
		ReserveQuote: sdk.ZeroInt(),
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
		k.Logger(ctx).Error(err.Error())
	}

	return &types.MsgCreateLiquidityPoolResponse{
		Id: poolId,
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
	product := sdk.NewDecFromInt(baseCoin.Amount).Mul(sdk.NewDecFromInt(quoteCoin.Amount))
	// Calculate sqrt(base * quote)
	sqrtProduct, err := product.ApproxSqrt()
	if err != nil {
		return
	}

	// Scale by a multiplier to preserve precision
	multiplier := sdk.NewDec(10).Power(sharesScaleExponent)
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

func (k msgServer) getProvidedReserves(baseDenom, quoteDenom string, baseAmt, quoteAmt sdk.Int) (baseCoin, quoteCoin sdk.Coin, err error) {
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
	if !feeDest.Burner.Add(feeDest.Treasury).Add(feeDest.Liquidity).Add(feeDest.Providers).Equal(sdk.NewDecWithPrec(1, 0)) {
		return types.ErrInvalidFeeDestination
	}

	//do not allow any of the destinations to be negative
	if feeDest.Treasury.IsNegative() || feeDest.Burner.IsNegative() || feeDest.Providers.IsNegative() || feeDest.Liquidity.IsNegative() {
		return types.ErrNegativeFeeDestination
	}

	return nil
}

func (k msgServer) validateFee(fee *sdk.Dec) error {
	if !fee.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "fee must be positive")
	}

	if fee.LT(sdk.NewDecWithPrec(1, 3)) {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "fee must be at least 0.001 (0.1%)")
	}

	if fee.GT(sdk.NewDecWithPrec(5, 2)) {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "fee must be at most 0.05 (5%)")
	}

	return nil
}

func (k msgServer) parseValidPoolFees(msg *types.MsgCreateLiquidityPool) (fee sdk.Dec, feeDest types.FeeDestination, err error) {
	fee, err = sdk.NewDecFromStr(msg.Fee)
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

func (k msgServer) AddLiquidity(goCtx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	creatorAcc := msg.GetCreatorAcc()
	if creatorAcc == nil {
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, "invalid creator address")
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
		k.Logger(ctx).Error(err.Error())
	}

	return &types.MsgAddLiquidityResponse{
		MintedAmount: minted.Amount,
	}, nil
}

func (k msgServer) mintDepositLpTokens(ctx sdk.Context, baseAmount, quoteAmount, poolBaseReserve, poolQuoteReserve *sdk.Int, lp *types.LiquidityPool) (mintedLp sdk.Coin, err error) {
	lpSupply := k.bankKeeper.GetSupply(ctx, lp.GetLpDenom())
	if !lpSupply.IsPositive() {
		return mintedLp, errors.Wrapf(types.ErrInvalidDenom, "could not find supply for pool %s", lp.GetId())
	}

	baseRatio := sdk.NewDecFromInt(*baseAmount).Quo(sdk.NewDecFromInt(*poolBaseReserve))
	quoteRatio := sdk.NewDecFromInt(*quoteAmount).Quo(sdk.NewDecFromInt(*poolQuoteReserve))

	var mintRatio sdk.Dec
	if baseRatio.LT(quoteRatio) {
		mintRatio = baseRatio
	} else {
		mintRatio = quoteRatio
	}

	tokensToMint := mintRatio.Mul(sdk.NewDecFromInt(lpSupply.Amount)).TruncateInt()
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

func (k msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	creatorAcc := msg.GetCreatorAcc()
	if creatorAcc == nil {
		return nil, errors.Wrapf(sdkerrors.ErrUnauthorized, "invalid creator address")
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
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAcc, types.ModuleName, sdk.NewCoins(toRemove))
	if err != nil {
		return nil, errors.Wrap(err, "failed to send LP Tokens to module account")
	}

	userShare := sdk.NewDecFromInt(toRemove.Amount).Quo(sdk.NewDecFromInt(lpSupply.Amount))
	base := sdk.NewDecFromInt(pool.ReserveBase).Mul(userShare).TruncateInt()
	quote := sdk.NewDecFromInt(pool.ReserveQuote).Mul(userShare).TruncateInt()

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
		k.Logger(ctx).Error(err.Error())
	}

	return &types.MsgRemoveLiquidityResponse{
		Base:  base,
		Quote: quote,
	}, nil
}
