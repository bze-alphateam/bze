package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

// expectKeptPage expects a validator pass to read page after afterDenom and keep every coin of it (deep liquidity,
// too small to swap).
func (suite *IntegrationTestSuite) expectKeptPage(moduleAddr sdk.AccAddress, afterDenom string, limit int, page sdk.Coins, last string) *gomock.Call {
	call := suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), moduleAddr, afterDenom, limit).Return(page, last).Times(1)
	for _, c := range page {
		suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), c.Denom).Return(false).Times(1)
		suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), c).Return(false).Times(1)
		suite.tradeMock.EXPECT().HasDeepLiquidityWithNativeDenom(gomock.Any(), c.Denom).Return(true).Times(1)
	}

	return call
}

func (suite *IntegrationTestSuite) setMaxBalanceIterations(n uint64) {
	params := suite.k.GetParams(suite.ctx)
	params.MaxBalanceIterations = n
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
}

// A full page moves the cursor to its last denom, the next pass starts strictly after it, and a short page (the end of
// the balances) sends the pass after it back to the first denom.
func (suite *IntegrationTestSuite) TestCollectorCursor_MovesAndWraps() {
	suite.setMaxBalanceIterations(2)
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	suite.accountMock.EXPECT().GetModuleAddress(types.ModuleName).Return(moduleAddr).AnyTimes()
	a, b, c := sdk.NewInt64Coin("ibc/A", 1), sdk.NewInt64Coin("ibc/B", 2), sdk.NewInt64Coin("ibc/C", 3)

	gomock.InOrder(
		suite.expectKeptPage(moduleAddr, "", 2, sdk.NewCoins(a, b), "ibc/B"),
		suite.expectKeptPage(moduleAddr, "ibc/B", 2, sdk.NewCoins(c), "ibc/C"),
		suite.expectKeptPage(moduleAddr, "", 2, sdk.NewCoins(a, b), "ibc/B"),
	)

	for range 3 {
		suite.Require().NoError(suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx))
	}
}

// A cursor past the last denom (its balance was spent, or the last page was exactly full) gives an empty page: the same
// pass reads again from the first denom.
func (suite *IntegrationTestSuite) TestCollectorCursor_EmptyPageAfterCursorRereadsFromStart() {
	suite.setMaxBalanceIterations(2)
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	suite.accountMock.EXPECT().GetModuleAddress(types.ModuleName).Return(moduleAddr).AnyTimes()
	a, b := sdk.NewInt64Coin("ibc/A", 1), sdk.NewInt64Coin("ibc/B", 2)

	gomock.InOrder(
		suite.expectKeptPage(moduleAddr, "", 2, sdk.NewCoins(a, b), "ibc/B"),
		suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), moduleAddr, "ibc/B", 2).Return(nil, "").Times(1),
		suite.expectKeptPage(moduleAddr, "", 2, sdk.NewCoins(a), "ibc/A"),
		suite.expectKeptPage(moduleAddr, "", 2, sdk.NewCoins(a), "ibc/A"),
	)

	for range 3 {
		suite.Require().NoError(suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx))
	}
}

// Each collector has its own cursor: the validator pass moving its cursor doesn't move the CP collector's.
func (suite *IntegrationTestSuite) TestCollectorCursor_OnePerCollector() {
	suite.setMaxBalanceIterations(1)
	moduleAddr := sdk.AccAddress("txfeecollector_addr")
	suite.accountMock.EXPECT().GetModuleAddress(types.ModuleName).Return(moduleAddr).AnyTimes()
	a := sdk.NewInt64Coin("ibc/A", 1)
	suite.expectKeptPage(moduleAddr, "", 1, sdk.NewCoins(a), "ibc/A")
	suite.Require().NoError(suite.k.ConvertCollectedFeesToNativeDenom(suite.ctx))

	// the CP pass still starts from the first denom
	suite.accountMock.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpModuleAddr).AnyTimes()
	suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), cpModuleAddr, "", 1).Return(nil, "").Times(1)
	suite.Require().NoError(suite.k.ConvertCommunityPoolFees(suite.ctx))
}
