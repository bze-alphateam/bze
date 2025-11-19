package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestStoreBurn_SetAndGetBurnedCoins() {
	// Test data
	burnedCoins := types.BurnedCoins{
		Burned: "1000utoken,500stake",
		Height: "12345",
	}

	// Test SetBurnedCoins
	suite.k.SetBurnedCoins(suite.ctx, burnedCoins)

	// Test GetBurnedCoins - should find the burned coins
	retrievedCoins, found := suite.k.GetBurnedCoins(suite.ctx, burnedCoins.Height)
	suite.Require().True(found)
	suite.Require().Equal(burnedCoins.Height, retrievedCoins.Height)
	suite.Require().Equal(burnedCoins.Burned, retrievedCoins.Burned)

	// Test GetBurnedCoins with non-existent height
	_, found = suite.k.GetBurnedCoins(suite.ctx, "nonexistent")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreBurn_GetAllBurnedCoins() {
	// Create multiple burned coins entries
	burnedCoinsEntries := []types.BurnedCoins{
		{
			Burned: "1000utoken",
			Height: "100",
		},
		{
			Burned: "2000utoken,500stake",
			Height: "200",
		},
		{
			Burned: "3000utoken,1000stake,100atom",
			Height: "300",
		},
	}

	// Set all burned coins entries
	for _, entry := range burnedCoinsEntries {
		suite.k.SetBurnedCoins(suite.ctx, entry)
	}

	// Test GetAllBurnedCoins
	allBurnedCoins := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Len(allBurnedCoins, 3)

	// Verify all entries are present
	heightMap := make(map[string]types.BurnedCoins)
	for _, entry := range allBurnedCoins {
		heightMap[entry.Height] = entry
	}

	for _, originalEntry := range burnedCoinsEntries {
		retrievedEntry, exists := heightMap[originalEntry.Height]
		suite.Require().True(exists)
		suite.Require().Equal(originalEntry.Burned, retrievedEntry.Burned)
	}
}

func (suite *IntegrationTestSuite) TestStoreBurn_SaveBurnedCoins_NewEntry() {
	// Create coins to burn
	coins := sdk.NewCoins(
		sdk.NewInt64Coin("utoken", 1000),
		sdk.NewInt64Coin("stake", 500),
	)

	// Mock the block height
	header := suite.ctx.BlockHeader()
	header.Height = 12345
	suite.ctx = suite.ctx.WithBlockHeader(header)

	// Test SaveBurnedCoins with new entry
	err := suite.k.SaveBurnedCoins(suite.ctx, coins)
	suite.Require().NoError(err)

	// Verify the entry was created
	burnedCoins, found := suite.k.GetBurnedCoins(suite.ctx, "12345")
	suite.Require().True(found)
	suite.Require().Equal("12345", burnedCoins.Height)
	suite.Require().Equal("500stake,1000utoken", burnedCoins.Burned)
}

func (suite *IntegrationTestSuite) TestStoreBurn_SaveBurnedCoins_ExistingEntry() {
	// Set up initial burned coins entry
	initialBurnedCoins := types.BurnedCoins{
		Burned: "1000utoken,500stake",
		Height: "12345",
	}
	suite.k.SetBurnedCoins(suite.ctx, initialBurnedCoins)

	// Mock the block height
	header := suite.ctx.BlockHeader()
	header.Height = 12345
	suite.ctx = suite.ctx.WithBlockHeader(header)

	// Create additional coins to burn
	additionalCoins := sdk.NewCoins(
		sdk.NewInt64Coin("utoken", 500),
		sdk.NewInt64Coin("atom", 200),
	)

	// Test SaveBurnedCoins with existing entry
	err := suite.k.SaveBurnedCoins(suite.ctx, additionalCoins)
	suite.Require().NoError(err)

	// Verify the entry was updated correctly
	burnedCoins, found := suite.k.GetBurnedCoins(suite.ctx, "12345")
	suite.Require().True(found)
	suite.Require().Equal("12345", burnedCoins.Height)

	// Parse and verify the coins were added correctly
	totalBurned, err := sdk.ParseCoinsNormalized(burnedCoins.Burned)
	suite.Require().NoError(err)

	expectedCoins := sdk.NewCoins(
		sdk.NewInt64Coin("atom", 200),
		sdk.NewInt64Coin("stake", 500),
		sdk.NewInt64Coin("utoken", 1500), // 1000 + 500
	)

	suite.Require().True(totalBurned.Equal(expectedCoins))
}

func (suite *IntegrationTestSuite) TestStoreBurn_SaveBurnedCoins_EmptyCoins() {
	// Create empty coins
	emptyCoins := sdk.NewCoins()

	// Mock the block height
	header := suite.ctx.BlockHeader()
	header.Height = 12345
	suite.ctx = suite.ctx.WithBlockHeader(header)

	// Test SaveBurnedCoins with empty coins
	err := suite.k.SaveBurnedCoins(suite.ctx, emptyCoins)
	suite.Require().NoError(err)

	// Verify the entry was created with empty string
	burnedCoins, found := suite.k.GetBurnedCoins(suite.ctx, "12345")
	suite.Require().True(found)
	suite.Require().Equal("12345", burnedCoins.Height)
	suite.Require().Equal("", burnedCoins.Burned)
}

func (suite *IntegrationTestSuite) TestStoreBurn_SaveBurnedCoins_MultipleHeights() {
	// Test saving burned coins at different heights
	testCases := []struct {
		height int64
		coins  sdk.Coins
	}{
		{
			height: 100,
			coins:  sdk.NewCoins(sdk.NewInt64Coin("utoken", 1000)),
		},
		{
			height: 200,
			coins:  sdk.NewCoins(sdk.NewInt64Coin("stake", 500)),
		},
		{
			height: 300,
			coins:  sdk.NewCoins(sdk.NewInt64Coin("atom", 200)),
		},
	}

	for _, tc := range testCases {
		// Mock the block height
		header := suite.ctx.BlockHeader()
		header.Height = tc.height
		ctx := suite.ctx.WithBlockHeader(header)

		// Save burned coins
		err := suite.k.SaveBurnedCoins(ctx, tc.coins)
		suite.Require().NoError(err)
	}

	// Verify all entries exist with correct heights
	heights := []string{"100", "200", "300"}
	expectedBurned := []string{"1000utoken", "500stake", "200atom"}

	for i, height := range heights {
		burnedCoins, found := suite.k.GetBurnedCoins(suite.ctx, height)
		suite.Require().True(found)
		suite.Require().Equal(height, burnedCoins.Height)
		suite.Require().Equal(expectedBurned[i], burnedCoins.Burned)
	}

	// Verify GetAllBurnedCoins returns all entries
	allBurnedCoins := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Len(allBurnedCoins, 3)
}

func (suite *IntegrationTestSuite) TestStoreBurn_SaveBurnedCoins_LargeHeight() {
	// Test with very large height number
	largeHeight := "999999999999999999"
	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1000000))

	// Mock a large block height
	header := suite.ctx.BlockHeader()
	header.Height = 999999999999999999
	suite.ctx = suite.ctx.WithBlockHeader(header)

	// Test SaveBurnedCoins with large height
	err := suite.k.SaveBurnedCoins(suite.ctx, coins)
	suite.Require().NoError(err)

	// Verify the entry was created with correct large height
	burnedCoins, found := suite.k.GetBurnedCoins(suite.ctx, largeHeight)
	suite.Require().True(found)
	suite.Require().Equal(largeHeight, burnedCoins.Height)
	suite.Require().Equal("1000000ubze", burnedCoins.Burned)
}
