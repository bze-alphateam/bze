package keeper_test

import (
	"github.com/bze-alphateam/bze/x/cointrunk/types"
)

func (suite *IntegrationTestSuite) TestSetArticle_WithId() {
	// Test data with existing ID
	article := types.Article{
		Id:        123,
		Title:     "Test Article",
		Url:       "https://example.com/article",
		Picture:   "https://example.com/picture.jpg",
		Publisher: "test-publisher",
		Paid:      false,
		CreatedAt: 1234567890,
	}

	// Test SetArticle
	suite.k.SetArticle(suite.ctx, &article)

	// Verify article was stored
	allArticles := suite.k.GetAllArticles(suite.ctx)
	suite.Require().Len(allArticles, 1)

	retrievedArticle := allArticles[0]
	suite.Require().Equal(article.Id, retrievedArticle.Id)
	suite.Require().Equal(article.Title, retrievedArticle.Title)
	suite.Require().Equal(article.Url, retrievedArticle.Url)
	suite.Require().Equal(article.Picture, retrievedArticle.Picture)
	suite.Require().Equal(article.Publisher, retrievedArticle.Publisher)
	suite.Require().Equal(article.Paid, retrievedArticle.Paid)
	suite.Require().Equal(article.CreatedAt, retrievedArticle.CreatedAt)
}

func (suite *IntegrationTestSuite) TestSetArticle_AutoIncrementId() {
	// Test data without ID (should auto-increment)
	article := types.Article{
		Id:        0, // Should be auto-incremented
		Title:     "Auto ID Article",
		Url:       "https://example.com/auto",
		Picture:   "https://example.com/auto.jpg",
		Publisher: "auto-publisher",
		Paid:      false,
		CreatedAt: 1234567890,
	}

	// Test SetArticle
	suite.k.SetArticle(suite.ctx, &article)

	// Verify the caller's copy was updated with the auto-incremented ID
	suite.Require().Equal(uint64(1), article.Id)

	// Verify article was stored with auto-incremented ID
	allArticles := suite.k.GetAllArticles(suite.ctx)
	suite.Require().Len(allArticles, 1)

	retrievedArticle := allArticles[0]
	suite.Require().Equal(uint64(1), retrievedArticle.Id) // Should be auto-incremented to 1
	suite.Require().Equal(article.Title, retrievedArticle.Title)
	suite.Require().Equal(article.Url, retrievedArticle.Url)
	suite.Require().Equal(article.Picture, retrievedArticle.Picture)
	suite.Require().Equal(article.Publisher, retrievedArticle.Publisher)
	suite.Require().Equal(article.Paid, retrievedArticle.Paid)
	suite.Require().Equal(article.CreatedAt, retrievedArticle.CreatedAt)
}

func (suite *IntegrationTestSuite) TestSetArticle_MultipleAutoIncrement() {
	// Test multiple articles with auto-increment
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
		{
			Id:        0,
			Title:     "Article 3",
			Url:       "https://example.com/3",
			Publisher: "publisher3",
			Paid:      false,
			CreatedAt: 1234567892,
		},
	}

	// Set all articles
	for _, article := range articles {
		suite.k.SetArticle(suite.ctx, &article)
	}

	// Verify all articles were stored with incrementing IDs
	allArticles := suite.k.GetAllArticles(suite.ctx)
	suite.Require().Len(allArticles, 3)

	// Verify IDs are properly incremented
	idMap := make(map[uint64]types.Article)
	for _, article := range allArticles {
		idMap[article.Id] = article
	}

	// Should have IDs 1, 2, 3
	for i := uint64(1); i <= 3; i++ {
		article, exists := idMap[i]
		suite.Require().True(exists)
		suite.Require().Equal(i, article.Id)
	}
}

func (suite *IntegrationTestSuite) TestSetArticle_PaidArticleCounter() {
	// Test that paid articles increment the monthly counter
	initialCounter := suite.k.GetMonthlyPaidArticleCounter(suite.ctx)
	suite.Require().Equal(uint64(0), initialCounter)

	// Create paid article
	paidArticle := types.Article{
		Id:        0,
		Title:     "Paid Article",
		Url:       "https://example.com/paid",
		Publisher: "paid-publisher",
		Paid:      true,
		CreatedAt: 1234567890,
	}

	suite.k.SetArticle(suite.ctx, &paidArticle)

	// Verify counter incremented
	counterAfterPaid := suite.k.GetMonthlyPaidArticleCounter(suite.ctx)
	suite.Require().Equal(uint64(1), counterAfterPaid)

	// Create unpaid article
	unpaidArticle := types.Article{
		Id:        0,
		Title:     "Unpaid Article",
		Url:       "https://example.com/unpaid",
		Publisher: "unpaid-publisher",
		Paid:      false,
		CreatedAt: 1234567891,
	}

	suite.k.SetArticle(suite.ctx, &unpaidArticle)

	// Verify counter didn't increment for unpaid article
	counterAfterUnpaid := suite.k.GetMonthlyPaidArticleCounter(suite.ctx)
	suite.Require().Equal(uint64(1), counterAfterUnpaid)

	// Add another paid article
	anotherPaidArticle := types.Article{
		Id:        0,
		Title:     "Another Paid Article",
		Url:       "https://example.com/paid2",
		Publisher: "paid-publisher2",
		Paid:      true,
		CreatedAt: 1234567892,
	}

	suite.k.SetArticle(suite.ctx, &anotherPaidArticle)

	// Verify counter incremented again
	finalCounter := suite.k.GetMonthlyPaidArticleCounter(suite.ctx)
	suite.Require().Equal(uint64(2), finalCounter)
}

func (suite *IntegrationTestSuite) TestGetAllArticles_EmptyStore() {
	// Test GetAllArticles with no articles
	allArticles := suite.k.GetAllArticles(suite.ctx)
	suite.Require().Len(allArticles, 0)
}

func (suite *IntegrationTestSuite) TestGetArticleCounter() {
	// Test initial counter
	initialCounter := suite.k.GetArticleCounter(suite.ctx)
	suite.Require().Equal(uint64(0), initialCounter)

	// Add an article (should increment counter)
	article := types.Article{
		Id:        0,
		Title:     "Counter Test Article",
		Publisher: "test-publisher",
		Paid:      false,
		CreatedAt: 1234567890,
	}

	suite.k.SetArticle(suite.ctx, &article)

	// Verify counter incremented
	counter := suite.k.GetArticleCounter(suite.ctx)
	suite.Require().Equal(uint64(1), counter)
}

func (suite *IntegrationTestSuite) TestSetArticleCounter() {
	// Test setting custom counter value
	suite.k.SetArticleCounter(suite.ctx, 100)

	// Verify counter was set
	counter := suite.k.GetArticleCounter(suite.ctx)
	suite.Require().Equal(uint64(100), counter)

	// Add article with auto-increment (should use next counter value)
	article := types.Article{
		Id:        0,
		Title:     "Article After Custom Counter",
		Publisher: "test-publisher",
		Paid:      false,
		CreatedAt: 1234567890,
	}

	suite.k.SetArticle(suite.ctx, &article)

	// Verify article got ID 101 (next after 100)
	allArticles := suite.k.GetAllArticles(suite.ctx)
	suite.Require().Len(allArticles, 1)
	suite.Require().Equal(uint64(101), allArticles[0].Id)

	// Verify counter is now 101
	finalCounter := suite.k.GetArticleCounter(suite.ctx)
	suite.Require().Equal(uint64(101), finalCounter)
}

func (suite *IntegrationTestSuite) TestSetArticle_EmptyFields() {
	// Test setting article with empty optional fields
	article := types.Article{
		Id:        0,
		Title:     "",
		Url:       "",
		Picture:   "",
		Publisher: "",
		Paid:      false,
		CreatedAt: 0,
	}

	suite.k.SetArticle(suite.ctx, &article)

	// Verify it was stored correctly
	allArticles := suite.k.GetAllArticles(suite.ctx)
	suite.Require().Len(allArticles, 1)

	retrievedArticle := allArticles[0]
	suite.Require().Equal(uint64(1), retrievedArticle.Id) // Auto-incremented
	suite.Require().Equal("", retrievedArticle.Title)
	suite.Require().Equal("", retrievedArticle.Url)
	suite.Require().Equal("", retrievedArticle.Picture)
	suite.Require().Equal("", retrievedArticle.Publisher)
	suite.Require().False(retrievedArticle.Paid)
	suite.Require().Equal(int64(0), retrievedArticle.CreatedAt)
}
