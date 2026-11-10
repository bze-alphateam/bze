package keeper_test

import (
	"errors"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// The message servers check the amount form themselves, without relying on ValidateBasic.

var nonCanonicalOrderAmounts = []string{"0100", "00", "+10", "0x10", "0o10", "0b10", "1_0", "-0", "", " ", " 10", "10 "}

func (suite *IntegrationTestSuite) TestMsgCreateOrder_AmountFormat() {
	creator := sdk.AccAddress("addr1_______________").String()
	for _, amount := range nonCanonicalOrderAmounts {
		suite.Run(amount, func() {
			// no market is set: the amount is refused before the market lookup
			_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
				Creator:   creator,
				MarketId:  getMarketId(),
				OrderType: types.OrderTypeBuy,
				Amount:    amount,
				Price:     "1",
			})
			suite.Require().ErrorIs(err, types.ErrInvalidOrderAmount, "amount %q must be refused", amount)
		})
	}

	_, err := suite.msgServer.CreateOrder(suite.ctx, &types.MsgCreateOrder{
		Creator:   creator,
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    "100",
		Price:     "1",
	})
	suite.Require().ErrorIs(err, types.ErrMarketNotFound, "a canonical amount goes on to the market lookup")
}

func (suite *IntegrationTestSuite) TestMsgFillOrders_AmountFormat() {
	suite.k.SetMarket(suite.ctx, market)
	creator := sdk.AccAddress("addr2_______________")
	errEscrow := errors.New("escrow reached")
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), creator, types.ModuleName, gomock.Any()).Return(errEscrow).AnyTimes()

	for _, amount := range nonCanonicalOrderAmounts {
		suite.Run(amount, func() {
			_, err := suite.msgServer.FillOrders(suite.ctx, &types.MsgFillOrders{
				Creator:   creator.String(),
				MarketId:  getMarketId(),
				OrderType: types.OrderTypeSell,
				Orders:    []*types.FillOrderItem{{Amount: amount, Price: "1"}},
			})
			suite.Require().ErrorIs(err, types.ErrInvalidOrderAmount, "amount %q must be refused", amount)
		})
	}

	_, err := suite.msgServer.FillOrders(suite.ctx, &types.MsgFillOrders{
		Creator:   creator.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Orders:    []*types.FillOrderItem{{Amount: "100", Price: "1"}},
	})
	suite.Require().ErrorIs(err, errEscrow, "a canonical amount goes on to the escrow")
}
