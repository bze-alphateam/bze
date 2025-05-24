package keeper_test

import (
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestAcceptDomain_ValidRequest_NewDomain() {
	authority := suite.k.GetAuthority()
	domain := "example.com"

	msg := &types.MsgAcceptDomain{
		Authority: authority,
		Domain:    domain,
		Active:    true,
	}

	res, err := suite.msgServer.AcceptDomain(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify domain was created
	acceptedDomain, found := suite.k.GetAcceptedDomain(suite.ctx, domain)
	suite.Require().True(found)
	suite.Require().Equal(domain, acceptedDomain.Domain)
	suite.Require().True(acceptedDomain.Active)
}

func (suite *IntegrationTestSuite) TestAcceptDomain_ValidRequest_UpdateExisting() {
	authority := suite.k.GetAuthority()
	domain := "update-test.com"

	// Create initial domain
	initialDomain := types.AcceptedDomain{
		Domain: domain,
		Active: false,
	}
	suite.k.SetAcceptedDomain(suite.ctx, initialDomain)

	// Update domain
	msg := &types.MsgAcceptDomain{
		Authority: authority,
		Domain:    domain,
		Active:    true,
	}

	res, err := suite.msgServer.AcceptDomain(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify domain was updated
	acceptedDomain, found := suite.k.GetAcceptedDomain(suite.ctx, domain)
	suite.Require().True(found)
	suite.Require().Equal(domain, acceptedDomain.Domain)
	suite.Require().True(acceptedDomain.Active)
}

func (suite *IntegrationTestSuite) TestAcceptDomain_InvalidAuthority() {
	invalidAuthority := sdk.AccAddress("invalid").String()

	msg := &types.MsgAcceptDomain{
		Authority: invalidAuthority,
		Domain:    "example.com",
		Active:    true,
	}

	res, err := suite.msgServer.AcceptDomain(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestAcceptDomain_EmptyDomain() {
	authority := suite.k.GetAuthority()

	msg := &types.MsgAcceptDomain{
		Authority: authority,
		Domain:    "",
		Active:    true,
	}

	res, err := suite.msgServer.AcceptDomain(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify empty domain was stored (even though it might not be practical)
	acceptedDomain, found := suite.k.GetAcceptedDomain(suite.ctx, "")
	suite.Require().True(found)
	suite.Require().Equal("", acceptedDomain.Domain)
	suite.Require().True(acceptedDomain.Active)
}

func (suite *IntegrationTestSuite) TestAcceptDomain_ToggleStatus() {
	authority := suite.k.GetAuthority()
	domain := "toggle-test.com"

	// First, activate domain
	msg1 := &types.MsgAcceptDomain{
		Authority: authority,
		Domain:    domain,
		Active:    true,
	}

	res, err := suite.msgServer.AcceptDomain(suite.ctx, msg1)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify active
	acceptedDomain, found := suite.k.GetAcceptedDomain(suite.ctx, domain)
	suite.Require().True(found)
	suite.Require().True(acceptedDomain.Active)

	// Then, deactivate domain
	msg2 := &types.MsgAcceptDomain{
		Authority: authority,
		Domain:    domain,
		Active:    false,
	}

	res, err = suite.msgServer.AcceptDomain(suite.ctx, msg2)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify deactivated
	acceptedDomain, found = suite.k.GetAcceptedDomain(suite.ctx, domain)
	suite.Require().True(found)
	suite.Require().False(acceptedDomain.Active)
}

func (suite *IntegrationTestSuite) TestSavePublisher_ValidRequest_NewPublisher() {
	authority := suite.k.GetAuthority()
	address := sdk.AccAddress("publisher").String()

	// Set a proper block time for the test
	header := suite.ctx.BlockHeader()
	header.Time = header.Time.Add(1000000) // Add some time to avoid negative Unix timestamp
	suite.ctx = suite.ctx.WithBlockHeader(header)

	msg := &types.MsgSavePublisher{
		Authority: authority,
		Address:   address,
		Name:      "Test Publisher",
		Active:    true,
	}

	res, err := suite.msgServer.SavePublisher(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify publisher was created
	publisher, found := suite.k.GetPublisher(suite.ctx, address)
	suite.Require().True(found)
	suite.Require().Equal(address, publisher.Address)
	suite.Require().Equal("Test Publisher", publisher.Name)
	suite.Require().True(publisher.Active)
	suite.Require().Equal(uint32(0), publisher.ArticlesCount)
	suite.Require().Equal(int64(0), publisher.Respect)
	suite.Require().NotZero(publisher.CreatedAt) // Just check it's not zero
}

func (suite *IntegrationTestSuite) TestSavePublisher_ValidRequest_UpdateExisting() {
	authority := suite.k.GetAuthority()
	address := sdk.AccAddress("publisher").String()

	// Create initial publisher
	initialPublisher := types.Publisher{
		Name:          "Original Name",
		Address:       address,
		Active:        false,
		ArticlesCount: 10,
		CreatedAt:     1234567890,
		Respect:       50,
	}
	suite.k.SetPublisher(suite.ctx, initialPublisher)

	// Update publisher
	msg := &types.MsgSavePublisher{
		Authority: authority,
		Address:   address,
		Name:      "Updated Name",
		Active:    true,
	}

	res, err := suite.msgServer.SavePublisher(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify publisher was updated
	publisher, found := suite.k.GetPublisher(suite.ctx, address)
	suite.Require().True(found)
	suite.Require().Equal(address, publisher.Address)
	suite.Require().Equal("Updated Name", publisher.Name)
	suite.Require().True(publisher.Active)
	// These should be preserved from original
	suite.Require().Equal(uint32(10), publisher.ArticlesCount)
	suite.Require().Equal(int64(1234567890), publisher.CreatedAt)
	suite.Require().Equal(int64(50), publisher.Respect)
}

func (suite *IntegrationTestSuite) TestSavePublisher_InvalidAuthority() {
	invalidAuthority := sdk.AccAddress("invalid").String()

	msg := &types.MsgSavePublisher{
		Authority: invalidAuthority,
		Address:   sdk.AccAddress("publisher").String(),
		Name:      "Test Publisher",
		Active:    true,
	}

	res, err := suite.msgServer.SavePublisher(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestSavePublisher_EmptyName() {
	authority := suite.k.GetAuthority()
	address := sdk.AccAddress("publisher").String()

	msg := &types.MsgSavePublisher{
		Authority: authority,
		Address:   address,
		Name:      "",
		Active:    true,
	}

	res, err := suite.msgServer.SavePublisher(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify publisher was created with empty name
	publisher, found := suite.k.GetPublisher(suite.ctx, address)
	suite.Require().True(found)
	suite.Require().Equal("", publisher.Name)
	suite.Require().True(publisher.Active)
}

func (suite *IntegrationTestSuite) TestSavePublisher_EmptyAddress() {
	authority := suite.k.GetAuthority()

	msg := &types.MsgSavePublisher{
		Authority: authority,
		Address:   "",
		Name:      "Empty Address Publisher",
		Active:    true,
	}

	res, err := suite.msgServer.SavePublisher(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify publisher was created with empty address
	publisher, found := suite.k.GetPublisher(suite.ctx, "")
	suite.Require().True(found)
	suite.Require().Equal("", publisher.Address)
	suite.Require().Equal("Empty Address Publisher", publisher.Name)
}

func (suite *IntegrationTestSuite) TestSavePublisher_ToggleStatus() {
	authority := suite.k.GetAuthority()
	address := sdk.AccAddress("publisher").String()

	// First, create active publisher
	msg1 := &types.MsgSavePublisher{
		Authority: authority,
		Address:   address,
		Name:      "Toggle Publisher",
		Active:    true,
	}

	res, err := suite.msgServer.SavePublisher(suite.ctx, msg1)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify active
	publisher, found := suite.k.GetPublisher(suite.ctx, address)
	suite.Require().True(found)
	suite.Require().True(publisher.Active)

	// Then, deactivate publisher
	msg2 := &types.MsgSavePublisher{
		Authority: authority,
		Address:   address,
		Name:      "Toggle Publisher Updated",
		Active:    false,
	}

	res, err = suite.msgServer.SavePublisher(suite.ctx, msg2)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify deactivated and name updated
	publisher, found = suite.k.GetPublisher(suite.ctx, address)
	suite.Require().True(found)
	suite.Require().False(publisher.Active)
	suite.Require().Equal("Toggle Publisher Updated", publisher.Name)
}
