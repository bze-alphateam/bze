package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/app"
	epochstypes "github.com/bze-alphateam/bze/x/epochs/types"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
	tokenfactorytypes "github.com/bze-alphateam/bze/x/tokenfactory/types"
	tradebinkeeper "github.com/bze-alphateam/bze/x/tradebin/keeper"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// transferFeeToken sets up a factory token with a community pool, a 1% transfer fee and a deep pool against the
// native denom, so protocol fees can be paid in it.
func transferFeeToken(t *testing.T) (*app.App, sdk.Context, string, tradebintypes.LiquidityPool) {
	t.Helper()
	bApp, ctx, admin, treasury := newAdminDaoApp(t)

	token := "factory/" + admin.String() + "/fee"
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomAuthority(ctx, token, tokenfactorytypes.DenomAuthority{Admin: admin.String(), Mintable: true}))
	require.NoError(t, bApp.TokenfactoryKeeper.SetDenomConfig(ctx, token, tokenfactorytypes.DenomConfig{
		CommunityPool: treasury.String(),
		TransferFee:   math.LegacyMustNewDecFromStr("0.01"),
		GasSurcharge:  math.LegacyZeroDec(),
	}))

	nativeDenom := bApp.TradebinKeeper.GetParams(ctx).NativeDenom
	reserves := sdk.NewCoins(sdk.NewInt64Coin(token, 200_000_000_000), sdk.NewInt64Coin(nativeDenom, 200_000_000_000))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, reserves))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToModule(ctx, tokenfactorytypes.ModuleName, tradebintypes.ModuleName, reserves))
	pool := tradebintypes.LiquidityPool{
		Id:           token + "_" + nativeDenom,
		Base:         token,
		Quote:        nativeDenom,
		LpDenom:      "ulp_fee",
		ReserveBase:  reserves.AmountOf(token),
		ReserveQuote: reserves.AmountOf(nativeDenom),
		Fee:          math.LegacyNewDecWithPrec(3, 3),
		FeeDest: &tradebintypes.FeeDestination{
			Treasury:  math.LegacyZeroDec(),
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyOneDec(),
		},
	}
	bApp.TradebinKeeper.SetLiquidityPool(ctx, pool)

	return bApp, ctx, token, pool
}

// fundTokenPayer gives the payer twice the token amount the native fee converts to and returns that amount.
func fundTokenPayer(t *testing.T, bApp *app.App, ctx sdk.Context, pool tradebintypes.LiquidityPool, nativeFee sdk.Coin, payer sdk.AccAddress) sdk.Coin {
	t.Helper()
	required, err := bApp.TradebinKeeper.CalculateOptimalInputForOutput(pool, nativeFee)
	require.NoError(t, err)

	userTokens := sdk.NewCoins(sdk.NewCoin(pool.Base, required.Amount.MulRaw(2)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, userTokens))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, tokenfactorytypes.ModuleName, payer, userTokens))

	return required
}

// requireTransferFeePaid checks that ceil(required × 1%) reached txfeecollector_cp and left the payer.
func requireTransferFeePaid(t *testing.T, bApp *app.App, ctx sdk.Context, payer sdk.AccAddress, required sdk.Coin) {
	t.Helper()
	transferFee := required.Amount.ToLegacyDec().Mul(math.LegacyMustNewDecFromStr("0.01")).Ceil().TruncateInt()
	cpCollector := authtypes.NewModuleAddress(txfeecollectortypes.CpFeeCollector)
	require.Equal(t, transferFee, bApp.BankKeeper.GetBalance(ctx, cpCollector, required.Denom).Amount, "the transfer fee reaches txfeecollector_cp")
	require.Equal(t, required.Amount.MulRaw(2).Sub(required.Amount).Sub(transferFee), bApp.BankKeeper.GetBalance(ctx, payer, required.Denom).Amount)
}

// The app wires the tokenfactory keeper as tradebin's denom fee rates provider: a protocol fee paid in a factory token
// with a transfer fee pays ceil(fee in the token × rate) on top, into txfeecollector_cp.
func TestTradebin_ProtocolFeeInFactoryTokenPaysTransferFee(t *testing.T) {
	bApp, ctx, token, pool := transferFeeToken(t)

	nativeFee := sdk.NewInt64Coin(pool.Quote, 100_000_000)
	payer := sdk.AccAddress("protocol_fee_payer__")
	required := fundTokenPayer(t, bApp, ctx, pool, nativeFee, payer)

	feeCtx := ctx.WithValue(tradebinkeeper.CtxFeeDenomKey, token)
	captured, err := bApp.TradebinKeeper.CaptureAndSwapUserFee(feeCtx, payer, sdk.NewCoins(nativeFee), tradebintypes.ModuleName)
	require.NoError(t, err)
	require.Equal(t, pool.Quote, captured[0].Denom)

	requireTransferFeePaid(t, bApp, ctx, payer, required)
}

// The tokenfactory msg server registered on the app router sees the same tradebin keeper: a create-denom fee paid in
// the token through the real message handler pays the transfer fee.
func TestTokenfactory_CreateDenomFeeInFactoryTokenPaysTransferFee(t *testing.T) {
	bApp, ctx, token, pool := transferFeeToken(t)

	createDenomFee := sdk.NewInt64Coin(pool.Quote, tokenfactorytypes.DefaultCreateDenomFee)
	require.NoError(t, bApp.TokenfactoryKeeper.SetParams(ctx, tokenfactorytypes.NewParams(createDenomFee)))
	creator := sdk.AccAddress("create_denom_payer__")
	required := fundTokenPayer(t, bApp, ctx, pool, createDenomFee, creator)

	msg := &tokenfactorytypes.MsgCreateDenom{Creator: creator.String(), Subdenom: "paidintoken"}
	handler := bApp.MsgServiceRouter().Handler(msg)
	require.NotNil(t, handler)
	_, err := handler(ctx.WithValue(tradebinkeeper.CtxFeeDenomKey, token), msg)
	require.NoError(t, err)

	requireTransferFeePaid(t, bApp, ctx, creator, required)
}

// Every protocol fee captured through tradebin's fee payer pays the transfer fee when it is paid in the token: each
// message goes through its real handler on the app router, with its fee param set to the same native amount and the
// prize (when there is one) paid in the native denom, so the payer's token balance moves by the fee and the transfer
// fee only.
func TestProtocolFeesInFactoryTokenPayTransferFee(t *testing.T) {
	const prize = 1_000
	feeParam := sdk.NewInt64Coin("ubze", 100_000_000)
	zero := sdk.NewInt64Coin("ubze", 0)

	for _, c := range []struct {
		name string
		// setup sets the fee params and any state the message needs, and returns the message
		setup func(t *testing.T, bApp *app.App, ctx sdk.Context, token string, pool tradebintypes.LiquidityPool, payer sdk.AccAddress) sdk.Msg
	}{
		{
			name: "create market",
			setup: func(t *testing.T, bApp *app.App, ctx sdk.Context, token string, pool tradebintypes.LiquidityPool, payer sdk.AccAddress) sdk.Msg {
				params := bApp.TradebinKeeper.GetParams(ctx)
				params.CreateMarketFee = feeParam
				require.NoError(t, bApp.TradebinKeeper.SetParams(ctx, params))
				return &tradebintypes.MsgCreateMarket{Creator: payer.String(), Base: token, Quote: pool.Quote}
			},
		},
		{
			name: "create staking reward",
			setup: func(t *testing.T, bApp *app.App, ctx sdk.Context, token string, pool tradebintypes.LiquidityPool, payer sdk.AccAddress) sdk.Msg {
				setRewardFees(t, bApp, ctx, func(p *rewardstypes.Params) { p.CreateStakingRewardFee = feeParam })
				return &rewardstypes.MsgCreateStakingReward{
					Creator: payer.String(), PrizeAmount: "1000", PrizeDenom: pool.Quote, StakingDenom: token,
					Duration: "1", MinStake: "0", Lock: "0",
				}
			},
		},
		{
			name: "create trading reward",
			setup: func(t *testing.T, bApp *app.App, ctx sdk.Context, token string, pool tradebintypes.LiquidityPool, payer sdk.AccAddress) sdk.Msg {
				setRewardFees(t, bApp, ctx, func(p *rewardstypes.Params) { p.CreateTradingRewardFee = feeParam })
				return &rewardstypes.MsgCreateTradingReward{
					Creator: payer.String(), PrizeAmount: "1000", PrizeDenom: pool.Quote, Duration: "1", MarketId: pool.Id, Slots: "1",
				}
			},
		},
		{
			name: "create denom reward",
			setup: func(t *testing.T, bApp *app.App, ctx sdk.Context, token string, pool tradebintypes.LiquidityPool, payer sdk.AccAddress) sdk.Msg {
				setRewardFees(t, bApp, ctx, func(p *rewardstypes.Params) { p.CreateDenomRewardFee = feeParam })
				return &rewardstypes.MsgCreateDenomReward{Creator: payer.String(), Denom: token}
			},
		},
		{
			name: "denom reward schedule fee",
			setup: func(t *testing.T, bApp *app.App, ctx sdk.Context, token string, pool tradebintypes.LiquidityPool, payer sdk.AccAddress) sdk.Msg {
				setRewardFees(t, bApp, ctx, func(p *rewardstypes.Params) {
					p.AddDenomRewardScheduleFee = feeParam
					p.CreateDenomRewardPrizeFee = zero
				})
				bApp.RewardsKeeper.SetDenomReward(ctx, rewardstypes.DenomReward{StakingDenom: token, StakedAmount: math.ZeroInt()})
				return &rewardstypes.MsgCreateDenomRewardSchedule{
					Creator: payer.String(), Denom: token, PrizeDenom: pool.Quote, DailyAmount: math.NewInt(prize), Duration: "1",
				}
			},
		},
		{
			name: "denom reward prize fee",
			setup: func(t *testing.T, bApp *app.App, ctx sdk.Context, token string, pool tradebintypes.LiquidityPool, payer sdk.AccAddress) sdk.Msg {
				setRewardFees(t, bApp, ctx, func(p *rewardstypes.Params) {
					p.AddDenomRewardScheduleFee = zero
					p.CreateDenomRewardPrizeFee = feeParam
				})
				bApp.RewardsKeeper.SetDenomReward(ctx, rewardstypes.DenomReward{StakingDenom: token, StakedAmount: math.ZeroInt()})
				return &rewardstypes.MsgCreateDenomRewardSchedule{
					Creator: payer.String(), Denom: token, PrizeDenom: pool.Quote, DailyAmount: math.NewInt(prize), Duration: "1",
				}
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			bApp, ctx, token, pool := transferFeeToken(t)
			for _, id := range []string{"hour", "day"} {
				require.NoError(t, bApp.EpochKeeper.AddEpochInfo(ctx, epochstypes.EpochInfo{Identifier: id, Duration: time.Hour}))
			}
			payer := sdk.AccAddress("protocol_fee_payer__")
			msg := c.setup(t, bApp, ctx, token, pool, payer)

			prizeCoins := sdk.NewCoins(sdk.NewInt64Coin(pool.Quote, prize))
			require.NoError(t, bApp.BankKeeper.MintCoins(ctx, tokenfactorytypes.ModuleName, prizeCoins))
			require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, tokenfactorytypes.ModuleName, payer, prizeCoins))
			required := fundTokenPayer(t, bApp, ctx, pool, feeParam, payer)

			handler := bApp.MsgServiceRouter().Handler(msg)
			require.NotNil(t, handler)
			_, err := handler(ctx.WithValue(tradebinkeeper.CtxFeeDenomKey, token), msg)
			require.NoError(t, err)

			requireTransferFeePaid(t, bApp, ctx, payer, required)
		})
	}
}

// setRewardFees starts from the default rewards params (the test app runs no InitChain) and applies edit.
func setRewardFees(t *testing.T, bApp *app.App, ctx sdk.Context, edit func(*rewardstypes.Params)) {
	t.Helper()
	params := rewardstypes.DefaultParams()
	edit(&params)
	require.NoError(t, bApp.RewardsKeeper.SetParams(ctx, params))
}
