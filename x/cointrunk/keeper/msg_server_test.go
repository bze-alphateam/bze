package keeper_test

import (
	"cosmossdk.io/math"
	"errors"
	"fmt"
	"strings"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestAddArticle_ValidRequest_PublishedByActivePublisher() {
	// Set up active publisher
	publisher := types.Publisher{
		Name:          "Active Publisher",
		Address:       sdk.AccAddress("publisher").String(),
		Active:        true,
		ArticlesCount: 5,
		CreatedAt:     1234567890,
		Respect:       "100",
	}
	suite.k.SetPublisher(suite.ctx, publisher)

	// Set up accepted domain
	domain := types.AcceptedDomain{
		Domain: "example.com",
		Active: true,
	}
	suite.k.SetAcceptedDomain(suite.ctx, domain)

	msg := &types.MsgAddArticle{
		Publisher: publisher.Address,
		Title:     "Test Article",
		Url:       "https://example.com/article",
		Picture:   "https://example.com/picture.jpg",
	}

	res, err := suite.msgServer.AddArticle(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify article was created as unpaid (active publisher)
	articles := suite.k.GetAllArticles(suite.ctx)
	suite.Require().Len(articles, 1)
	suite.Require().False(articles[0].Paid)
	suite.Require().Equal(msg.Title, articles[0].Title)
	suite.Require().Equal(msg.Url, articles[0].Url)
	suite.Require().Equal(msg.Picture, articles[0].Picture)
	suite.Require().Equal(msg.Publisher, articles[0].Publisher)

	// Verify publisher article count was incremented
	updatedPublisher, found := suite.k.GetPublisher(suite.ctx, publisher.Address)
	suite.Require().True(found)
	suite.Require().Equal(uint32(6), updatedPublisher.ArticlesCount)

	// Verify the emitted event contains the correct (non-zero) article ID
	events := suite.ctx.EventManager().Events()
	hasArticleAddedEvent := false
	for _, event := range events {
		if strings.Contains(event.Type, "ArticleAddedEvent") {
			hasArticleAddedEvent = true
			for _, attr := range event.Attributes {
				if string(attr.Key) == "article_id" {
					suite.Require().Contains(string(attr.Value), fmt.Sprintf("%d", articles[0].Id))
					suite.Require().NotEqual("\"0\"", string(attr.Value))
				}
			}
		}
	}
	suite.Require().True(hasArticleAddedEvent, "ArticleAddedEvent should be emitted")
}

func (suite *IntegrationTestSuite) TestAddArticle_ValidRequest_PaidArticle() {
	// Set up accepted domain
	domain := types.AcceptedDomain{
		Domain: "example.com",
		Active: true,
	}
	suite.k.SetAcceptedDomain(suite.ctx, domain)

	// Set up params for paid articles
	params := types.Params{
		AnonArticleLimit: 10,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"),
			Denom: "ubze",
		},
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	publisherAddr := sdk.AccAddress("nonexistent").String()
	publisherAccAddr, err := sdk.AccAddressFromBech32(publisherAddr)
	suite.Require().NoError(err)

	expectedCost := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1000))

	// Mock distribution keeper expectation
	suite.distr.EXPECT().FundCommunityPool(suite.ctx, expectedCost, publisherAccAddr).Return(nil).Times(1)

	msg := &types.MsgAddArticle{
		Publisher: publisherAddr,
		Title:     "Paid Article",
		Url:       "https://example.com/paid-article",
		Picture:   "",
	}

	res, err := suite.msgServer.AddArticle(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// Verify article was created as paid
	articles := suite.k.GetAllArticles(suite.ctx)
	suite.Require().Len(articles, 1)
	suite.Require().True(articles[0].Paid)

	// Verify the emitted event contains the correct (non-zero) article ID
	events := suite.ctx.EventManager().Events()
	hasArticleAddedEvent := false
	for _, event := range events {
		if strings.Contains(event.Type, "ArticleAddedEvent") {
			hasArticleAddedEvent = true
			for _, attr := range event.Attributes {
				if string(attr.Key) == "article_id" {
					suite.Require().Contains(string(attr.Value), fmt.Sprintf("%d", articles[0].Id))
					suite.Require().NotEqual("\"0\"", string(attr.Value))
				}
			}
		}
	}
	suite.Require().True(hasArticleAddedEvent, "ArticleAddedEvent should be emitted")
}

func (suite *IntegrationTestSuite) TestAddArticle_InvalidDomain() {
	msg := &types.MsgAddArticle{
		Publisher: sdk.AccAddress("publisher").String(),
		Title:     "Test Article",
		Url:       "https://invalid-domain.com/article",
		Picture:   "",
	}

	res, err := suite.msgServer.AddArticle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "is not an accepted domain")
}

func (suite *IntegrationTestSuite) TestAddArticle_InactiveDomain() {
	// Set up inactive domain
	domain := types.AcceptedDomain{
		Domain: "inactive.com",
		Active: false,
	}
	suite.k.SetAcceptedDomain(suite.ctx, domain)

	msg := &types.MsgAddArticle{
		Publisher: sdk.AccAddress("publisher").String(),
		Title:     "Test Article",
		Url:       "https://inactive.com/article",
		Picture:   "",
	}

	res, err := suite.msgServer.AddArticle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "is NOT active")
}

func (suite *IntegrationTestSuite) TestAddArticle_PaidArticleLimitReached() {
	// Set up accepted domain
	domain := types.AcceptedDomain{
		Domain: "example.com",
		Active: true,
	}
	suite.k.SetAcceptedDomain(suite.ctx, domain)

	// Set up params with limit of 0
	params := types.Params{
		AnonArticleLimit: 0,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"),
			Denom: "ubze",
		},
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	msg := &types.MsgAddArticle{
		Publisher: sdk.AccAddress("nonexistent").String(),
		Title:     "Test Article",
		Url:       "https://example.com/article",
		Picture:   "",
	}

	res, err := suite.msgServer.AddArticle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "Paid article limit reached")
}

func (suite *IntegrationTestSuite) TestAddArticle_FundCommunityPoolError() {
	// Set up accepted domain
	domain := types.AcceptedDomain{
		Domain: "example.com",
		Active: true,
	}
	suite.k.SetAcceptedDomain(suite.ctx, domain)

	// Set up params
	params := types.Params{
		AnonArticleLimit: 10,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"),
			Denom: "ubze",
		},
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	publisherAddr := sdk.AccAddress("nonexistent").String()
	publisherAccAddr, err := sdk.AccAddressFromBech32(publisherAddr)
	suite.Require().NoError(err)

	expectedCost := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1000))
	distrError := errors.New("insufficient funds")

	// Mock distribution keeper to return error
	suite.distr.EXPECT().FundCommunityPool(suite.ctx, expectedCost, publisherAccAddr).Return(distrError).Times(1)

	msg := &types.MsgAddArticle{
		Publisher: publisherAddr,
		Title:     "Test Article",
		Url:       "https://example.com/article",
		Picture:   "",
	}

	res, err := suite.msgServer.AddArticle(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(distrError, err)
}

func (suite *IntegrationTestSuite) TestPayPublisherRespect_ValidRequest() {
	// Set up publisher
	publisher := types.Publisher{
		Name:          "Test Publisher",
		Address:       sdk.AccAddress("publisher").String(),
		Active:        true,
		ArticlesCount: 5,
		CreatedAt:     1234567890,
		Respect:       "100",
	}
	suite.k.SetPublisher(suite.ctx, publisher)

	// Set up params
	params := types.Params{
		AnonArticleLimit: 10,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"), // 10% tax
			Denom: "ubze",
		},
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	creator := sdk.AccAddress("creator").String()
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)
	publisherAddr, err := sdk.AccAddressFromBech32(publisher.Address)
	suite.Require().NoError(err)

	// Expected calculations: 1000 total, 100 tax (10%), 900 to publisher
	publisherReward := sdk.NewCoins(sdk.NewInt64Coin("ubze", 900))
	taxAmount := sdk.NewCoins(sdk.NewInt64Coin("ubze", 100))

	// Mock expectations
	suite.bank.EXPECT().SendCoins(suite.ctx, creatorAddr, publisherAddr, publisherReward).Return(nil).Times(1)
	suite.distr.EXPECT().FundCommunityPool(suite.ctx, taxAmount, creatorAddr).Return(nil).Times(1)

	msg := &types.MsgPayPublisherRespect{
		Creator: creator,
		Address: publisher.Address,
		Amount:  "1000ubze",
	}

	res, err := suite.msgServer.PayPublisherRespect(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(uint64(1000), res.RespectPaid)
	suite.Require().Equal(uint64(900), res.PublisherReward)
	suite.Require().Equal(uint64(100), res.CommunityPoolFunds)

	// Verify publisher respect was updated
	updatedPublisher, found := suite.k.GetPublisher(suite.ctx, publisher.Address)
	suite.Require().True(found)
	suite.Require().Equal("1100", updatedPublisher.Respect) // "100" + 1000
}

func (suite *IntegrationTestSuite) TestPayPublisherRespect_InvalidAmount() {
	msg := &types.MsgPayPublisherRespect{
		Creator: sdk.AccAddress("creator").String(),
		Address: sdk.AccAddress("publisher").String(),
		Amount:  "invalid-amount",
	}

	res, err := suite.msgServer.PayPublisherRespect(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid amount")
}

func (suite *IntegrationTestSuite) TestPayPublisherRespect_WrongDenom() {
	// Set up params expecting "ubze"
	params := types.Params{
		AnonArticleLimit: 10,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"),
			Denom: "ubze",
		},
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	msg := &types.MsgPayPublisherRespect{
		Creator: sdk.AccAddress("creator").String(),
		Address: sdk.AccAddress("publisher").String(),
		Amount:  "1000uatom", // Wrong denom
	}

	res, err := suite.msgServer.PayPublisherRespect(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid coin denom")
}

func (suite *IntegrationTestSuite) TestPayPublisherRespect_PublisherNotFound() {
	// Set up params
	params := types.Params{
		AnonArticleLimit: 10,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"),
			Denom: "ubze",
		},
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	msg := &types.MsgPayPublisherRespect{
		Creator: sdk.AccAddress("creator").String(),
		Address: sdk.AccAddress("nonexistent").String(),
		Amount:  "1000ubze",
	}

	res, err := suite.msgServer.PayPublisherRespect(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "could not be found")
}

func (suite *IntegrationTestSuite) TestPayPublisherRespect_ZeroAmount() {
	// Set up params
	params := types.Params{
		AnonArticleLimit: 10,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"),
			Denom: "ubze",
		},
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	msg := &types.MsgPayPublisherRespect{
		Creator: sdk.AccAddress("creator").String(),
		Address: sdk.AccAddress("publisher").String(),
		Amount:  "0ubze",
	}

	res, err := suite.msgServer.PayPublisherRespect(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "amount should be positive")
}

func (suite *IntegrationTestSuite) TestPayPublisherRespect_BankError() {
	// Set up publisher
	publisher := types.Publisher{
		Name:          "Test Publisher",
		Address:       sdk.AccAddress("publisher").String(),
		Active:        true,
		ArticlesCount: 5,
		CreatedAt:     1234567890,
		Respect:       "100",
	}
	suite.k.SetPublisher(suite.ctx, publisher)

	// Set up params
	params := types.Params{
		AnonArticleLimit: 10,
		AnonArticleCost:  sdk.NewInt64Coin("ubze", 1000),
		PublisherRespectParams: types.PublisherRespectParams{
			Tax:   math.LegacyMustNewDecFromStr("0.1"),
			Denom: "ubze",
		},
	}
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	creator := sdk.AccAddress("creator").String()
	creatorAddr, err := sdk.AccAddressFromBech32(creator)
	suite.Require().NoError(err)
	publisherAddr, err := sdk.AccAddressFromBech32(publisher.Address)
	suite.Require().NoError(err)

	publisherReward := sdk.NewCoins(sdk.NewInt64Coin("ubze", 900))
	bankError := errors.New("insufficient funds")

	// Mock bank to return error
	suite.bank.EXPECT().SendCoins(suite.ctx, creatorAddr, publisherAddr, publisherReward).Return(bankError).Times(1)

	msg := &types.MsgPayPublisherRespect{
		Creator: creator,
		Address: publisher.Address,
		Amount:  "1000ubze",
	}

	res, err := suite.msgServer.PayPublisherRespect(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(bankError, err)
}
