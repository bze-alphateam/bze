package keeper_test

import (
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
