package keeper_test

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// takerGasForFills rests fills sell orders of 2 units at price 1, each from a distinct owner, and
// returns the gas one MsgCreateOrder consumes to fill all of them. The bank keeper is mocked, so the
// figure covers the tradebin store work (and the gas the message server burns itself) but not the bank
// sends.
func (suite *IntegrationTestSuite) takerGasForFills(fills int) uint64 {
	suite.SetupTest()
	suite.k.SetMarket(suite.ctx, market)
	suite.setMaxFillsPerOrder(uint64(fills))
	suite.restSells(fills, "2", "1")
	suite.allowBank()
	taker := sdk.AccAddress("addr2_______________")

	// only the taker message runs on the fresh gas meter
	gasCtx := suite.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	_, err := suite.msgServer.CreateOrder(gasCtx, &types.MsgCreateOrder{
		Creator:   taker.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeBuy,
		Amount:    fmt.Sprintf("%d", 2*fills),
		Price:     "1",
	})
	suite.Require().NoError(err)

	// every maker was filled in that same call
	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
	suite.Require().Empty(suite.k.GetAllAggregatedOrder(suite.ctx))
	suite.Require().Len(suite.k.GetAllHistoryOrder(suite.ctx), fills)

	return gasCtx.GasMeter().GasConsumed()
}

// TestMsgCreateOrder_TakerGasPerFill logs the gas a taker pays for 1, 10 and 100 fills. The matching
// runs in the transaction, so its work is charged to the transaction's gas meter and grows with the
// number of fills, which maxFillsPerOrder bounds.
func (suite *IntegrationTestSuite) TestMsgCreateOrder_TakerGasPerFill() {
	gas := map[int]uint64{}
	for _, fills := range []int{1, 10, 100} {
		gas[fills] = suite.takerGasForFills(fills)
		suite.T().Logf("gas consumed by one MsgCreateOrder filling %d resting orders: %d", fills, gas[fills])
	}
	suite.T().Logf("marginal gas per fill (1 -> 100 fills): %d", (gas[100]-gas[1])/99)

	suite.Require().Greater(gas[10], gas[1])
	suite.Require().Greater(gas[100], gas[10])
	// well below 10% of the 1,000,000,000 block gas limit
	suite.Require().Less(gas[100], uint64(100_000_000))
}
