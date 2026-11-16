package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	txfeecollectormoduletypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

var orderFeeDestinations = []struct {
	param  string
	module string
}{
	{v2types.FeeDestinationBurnerModule, burnermoduletypes.ModuleName},
	{v2types.FeeDestinationCommunityPool, txfeecollectormoduletypes.CpFeeCollector},
}

// The maker fee goes straight to the burner module with maker_fee_destination burner, and to the community pool fee
// collector with community_pool.
func (suite *IntegrationTestSuite) TestOrderFeeDestination_Maker() {
	for _, tc := range orderFeeDestinations {
		suite.Run(tc.param, func() {
			suite.SetupTest()
			params := suite.k.GetParams(suite.ctx)
			params.MakerFeeDestination = tc.param
			suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
			suite.k.SetMarket(suite.ctx, market)

			addr1 := sdk.AccAddress("addr1_______________")
			makerFee := sdk.NewCoins(params.MarketMakerFee)
			suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, makerFee).Return(nil).Times(1)
			suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, tc.module, makerFee).Return(nil).Times(1)
			suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr1, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 2000000))).Return(nil).Times(1)

			_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
				Amount:    "1000000",
				Price:     "2",
				MarketId:  getMarketId(),
				OrderType: types.OrderTypeBuy,
				Creator:   addr1.String(),
			})
			suite.Require().NoError(err)
		})
	}
}

// The taker fee goes straight to the burner module with taker_fee_destination burner, and to the community pool fee
// collector with community_pool.
func (suite *IntegrationTestSuite) TestOrderFeeDestination_Taker() {
	for _, tc := range orderFeeDestinations {
		suite.Run(tc.param, func() {
			suite.SetupTest()
			allPrices, addr1, addr2 := suite.msgOrderFillSetup(types.OrderTypeSell)
			params := suite.k.GetParams(suite.ctx)
			params.TakerFeeDestination = tc.param
			suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

			takerFee := sdk.NewCoins(params.MarketTakerFee)
			escrow := sdk.NewCoins(sdk.NewInt64Coin(market.Quote, 500000))
			suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, escrow).Return(nil).Times(1)
			suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr2, types.ModuleName, takerFee).Return(nil).Times(1)
			suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, tc.module, takerFee).Return(nil).Times(1)
			suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr1, escrow).Return(nil).Times(1)
			suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, addr2, sdk.NewCoins(sdk.NewInt64Coin(market.Base, 500000))).Return(nil).Times(1)

			_, err := suite.msgServer.FillOrders(suite.ctx, &types.MsgFillOrders{
				Creator:   addr2.String(),
				MarketId:  getMarketId(),
				OrderType: types.OrderTypeSell,
				Orders:    []*types.FillOrderItem{{Amount: "500000", Price: allPrices[0]}},
			})
			suite.Require().NoError(err)
		})
	}
}
