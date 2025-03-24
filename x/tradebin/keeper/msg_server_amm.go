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
		ReserveBase:  0,
		ReserveQuote: 0,
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
		&types.LpCreatedEvent{
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

	lp.ReserveBase = baseCoin.Amount.Uint64()
	lp.ReserveQuote = quoteCoin.Amount.Uint64()

	return
}

func (k msgServer) getProvidedReserves(baseDenom, quoteDenom, baseAmt, quoteAmt string) (baseCoin, quoteCoin sdk.Coin, err error) {
	baseCoin, err = sdk.ParseCoinNormalized(fmt.Sprintf("%s%s", baseAmt, baseDenom))
	if err != nil {
		return
	}

	quoteCoin, err = sdk.ParseCoinNormalized(fmt.Sprintf("%s%s", quoteAmt, quoteDenom))
	if err != nil {
		return
	}

	if !baseCoin.IsValid() || !quoteCoin.IsValid() {
		err = errors.Wrap(sdkerrors.ErrInvalidCoins, "invalid reserve")
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
