package keeper_test

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const priceLevelGasDescriptor = "price level spam protection"

// gasByDescriptor is a gas meter that also sums the gas consumed under each descriptor
type gasByDescriptor struct {
	storetypes.GasMeter
	consumed map[string]uint64
}

func newGasByDescriptor(meter storetypes.GasMeter) *gasByDescriptor {
	return &gasByDescriptor{GasMeter: meter, consumed: map[string]uint64{}}
}

func (g *gasByDescriptor) ConsumeGas(amount storetypes.Gas, descriptor string) {
	g.consumed[descriptor] += amount
	g.GasMeter.ConsumeGas(amount, descriptor)
}

// seedPriceLevelOwners writes a sell aggregate at price 1 counting owners distinct owners (the message
// server reads only the count)
func (suite *IntegrationTestSuite) seedPriceLevelOwners(owners uint64) {
	suite.k.SetAggregatedOrder(suite.ctx, types.AggregatedOrder{
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Amount:    "1000",
		Price:     "1",
		Owners:    owners,
	})
}

// priceLevelGas runs a sell MsgCreateOrder of creator at price 1 and returns the gas burned by the
// price level surcharge
func (suite *IntegrationTestSuite) priceLevelGas(creator sdk.AccAddress, amount string) uint64 {
	meter := newGasByDescriptor(storetypes.NewInfiniteGasMeter())
	_, err := suite.msgServer.CreateOrder(suite.ctx.WithGasMeter(meter), &types.MsgCreateOrder{
		Creator:   creator.String(),
		MarketId:  getMarketId(),
		OrderType: types.OrderTypeSell,
		Amount:    amount,
		Price:     "1",
	})
	suite.Require().NoError(err)

	return meter.consumed[priceLevelGasDescriptor]
}

// A new owner joining a level of more than priceLevelOwnersWindow owners pays
// (owners - window) * priceLevelExtraGas, counted before the order joins.
func (suite *IntegrationTestSuite) TestMsgCreateOrder_PriceLevelGas_NewOwner() {
	window := uint64(500)
	for _, tc := range []struct {
		name   string
		owners uint64
		seed   bool
		want   uint64
	}{
		{name: "no level", seed: false, want: 0},
		{name: "0 owners", owners: 0, seed: true, want: 0},
		{name: "window - 1", owners: window - 1, seed: true, want: 0},
		{name: "window", owners: window, seed: true, want: 0},
		{name: "window + 1", owners: window + 1, seed: true, want: 25_000},
		{name: "window + 10", owners: window + 10, seed: true, want: 250_000},
		{name: "window + 100", owners: window + 100, seed: true, want: 2_500_000},
	} {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.k.SetMarket(suite.ctx, market)
			suite.allowBank()
			if tc.seed {
				suite.seedPriceLevelOwners(tc.owners)
			}

			suite.Require().Equal(tc.want, suite.priceLevelGas(sdk.AccAddress("addr1_______________"), "10"))
		})
	}
}

// The window and the gas per owner are the params.
func (suite *IntegrationTestSuite) TestMsgCreateOrder_PriceLevelGas_Params() {
	suite.k.SetMarket(suite.ctx, market)
	suite.allowBank()
	params := suite.k.GetParams(suite.ctx)
	params.PriceLevelOwnersWindow = 2
	params.PriceLevelExtraGas = 1_000_000
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
	suite.seedPriceLevelOwners(5)

	suite.Require().Equal(uint64(3_000_000), suite.priceLevelGas(sdk.AccAddress("addr1_______________"), "10"))
}

// An owner who already rests an order at the level tops it up without the surcharge.
func (suite *IntegrationTestSuite) TestMsgCreateOrder_PriceLevelGas_ExistingOwnerPaysNothing() {
	suite.k.SetMarket(suite.ctx, market)
	suite.allowBank()
	owner := sdk.AccAddress("addr1_______________")
	suite.k.SaveOrder(suite.ctx, withId(limitOrder(types.OrderTypeSell, "1000", "1", owner), orderId(0)))
	suite.k.SetOrderCounter(suite.ctx, 1)
	suite.seedPriceLevelOwners(600)

	suite.Require().Zero(suite.priceLevelGas(owner, "10"))

	// the top-up merged into the resting order, the count is unchanged
	orders := suite.k.GetAllOrder(suite.ctx)
	suite.Require().Len(orders, 1)
	suite.Require().Equal("1010", orders[0].Amount)
	agg, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, "1")
	suite.Require().True(found)
	suite.Require().Equal(uint64(600), agg.Owners)
}

// A taker is charged for the level of its own side, where its remainder may rest, even when the
// opposite side fills it in full.
func (suite *IntegrationTestSuite) TestMsgCreateOrder_PriceLevelGas_TakerWithDeepOwnSide() {
	suite.k.SetMarket(suite.ctx, market)
	suite.allowBank()
	suite.seedRestingOrder(withId(limitOrder(types.OrderTypeBuy, "10", "1", makerAddr(0)), orderId(0)))
	suite.k.SetOrderCounter(suite.ctx, 1)
	suite.seedPriceLevelOwners(510)

	suite.Require().Equal(uint64(250_000), suite.priceLevelGas(sdk.AccAddress("addr1_______________"), "10"))

	// the taker was filled in full, it did not rest
	_, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeBuy, "1")
	suite.Require().False(found)
	agg, found := suite.k.GetAggregatedOrder(suite.ctx, getMarketId(), types.OrderTypeSell, "1")
	suite.Require().True(found)
	suite.Require().Equal(uint64(510), agg.Owners)
}

// The surcharge is consumed before the fee and the escrow: when it runs the transaction out of gas,
// nothing is sent (the bank mock expects no call) and nothing is written.
func (suite *IntegrationTestSuite) TestMsgCreateOrder_PriceLevelGas_OutOfGasBeforeFeeAndEscrow() {
	suite.k.SetMarket(suite.ctx, market)
	suite.seedPriceLevelOwners(600)
	ctx := suite.ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000))

	suite.Require().PanicsWithValue(storetypes.ErrorOutOfGas{Descriptor: priceLevelGasDescriptor}, func() {
		_, _ = suite.msgServer.CreateOrder(ctx, &types.MsgCreateOrder{
			Creator:   sdk.AccAddress("addr1_______________").String(),
			MarketId:  getMarketId(),
			OrderType: types.OrderTypeSell,
			Amount:    "10",
			Price:     "1",
		})
	})

	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
	suite.Require().Zero(suite.k.GetOrderCounter(suite.ctx))
}

// A surcharge beyond the uint64 range is the maximum gas, it does not wrap to a small amount.
func (suite *IntegrationTestSuite) TestMsgCreateOrder_PriceLevelGas_DoesNotWrap() {
	suite.k.SetMarket(suite.ctx, market)
	params := suite.k.GetParams(suite.ctx)
	params.PriceLevelOwnersWindow = 1
	params.PriceLevelExtraGas = 1 << 62
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
	suite.seedPriceLevelOwners(1 + 8)
	// a wrapped surcharge would let the order through
	suite.allowBank()

	suite.Require().PanicsWithValue(storetypes.ErrorGasOverflow{Descriptor: priceLevelGasDescriptor}, func() {
		_, _ = suite.msgServer.CreateOrder(suite.ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000_000)), &types.MsgCreateOrder{
			Creator:   sdk.AccAddress("addr1_______________").String(),
			MarketId:  getMarketId(),
			OrderType: types.OrderTypeSell,
			Amount:    "10",
			Price:     "1",
		})
	})
	suite.Require().Empty(suite.k.GetAllOrder(suite.ctx))
}
