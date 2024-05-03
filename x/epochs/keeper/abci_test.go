package keeper_test

import (
	"github.com/bze-alphateam/bze/x/epochs/types"
	"time"
)

func (suite *IntegrationTestSuite) TestEpochInitialization() {
	// Define an epoch to add
	epochInfo := types.NewGenesisEpochInfo("test-epoch", time.Hour)
	suite.NoError(suite.keeper.AddEpochInfo(suite.ctx, epochInfo))

	// Move time forward to trigger epoch initialization
	suite.keeper.BeginBlocker(suite.ctx)

	// Check if epoch was initialized
	updatedEpochInfo := suite.keeper.GetEpochInfo(suite.ctx, epochInfo.Identifier)
	suite.True(updatedEpochInfo.EpochCountingStarted)
	suite.Equal(int64(1), updatedEpochInfo.CurrentEpoch)
}
