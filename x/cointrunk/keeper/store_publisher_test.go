package keeper_test

import (
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestSetAndGetPublisher() {
	// Test data
	address := sdk.AccAddress("publisher").String()
	publisher := types.Publisher{
		Name:          "Test Publisher",
		Address:       address,
		Active:        true,
		ArticlesCount: 5,
		CreatedAt:     1234567890,
		Respect:       100,
	}

	// Test SetPublisher
	suite.k.SetPublisher(suite.ctx, publisher)

	// Test GetPublisher - should find the publisher
	retrievedPublisher, found := suite.k.GetPublisher(suite.ctx, publisher.Address)
	suite.Require().True(found)
	suite.Require().Equal(publisher.Name, retrievedPublisher.Name)
	suite.Require().Equal(publisher.Address, retrievedPublisher.Address)
	suite.Require().Equal(publisher.Active, retrievedPublisher.Active)
	suite.Require().Equal(publisher.ArticlesCount, retrievedPublisher.ArticlesCount)
	suite.Require().Equal(publisher.CreatedAt, retrievedPublisher.CreatedAt)
	suite.Require().Equal(publisher.Respect, retrievedPublisher.Respect)

	// Test GetPublisher with non-existent address
	_, found = suite.k.GetPublisher(suite.ctx, "nonexistent")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestGetAllPublisher() {
	// Create multiple publishers
	publishers := []types.Publisher{
		{
			Name:          "Publisher 1",
			Address:       sdk.AccAddress("publisher1").String(),
			Active:        true,
			ArticlesCount: 10,
			CreatedAt:     1234567890,
			Respect:       50,
		},
		{
			Name:          "Publisher 2",
			Address:       sdk.AccAddress("publisher2").String(),
			Active:        false,
			ArticlesCount: 5,
			CreatedAt:     1234567891,
			Respect:       75,
		},
		{
			Name:          "Publisher 3",
			Address:       sdk.AccAddress("publisher3").String(),
			Active:        true,
			ArticlesCount: 20,
			CreatedAt:     1234567892,
			Respect:       125,
		},
	}

	// Set all publishers
	for _, publisher := range publishers {
		suite.k.SetPublisher(suite.ctx, publisher)
	}

	// Test GetAllPublisher
	allPublishers := suite.k.GetAllPublisher(suite.ctx)
	suite.Require().Len(allPublishers, 3)

	// Verify all publishers are present
	addressMap := make(map[string]types.Publisher)
	for _, publisher := range allPublishers {
		addressMap[publisher.Address] = publisher
	}

	for _, originalPublisher := range publishers {
		retrievedPublisher, exists := addressMap[originalPublisher.Address]
		suite.Require().True(exists)
		suite.Require().Equal(originalPublisher.Name, retrievedPublisher.Name)
		suite.Require().Equal(originalPublisher.Active, retrievedPublisher.Active)
		suite.Require().Equal(originalPublisher.ArticlesCount, retrievedPublisher.ArticlesCount)
		suite.Require().Equal(originalPublisher.CreatedAt, retrievedPublisher.CreatedAt)
		suite.Require().Equal(originalPublisher.Respect, retrievedPublisher.Respect)
	}
}

func (suite *IntegrationTestSuite) TestSetPublisher_UpdateExisting() {
	// Create initial publisher
	address := sdk.AccAddress("publisher").String()
	initialPublisher := types.Publisher{
		Name:          "Original Name",
		Address:       address,
		Active:        false,
		ArticlesCount: 5,
		CreatedAt:     1234567890,
		Respect:       50,
	}

	// Set initial publisher
	suite.k.SetPublisher(suite.ctx, initialPublisher)

	// Verify initial state
	retrievedPublisher, found := suite.k.GetPublisher(suite.ctx, address)
	suite.Require().True(found)
	suite.Require().Equal("Original Name", retrievedPublisher.Name)
	suite.Require().False(retrievedPublisher.Active)
	suite.Require().Equal(uint32(5), retrievedPublisher.ArticlesCount)
	suite.Require().Equal(int64(50), retrievedPublisher.Respect)

	// Update publisher
	updatedPublisher := types.Publisher{
		Name:          "Updated Name",
		Address:       address,
		Active:        true,
		ArticlesCount: 15,
		CreatedAt:     1234567890,
		Respect:       150,
	}

	suite.k.SetPublisher(suite.ctx, updatedPublisher)

	// Verify updated state
	retrievedPublisher, found = suite.k.GetPublisher(suite.ctx, address)
	suite.Require().True(found)
	suite.Require().Equal("Updated Name", retrievedPublisher.Name)
	suite.Require().True(retrievedPublisher.Active)
	suite.Require().Equal(uint32(15), retrievedPublisher.ArticlesCount)
	suite.Require().Equal(int64(150), retrievedPublisher.Respect)

	// Verify only one publisher exists
	allPublishers := suite.k.GetAllPublisher(suite.ctx)
	suite.Require().Len(allPublishers, 1)
}

func (suite *IntegrationTestSuite) TestGetAllPublisher_EmptyStore() {
	// Test GetAllPublisher with no publishers
	allPublishers := suite.k.GetAllPublisher(suite.ctx)
	suite.Require().Len(allPublishers, 0)
}

func (suite *IntegrationTestSuite) TestGetPublisher_EmptyAddress() {
	// Test GetPublisher with empty address
	_, found := suite.k.GetPublisher(suite.ctx, "")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestSetPublisher_EmptyFields() {
	// Test setting publisher with empty optional fields
	address := sdk.AccAddress("publisher").String()
	publisher := types.Publisher{
		Name:          "",
		Address:       address,
		Active:        false,
		ArticlesCount: 0,
		CreatedAt:     0,
		Respect:       0,
	}

	suite.k.SetPublisher(suite.ctx, publisher)

	// Verify it was stored correctly
	retrievedPublisher, found := suite.k.GetPublisher(suite.ctx, address)
	suite.Require().True(found)
	suite.Require().Equal("", retrievedPublisher.Name)
	suite.Require().False(retrievedPublisher.Active)
	suite.Require().Equal(uint32(0), retrievedPublisher.ArticlesCount)
	suite.Require().Equal(int64(0), retrievedPublisher.CreatedAt)
	suite.Require().Equal(int64(0), retrievedPublisher.Respect)
}
