package keeper_test

import (
	"errors"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

func (suite *IntegrationTestSuite) expectMarketCreationSupply() {
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), "uatom").Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), denomBze).Return(true).Times(1)
}

// The market creation fee goes through the price-checked path: with the preferred fee denom's pool in range it is
// paid in that denom and swapped.
func (suite *IntegrationTestSuite) TestCreateMarket_FeeSwappedWhenPoolPriceInRange() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000_000, 50_000_000_000_000)
	marketFee := suite.k.CreateMarketFee(ctx)
	required, err := suite.k.CalculateOptimalInputForOutput(pool, marketFee)
	suite.Require().NoError(err)

	suite.expectMarketCreationSupply()
	suite.bankMock.EXPECT().SpendableCoins(gomock.Any(), safePayer).Return(sdk.NewCoins(sdk.NewCoin(denomStake, math.NewInt(900_000_000_000_000)))).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), safePayer, types.ModuleName, sdk.NewCoins(required)).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	_, err = suite.msgServer.CreateMarket(suite.safeFeeCtx(ctx), &types.MsgCreateMarket{Creator: safePayer.String(), Base: "uatom", Quote: denomBze})
	suite.Require().NoError(err)
}

// With the preferred fee denom's pool price away from its moving average, the market creation fee is charged in the
// native denom.
func (suite *IntegrationTestSuite) TestCreateMarket_FeeInNativeWhenPoolPriceDrifted() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000_000, 50_000_000_000_000)
	suite.safeMoveQuote(ctx, pool, math.NewInt(70_000_000_000_000))
	marketFee := sdk.NewCoins(suite.k.CreateMarketFee(ctx))

	suite.expectMarketCreationSupply()
	// no SpendableCoins and no payment in the preferred denom: a call would fail the test
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), safePayer, types.ModuleName, marketFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), marketFee).Return(nil).Times(1)

	_, err := suite.msgServer.CreateMarket(suite.safeFeeCtx(ctx), &types.MsgCreateMarket{Creator: safePayer.String(), Base: "uatom", Quote: denomBze})
	suite.Require().NoError(err)
	_, found := suite.k.GetMarket(ctx, "uatom", denomBze)
	suite.Require().True(found)
}

// The pool creation fee takes the same price-checked path: with the preferred fee denom's pool price away from its
// moving average, the fee is charged in the native denom.
func (suite *IntegrationTestSuite) TestCreateLiquidityPool_FeeInNativeWhenPoolPriceDrifted() {
	ctx := suite.safeCtx()
	pool := suite.safeSeedPool(ctx, denomStake, 100_000_000_000_000, 50_000_000_000_000)
	suite.safeMoveQuote(ctx, pool, math.NewInt(70_000_000_000_000))
	poolFee := sdk.NewCoins(suite.k.CreateMarketFee(ctx))
	reservesErr := errors.New("stop after the fee")

	suite.bankMock.EXPECT().HasSupply(gomock.Any(), "abc").Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), "def").Return(true).Times(1)
	// the fee is taken in the native denom, with no payment in the preferred denom
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), safePayer, types.ModuleName, poolFee).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), poolFee).Return(nil).Times(1)
	// the initial reserves fail, which ends the message right after the fee
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), safePayer, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("abc", 123), sdk.NewInt64Coin("def", 345))).
		Return(reservesErr).
		Times(1)

	_, err := suite.msgServer.CreateLiquidityPool(suite.safeFeeCtx(ctx), &types.MsgCreateLiquidityPool{
		Base:         "abc",
		Quote:        "def",
		Creator:      safePayer.String(),
		Fee:          "0.002",
		FeeDest:      getFeeDestinationString("0.25", "0.5", "0.25"),
		InitialBase:  math.NewInt(123),
		InitialQuote: math.NewInt(345),
	})
	suite.Require().ErrorIs(err, reservesErr)
}
