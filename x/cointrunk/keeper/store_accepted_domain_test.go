package keeper_test

import (
	"github.com/bze-alphateam/bze/x/cointrunk/types"
)

func (suite *IntegrationTestSuite) TestSetAndGetAcceptedDomain() {
	// Test data
	acceptedDomain := types.AcceptedDomain{
		Domain: "example.com",
		Active: true,
	}

	// Test SetAcceptedDomain
	suite.k.SetAcceptedDomain(suite.ctx, acceptedDomain)

	// Test GetAcceptedDomain - should find the domain
	retrievedDomain, found := suite.k.GetAcceptedDomain(suite.ctx, acceptedDomain.Domain)
	suite.Require().True(found)
	suite.Require().Equal(acceptedDomain.Domain, retrievedDomain.Domain)
	suite.Require().Equal(acceptedDomain.Active, retrievedDomain.Active)

	// Test GetAcceptedDomain with non-existent domain
	_, found = suite.k.GetAcceptedDomain(suite.ctx, "nonexistent.com")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestGetAllAcceptedDomain() {
	// Create multiple accepted domains
	domains := []types.AcceptedDomain{
		{
			Domain: "example1.com",
			Active: true,
		},
		{
			Domain: "example2.org",
			Active: false,
		},
		{
			Domain: "test.net",
			Active: true,
		},
	}

	// Set all domains
	for _, domain := range domains {
		suite.k.SetAcceptedDomain(suite.ctx, domain)
	}

	// Test GetAllAcceptedDomain
	allDomains := suite.k.GetAllAcceptedDomain(suite.ctx)
	suite.Require().Len(allDomains, 3)

	// Verify all domains are present
	domainMap := make(map[string]types.AcceptedDomain)
	for _, domain := range allDomains {
		domainMap[domain.Domain] = domain
	}

	for _, originalDomain := range domains {
		retrievedDomain, exists := domainMap[originalDomain.Domain]
		suite.Require().True(exists)
		suite.Require().Equal(originalDomain.Active, retrievedDomain.Active)
	}
}

func (suite *IntegrationTestSuite) TestSetAcceptedDomain_UpdateExisting() {
	// Create initial domain
	domain := "update-test.com"
	initialDomain := types.AcceptedDomain{
		Domain: domain,
		Active: false,
	}

	// Set initial domain
	suite.k.SetAcceptedDomain(suite.ctx, initialDomain)

	// Verify initial state
	retrievedDomain, found := suite.k.GetAcceptedDomain(suite.ctx, domain)
	suite.Require().True(found)
	suite.Require().Equal(domain, retrievedDomain.Domain)
	suite.Require().False(retrievedDomain.Active)

	// Update domain status
	updatedDomain := types.AcceptedDomain{
		Domain: domain,
		Active: true,
	}

	suite.k.SetAcceptedDomain(suite.ctx, updatedDomain)

	// Verify updated state
	retrievedDomain, found = suite.k.GetAcceptedDomain(suite.ctx, domain)
	suite.Require().True(found)
	suite.Require().Equal(domain, retrievedDomain.Domain)
	suite.Require().True(retrievedDomain.Active)

	// Verify only one domain exists
	allDomains := suite.k.GetAllAcceptedDomain(suite.ctx)
	suite.Require().Len(allDomains, 1)
}

func (suite *IntegrationTestSuite) TestGetAllAcceptedDomain_EmptyStore() {
	// Test GetAllAcceptedDomain with no domains
	allDomains := suite.k.GetAllAcceptedDomain(suite.ctx)
	suite.Require().Len(allDomains, 0)
}

func (suite *IntegrationTestSuite) TestGetAcceptedDomain_EmptyDomain() {
	// Test GetAcceptedDomain with empty domain
	_, found := suite.k.GetAcceptedDomain(suite.ctx, "")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestSetAcceptedDomain_ActiveInactive() {
	// Test setting both active and inactive domains
	activeDomain := types.AcceptedDomain{
		Domain: "active.com",
		Active: true,
	}

	inactiveDomain := types.AcceptedDomain{
		Domain: "inactive.com",
		Active: false,
	}

	// Set both domains
	suite.k.SetAcceptedDomain(suite.ctx, activeDomain)
	suite.k.SetAcceptedDomain(suite.ctx, inactiveDomain)

	// Verify active domain
	retrievedActive, found := suite.k.GetAcceptedDomain(suite.ctx, "active.com")
	suite.Require().True(found)
	suite.Require().True(retrievedActive.Active)

	// Verify inactive domain
	retrievedInactive, found := suite.k.GetAcceptedDomain(suite.ctx, "inactive.com")
	suite.Require().True(found)
	suite.Require().False(retrievedInactive.Active)

	// Verify both exist in GetAll
	allDomains := suite.k.GetAllAcceptedDomain(suite.ctx)
	suite.Require().Len(allDomains, 2)
}

func (suite *IntegrationTestSuite) TestSetAcceptedDomain_SpecialCharacters() {
	// Test domain with special characters and edge cases
	domains := []types.AcceptedDomain{
		{
			Domain: "sub.domain.example.com",
			Active: true,
		},
		{
			Domain: "test-domain.co.uk",
			Active: false,
		},
		{
			Domain: "123numbers.org",
			Active: true,
		},
	}

	// Set all domains
	for _, domain := range domains {
		suite.k.SetAcceptedDomain(suite.ctx, domain)
	}

	// Verify all domains can be retrieved
	for _, originalDomain := range domains {
		retrievedDomain, found := suite.k.GetAcceptedDomain(suite.ctx, originalDomain.Domain)
		suite.Require().True(found)
		suite.Require().Equal(originalDomain.Domain, retrievedDomain.Domain)
		suite.Require().Equal(originalDomain.Active, retrievedDomain.Active)
	}

	// Verify all exist in GetAll
	allDomains := suite.k.GetAllAcceptedDomain(suite.ctx)
	suite.Require().Len(allDomains, 3)
}
