package keeper_test

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ppaPool = "abc/xyz"

func ppaDec(s string) math.LegacyDec {
	return math.LegacyMustNewDecFromStr(s)
}

func ppaPoolWith(base, quote math.Int) types.LiquidityPool {
	return types.LiquidityPool{
		Id:           ppaPool,
		Base:         "abc",
		Quote:        "xyz",
		Fee:          math.LegacyZeroDec(),
		FeeDest:      &types.FeeDestination{},
		ReserveBase:  base,
		ReserveQuote: quote,
	}
}

func ppaPoolInts(base, quote int64) types.LiquidityPool {
	return ppaPoolWith(math.NewInt(base), math.NewInt(quote))
}

func (suite *IntegrationTestSuite) ppaHalfLife(h uint64) {
	params := suite.k.GetParams(suite.ctx)
	params.PoolPriceHalfLifeBlocks = h
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
}

func (suite *IntegrationTestSuite) ppaAt(height int64) sdk.Context {
	return suite.ctx.WithBlockHeight(height)
}

func (suite *IntegrationTestSuite) ppaRecord(ctx sdk.Context) types.PoolPriceAverage {
	record, found := suite.k.GetPoolPriceAverageRecord(ctx, ppaPool)
	suite.Require().True(found)

	return record
}

func (suite *IntegrationTestSuite) TestPoolPriceAverage_SeedOnFirstSave() {
	ctx := suite.ppaAt(10)
	_, found := suite.k.GetPoolPriceAverage(ctx, ppaPool)
	suite.Require().False(found)

	// quote / base = 200 / 100
	suite.k.SetLiquidityPool(ctx, ppaPoolInts(100, 200))

	record := suite.ppaRecord(ctx)
	suite.Require().Equal(ppaDec("2"), record.AvgPrice)
	suite.Require().Equal(int64(10), record.LastHeight)
	suite.Require().Equal(ppaPool, record.PoolId)
}

func (suite *IntegrationTestSuite) TestPoolPriceAverage_SecondSaveInTheSameBlockChangesNothing() {
	ctx := suite.ppaAt(10)
	suite.k.SetLiquidityPool(ctx, ppaPoolInts(100, 200))
	suite.k.SetLiquidityPool(ctx, ppaPoolInts(100, 900))

	record := suite.ppaRecord(ctx)
	suite.Require().Equal(ppaDec("2"), record.AvgPrice)
	suite.Require().Equal(int64(10), record.LastHeight)
}

// Half life 1 halves the gap to the start-of-block price with every block.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_FoldsAcrossBlocks() {
	suite.ppaHalfLife(1)
	// block 10: seed at price 2
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 200))
	// block 11: the first save finds the stored price 2 (equal to the average) and moves the pool to price 4
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolInts(100, 400))
	record := suite.ppaRecord(suite.ppaAt(11))
	suite.Require().Equal(ppaDec("2"), record.AvgPrice)
	suite.Require().Equal(int64(11), record.LastHeight)

	// block 12: stored price 4, average 2, one block: 4 + (2 - 4) / 2 = 3
	suite.k.SetLiquidityPool(suite.ppaAt(12), ppaPoolInts(100, 800))
	record = suite.ppaRecord(suite.ppaAt(12))
	suite.Require().Equal(ppaDec("3"), record.AvgPrice)
	suite.Require().Equal(int64(12), record.LastHeight)

	// block 14: stored price 8, average 3, two blocks: 8 + (3 - 8) / 4 = 6.75
	suite.k.SetLiquidityPool(suite.ppaAt(14), ppaPoolInts(100, 800))
	record = suite.ppaRecord(suite.ppaAt(14))
	suite.Require().Equal(ppaDec("6.75"), record.AvgPrice)
	suite.Require().Equal(int64(14), record.LastHeight)
}

// Half life 2: after two blocks the gap is halved, 2^-(2/2).
func (suite *IntegrationTestSuite) TestPoolPriceAverage_FoldsWithAnIrrationalDecay() {
	suite.ppaHalfLife(2)
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 200))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolInts(100, 400))
	// stored price 4, average 2 at height 11; at 13: 4 + (2 - 4) / 2 = 3
	suite.k.SetLiquidityPool(suite.ppaAt(13), ppaPoolInts(100, 400))

	got := suite.ppaRecord(suite.ppaAt(13)).AvgPrice
	suite.Require().True(got.Sub(ppaDec("3")).Abs().LT(ppaDec("0.000000000001")), "got %s", got)
}

func (suite *IntegrationTestSuite) TestPoolPriceAverage_LongGapEqualsTheStoredPrice() {
	suite.ppaHalfLife(1)
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 200))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolInts(100, 800))

	// 21 blocks later is more than 20 half lives: the average is the stored price, 8
	ctx := suite.ppaAt(32)
	avg, found := suite.k.GetPoolPriceAverage(ctx, ppaPool)
	suite.Require().True(found)
	suite.Require().Equal(ppaDec("8"), avg)

	suite.k.SetLiquidityPool(ctx, ppaPoolInts(100, 800))
	suite.Require().Equal(ppaDec("8"), suite.ppaRecord(ctx).AvgPrice)
}

func (suite *IntegrationTestSuite) TestPoolPriceAverage_HugeGapAndHalfLifeDoNotOverflow() {
	suite.ppaHalfLife(100000)
	suite.k.SetLiquidityPool(suite.ppaAt(1), ppaPoolInts(100, 200))
	suite.k.SetLiquidityPool(suite.ppaAt(2), ppaPoolInts(100, 800))

	// far beyond 20 half lives
	ctx := suite.ppaAt(1 << 50)
	avg, found := suite.k.GetPoolPriceAverage(ctx, ppaPool)
	suite.Require().True(found)
	suite.Require().Equal(ppaDec("8"), avg)
}

// The first save of a block folds the stored (start-of-block) price, never the reserves being saved.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_CurrentBlockReservesNeverMoveTheAverage() {
	suite.ppaHalfLife(1)
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 200))

	ctx := suite.ppaAt(11)
	// the pool is saved with a price of 1000 in this very block; the stored price is still 2
	suite.k.SetLiquidityPool(ctx, ppaPoolInts(1, 1000))
	suite.Require().Equal(ppaDec("2"), suite.ppaRecord(ctx).AvgPrice)

	avg, _ := suite.k.GetPoolPriceAverage(ctx, ppaPool)
	suite.Require().Equal(ppaDec("2"), avg)

	// a later save in the same block changes nothing either
	suite.k.SetLiquidityPool(ctx, ppaPoolInts(1, 5000))
	suite.Require().Equal(ppaDec("2"), suite.ppaRecord(ctx).AvgPrice)
}

func (suite *IntegrationTestSuite) TestPoolPriceAverage_ZeroReserveStoredPoolReseeds() {
	suite.ppaHalfLife(1)
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 200))
	// block 11 empties the pool (the stored price 2 is still defined here)
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolInts(0, 0))
	suite.Require().Equal(ppaDec("2"), suite.ppaRecord(suite.ppaAt(11)).AvgPrice)

	// block 12: the stored pool has no price; the average restarts from the pool being saved, 300 / 100
	ctx := suite.ppaAt(12)
	suite.k.SetLiquidityPool(ctx, ppaPoolInts(100, 300))
	record := suite.ppaRecord(ctx)
	suite.Require().Equal(ppaDec("3"), record.AvgPrice)
	suite.Require().Equal(int64(12), record.LastHeight)
}

// A price that rounds to 0 at 18 decimals is no price: it is never stored as an average.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_TinyPriceIsNoPrice() {
	tiny := ppaPoolWith(math.NewIntWithDecimal(1, 30), math.NewInt(1))

	// no record is created from it
	suite.k.SetLiquidityPool(suite.ppaAt(10), tiny)
	_, found := suite.k.GetPoolPriceAverageRecord(suite.ppaAt(10), ppaPool)
	suite.Require().False(found)

	// an existing record is not zeroed either: seed at 5, move the pool to the tiny price, then restart from a real one
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolInts(100, 500))
	suite.k.SetLiquidityPool(suite.ppaAt(12), tiny)
	suite.Require().Equal(ppaDec("5"), suite.ppaRecord(suite.ppaAt(12)).AvgPrice)

	ctx := suite.ppaAt(13)
	suite.k.SetLiquidityPool(ctx, ppaPoolInts(100, 700))
	record := suite.ppaRecord(ctx)
	suite.Require().Equal(ppaDec("7"), record.AvgPrice)
	suite.Require().True(record.AvgPrice.IsPositive())

	// saving another tiny price on top leaves the record alone
	ctx = suite.ppaAt(14)
	suite.k.SetLiquidityPool(ctx, tiny)
	suite.Require().Equal(ppaDec("7"), suite.ppaRecord(ctx).AvgPrice)
}

// The read-only view equals what the next save stores and never writes.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_ViewDoesNotWriteAndMatchesTheNextSave() {
	suite.ppaHalfLife(1)
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 200))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolInts(100, 400))

	ctx := suite.ppaAt(12)
	before := suite.ppaRecord(ctx)
	view, found := suite.k.GetPoolPriceAverage(ctx, ppaPool)
	suite.Require().True(found)
	suite.Require().Equal(ppaDec("3"), view)
	suite.Require().Equal(before, suite.ppaRecord(ctx), "the view must not write")

	suite.k.SetLiquidityPool(ctx, ppaPoolInts(100, 9000))
	suite.Require().Equal(view, suite.ppaRecord(ctx).AvgPrice)
}

func (suite *IntegrationTestSuite) TestPoolPriceAverage_UnknownPool() {
	_, found := suite.k.GetPoolPriceAverage(suite.ppaAt(5), "nope/nope")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestPoolPriceAverage_Query() {
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 250))

	res, err := suite.k.PoolPriceAverage(suite.ppaAt(10), &types.QueryPoolPriceAverageRequest{PoolId: ppaPool})
	suite.Require().NoError(err)
	suite.Require().Equal(ppaDec("2.5"), res.AvgPrice)
	suite.Require().Equal(int64(10), res.LastHeight)

	_, err = suite.k.PoolPriceAverage(suite.ppaAt(10), &types.QueryPoolPriceAverageRequest{PoolId: "unknown/pool"})
	suite.Require().Equal(codes.NotFound, status.Code(err))

	_, err = suite.k.PoolPriceAverage(suite.ppaAt(10), &types.QueryPoolPriceAverageRequest{})
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))

	_, err = suite.k.PoolPriceAverage(suite.ppaAt(10), nil)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
}

// Migrate4to5 sets the three params and seeds every existing pool without rewriting the pools.
func (suite *IntegrationTestSuite) TestMigrate4to5_PoolPriceAverage() {
	other := ppaPoolInts(50, 400)
	other.Id = "def/xyz"
	empty := ppaPoolInts(0, 0)
	empty.Id = "empty/pool"
	for _, p := range []types.LiquidityPool{ppaPoolInts(100, 200), other, empty} {
		suite.k.SetLiquidityPool(suite.ppaAt(3), p)
	}

	// state as consensus version 4 left it: no records and none of the new params
	store := suite.ctx.KVStore(suite.storeKey)
	var keys [][]byte
	it := storetypes.KVStorePrefixIterator(store, types.PoolPriceAveragePrefix())
	for ; it.Valid(); it.Next() {
		keys = append(keys, it.Key())
	}
	suite.Require().NoError(it.Close())
	for _, key := range keys {
		store.Delete(key)
	}
	params := suite.k.GetParams(suite.ctx)
	params.PoolPriceHalfLifeBlocks = 0
	params.FeeMaxPriceDrift = math.LegacyDec{}
	params.ModuleMaxPriceDrift = math.LegacyDec{}
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	ctx := suite.ppaAt(77)
	suite.Require().NoError(keeper.NewMigrator(suite.k, nil, suite.bankMock).Migrate4to5(ctx))

	got := suite.k.GetParams(ctx)
	suite.Require().Equal(uint64(20), got.PoolPriceHalfLifeBlocks)
	suite.Require().Equal(ppaDec("0.1"), got.FeeMaxPriceDrift)
	suite.Require().Equal(ppaDec("0.03"), got.ModuleMaxPriceDrift)
	suite.Require().NoError(got.Validate())

	rec := suite.ppaRecord(ctx)
	suite.Require().Equal(ppaDec("2"), rec.AvgPrice)
	suite.Require().Equal(int64(77), rec.LastHeight)
	rec2, found := suite.k.GetPoolPriceAverageRecord(ctx, "def/xyz")
	suite.Require().True(found)
	suite.Require().Equal(ppaDec("8"), rec2.AvgPrice)
	suite.Require().Equal(int64(77), rec2.LastHeight)
	// a pool without a price has no record
	_, found = suite.k.GetPoolPriceAverageRecord(ctx, "empty/pool")
	suite.Require().False(found)
}

// Records are never exported; a genesis import of the pools seeds them through the save hook.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_SeededByPoolSaves() {
	ctx := suite.ppaAt(4)
	suite.k.SetLiquidityPool(ctx, ppaPoolInts(100, 200))
	other := ppaPoolInts(10, 5)
	other.Id = "def/xyz"
	suite.k.SetLiquidityPool(ctx, other)

	suite.Require().Equal(ppaDec("2"), suite.ppaRecord(ctx).AvgPrice)
	rec, found := suite.k.GetPoolPriceAverageRecord(ctx, "def/xyz")
	suite.Require().True(found)
	suite.Require().Equal(ppaDec("0.5"), rec.AvgPrice)
}

// A priced pool is emptied and the empty pool is saved again in a later block: nothing to fold or restart
// from, so the record stays as it was.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_UnpricedSavesLeaveTheRecordUntouched() {
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 200))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolInts(0, 0))
	before := suite.ppaRecord(suite.ppaAt(11))
	suite.Require().Equal(ppaDec("2"), before.AvgPrice)
	suite.Require().Equal(int64(11), before.LastHeight)

	suite.k.SetLiquidityPool(suite.ppaAt(12), ppaPoolInts(0, 0))
	suite.Require().Equal(before, suite.ppaRecord(suite.ppaAt(12)))
}

// Half life 2, one block: the average moves by 1 - 2^(-1/2) of the gap. The expected value is computed
// independently: 4 + (2 - 4) * 0.70710678118654752440 = 2.5857864376269049512.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_FoldsWithARealRoot() {
	suite.ppaHalfLife(2)
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 200))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolInts(100, 400))
	suite.k.SetLiquidityPool(suite.ppaAt(12), ppaPoolInts(100, 400))

	got := suite.ppaRecord(suite.ppaAt(12)).AvgPrice
	want := ppaDec("2.585786437626904951")
	suite.Require().True(got.Sub(want).Abs().LTE(ppaDec("0.0000000000000001")), "got %s", got)
}

// Exactly 20 half lives still fold: with half life 1 the old gap of 6 shrinks by 2^-20 (2^-20 = 9.5367431640625e-7):
// 8 - 6 / 1048576 = 7.9999942779541015625. One block more and the average is the stored price.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_TwentyHalfLifeBoundary() {
	suite.ppaHalfLife(1)
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 200))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolInts(100, 800))

	atLimit, found := suite.k.GetPoolPriceAverage(suite.ppaAt(31), ppaPool)
	suite.Require().True(found)
	// the exact value has 19 decimals; allow the one unit of rounding of the 18-decimal arithmetic
	suite.Require().True(atLimit.Sub(ppaDec("7.999994277954101562")).Abs().LTE(ppaDec("0.000000000000000002")), "got %s", atLimit)
	suite.Require().NotEqual(ppaDec("8"), atLimit)

	pastLimit, found := suite.k.GetPoolPriceAverage(suite.ppaAt(32), ppaPool)
	suite.Require().True(found)
	suite.Require().Equal(ppaDec("8"), pastLimit)
}

// With the default half life of 20, 20 blocks land halfway between the average and the stored price.
func (suite *IntegrationTestSuite) TestPoolPriceAverage_DefaultHalfLifeHalfway() {
	suite.Require().Equal(uint64(20), suite.k.GetParams(suite.ctx).PoolPriceHalfLifeBlocks)
	suite.k.SetLiquidityPool(suite.ppaAt(10), ppaPoolInts(100, 200))
	suite.k.SetLiquidityPool(suite.ppaAt(11), ppaPoolInts(100, 400))

	got, found := suite.k.GetPoolPriceAverage(suite.ppaAt(31), ppaPool)
	suite.Require().True(found)
	suite.Require().True(got.Sub(ppaDec("3")).Abs().LT(ppaDec("0.000000000001")), "got %s", got)
}
