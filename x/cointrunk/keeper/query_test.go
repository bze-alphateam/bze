package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestAcceptedDomain_ValidRequest() {
	// Create test accepted domains
	domains := []types.AcceptedDomain{
		{Domain: "example1.com", Active: true},
		{Domain: "example2.org", Active: false},
	}

	for _, domain := range domains {
		suite.k.SetAcceptedDomain(suite.ctx, domain)
	}

	req := &types.QueryAcceptedDomainRequest{}
	res, err := suite.k.AcceptedDomain(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.AcceptedDomain, 2)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestAcceptedDomain_NilRequest() {
	res, err := suite.k.AcceptedDomain(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestAcceptedDomain_WithPagination() {
	// Create multiple domains
	for i := 0; i < 5; i++ {
		domain := types.AcceptedDomain{
			Domain: "example" + string(rune('1'+i)) + ".com",
			Active: i%2 == 0,
		}
		suite.k.SetAcceptedDomain(suite.ctx, domain)
	}

	req := &types.QueryAcceptedDomainRequest{
		Pagination: &query.PageRequest{
			Limit: 3,
		},
	}
	res, err := suite.k.AcceptedDomain(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.AcceptedDomain, 3)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestAllAnonArticlesCounters_ValidRequest() {
	// Since we can't easily create anon articles counters without going through the full flow,
	// we'll test with empty store
	req := &types.QueryAllAnonArticlesCountersRequest{}
	res, err := suite.k.AllAnonArticlesCounters(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.AnonArticlesCounters, 0)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestAllAnonArticlesCounters_NilRequest() {
	res, err := suite.k.AllAnonArticlesCounters(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestAllAnonArticlesCounters_WithPagination() {
	req := &types.QueryAllAnonArticlesCountersRequest{
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	}
	res, err := suite.k.AllAnonArticlesCounters(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestAllArticles_ValidRequest() {
	// Create test articles
	articles := []types.Article{
		{
			Id:        0,
			Title:     "Article 1",
			Url:       "https://example.com/1",
			Publisher: "publisher1",
			Paid:      false,
			CreatedAt: 1234567890,
		},
		{
			Id:        0,
			Title:     "Article 2",
			Url:       "https://example.com/2",
			Publisher: "publisher2",
			Paid:      true,
			CreatedAt: 1234567891,
		},
	}

	for _, article := range articles {
		suite.k.SetArticle(suite.ctx, article)
	}

	req := &types.QueryAllArticlesRequest{}
	res, err := suite.k.AllArticles(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.Article, 2)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestAllArticles_NilRequest() {
	res, err := suite.k.AllArticles(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestAllArticles_WithPagination() {
	// Create multiple articles
	for i := 0; i < 5; i++ {
		article := types.Article{
			Id:        0, // Will be auto-incremented
			Title:     "Article " + string(rune('1'+i)),
			Url:       "https://example.com/" + string(rune('1'+i)),
			Publisher: "publisher" + string(rune('1'+i)),
			Paid:      i%2 == 0,
			CreatedAt: 1234567890 + int64(i),
		}
		suite.k.SetArticle(suite.ctx, article)
	}

	req := &types.QueryAllArticlesRequest{
		Pagination: &query.PageRequest{
			Limit: 3,
		},
	}
	res, err := suite.k.AllArticles(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.Article, 3)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestPublisher_ValidRequest() {
	// Create test publisher
	address := sdk.AccAddress("publisher").String()
	publisher := types.Publisher{
		Name:          "Test Publisher",
		Address:       address,
		Active:        true,
		ArticlesCount: 5,
		CreatedAt:     1234567890,
		Respect:       "100",
	}

	suite.k.SetPublisher(suite.ctx, publisher)

	req := &types.QueryPublisherRequest{
		Address: address,
	}
	res, err := suite.k.Publisher(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NotNil(res.Publisher)
	suite.Require().Equal(publisher.Name, res.Publisher.Name)
	suite.Require().Equal(publisher.Address, res.Publisher.Address)
	suite.Require().Equal(publisher.Active, res.Publisher.Active)
	suite.Require().Equal(publisher.ArticlesCount, res.Publisher.ArticlesCount)
	suite.Require().Equal(publisher.CreatedAt, res.Publisher.CreatedAt)
	suite.Require().Equal(publisher.Respect, res.Publisher.Respect)
}

func (suite *IntegrationTestSuite) TestPublisher_NilRequest() {
	res, err := suite.k.Publisher(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestPublisher_NotFound() {
	req := &types.QueryPublisherRequest{
		Address: "nonexistent",
	}
	res, err := suite.k.Publisher(suite.ctx, req)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.NotFound, status.Code(err))
	suite.Require().Contains(err.Error(), "not found")
}

func (suite *IntegrationTestSuite) TestPublishers_ValidRequest() {
	// Create test publishers
	publishers := []types.Publisher{
		{
			Name:          "Publisher 1",
			Address:       sdk.AccAddress("publisher1").String(),
			Active:        true,
			ArticlesCount: 10,
			CreatedAt:     1234567890,
			Respect:       "50",
		},
		{
			Name:          "Publisher 2",
			Address:       sdk.AccAddress("publisher2").String(),
			Active:        false,
			ArticlesCount: 5,
			CreatedAt:     1234567891,
			Respect:       "75",
		},
	}

	for _, publisher := range publishers {
		suite.k.SetPublisher(suite.ctx, publisher)
	}

	req := &types.QueryPublishersRequest{}
	res, err := suite.k.Publishers(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.Publisher, 2)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestPublishers_NilRequest() {
	res, err := suite.k.Publishers(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestPublishers_WithPagination() {
	// Create multiple publishers
	for i := 0; i < 5; i++ {
		publisher := types.Publisher{
			Name:          "Publisher " + string(rune('1'+i)),
			Address:       sdk.AccAddress("publisher" + string(rune('1'+i))).String(),
			Active:        i%2 == 0,
			ArticlesCount: uint32(i * 5),
			CreatedAt:     1234567890 + int64(i),
			Respect:       fmt.Sprintf("%d", i*10),
		}
		suite.k.SetPublisher(suite.ctx, publisher)
	}

	req := &types.QueryPublishersRequest{
		Pagination: &query.PageRequest{
			Limit: 3,
		},
	}
	res, err := suite.k.Publishers(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.Publisher, 3)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestAllArticles_EmptyStore() {
	req := &types.QueryAllArticlesRequest{}
	res, err := suite.k.AllArticles(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.Article, 0)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestPublishers_EmptyStore() {
	req := &types.QueryPublishersRequest{}
	res, err := suite.k.Publishers(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.Publisher, 0)
	suite.Require().NotNil(res.Pagination)
}
