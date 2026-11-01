package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
)

// Migrate4to5 gives maxFillsPerOrder its default when the stored field 11 is 0 and keeps any other
// stored value.
func (suite *IntegrationTestSuite) TestMigrate4to5_MaxFillsPerOrder() {
	for _, tc := range []struct{ stored, want uint64 }{{0, 500}, {1, 1}, {750, 750}} {
		stored, want := tc.stored, tc.want
		suite.SetupTest()
		params := suite.k.GetParams(suite.ctx)
		params.MaxFillsPerOrder = stored
		suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

		migrator := keeper.NewMigrator(suite.k, nil, suite.bankMock)
		suite.Require().NoError(migrator.Migrate4to5(suite.ctx))

		got := suite.k.GetParams(suite.ctx)
		suite.Require().Equal(want, got.MaxFillsPerOrder, "stored %d", stored)
		params.MaxFillsPerOrder = want
		suite.Require().Equal(params, got)
	}
}
