package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *IntegrationTestSuite) TestQuery_Raffles_ValidRequest() {
	// Create test raffles
	raffles := []types.Raffle{
		{
			Pot:      "1000utoken1",
			Duration: 3600,
			Denom:    "utoken1",
			EndAt:    12345,
		},
		{
			Pot:      "2000utoken2",
			Duration: 7200,
			Denom:    "utoken2",
			EndAt:    23456,
		},
	}

	for _, raffle := range raffles {
		suite.k.SetRaffle(suite.ctx, raffle)
	}

	req := &types.QueryRafflesRequest{}
	res, err := suite.k.Raffles(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.List, 2)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestQuery_Raffles_NilRequest() {
	res, err := suite.k.Raffles(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQuery_Raffles_WithPagination() {
	// Create multiple raffles
	for i := 0; i < 5; i++ {
		raffle := types.Raffle{
			Pot:      "1000utoken",
			Duration: 3600,
			Denom:    "utoken" + string(rune('1'+i)),
			EndAt:    uint64(12345 + i),
		}
		suite.k.SetRaffle(suite.ctx, raffle)
	}

	req := &types.QueryRafflesRequest{
		Pagination: &query.PageRequest{
			Limit: 3,
		},
	}
	res, err := suite.k.Raffles(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.List, 3)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestQuery_RaffleWinners_ValidRequest() {
	// Create test winners
	denom := "utoken"
	winners := []types.RaffleWinner{
		{Index: "1", Denom: denom, Amount: "100utoken", Winner: "addr1"},
		{Index: "2", Denom: denom, Amount: "200utoken", Winner: "addr2"},
	}

	for _, winner := range winners {
		suite.k.SetRaffleWinner(suite.ctx, winner)
	}

	req := &types.QueryRaffleWinnersRequest{
		Denom: denom,
	}
	res, err := suite.k.RaffleWinners(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.List, 2)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestQuery_RaffleWinners_NilRequest() {
	res, err := suite.k.RaffleWinners(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQuery_RaffleWinners_EmptyDenom() {
	req := &types.QueryRaffleWinnersRequest{
		Denom: "",
	}
	res, err := suite.k.RaffleWinners(suite.ctx, req)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "denom required")
}

func (suite *IntegrationTestSuite) TestQuery_RaffleWinners_WithPagination() {
	denom := "utoken"
	// Create multiple winners
	for i := 0; i < 5; i++ {
		winner := types.RaffleWinner{
			Index:  string(rune('1' + i)),
			Denom:  denom,
			Amount: "100utoken",
			Winner: "addr" + string(rune('1'+i)),
		}
		suite.k.SetRaffleWinner(suite.ctx, winner)
	}

	req := &types.QueryRaffleWinnersRequest{
		Denom: denom,
		Pagination: &query.PageRequest{
			Limit: 3,
		},
	}
	res, err := suite.k.RaffleWinners(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.List, 3)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestQuery_AllBurnedCoins_ValidRequest() {
	// Create test burned coins
	burnedCoins := []types.BurnedCoins{
		{Burned: "1000utoken", Height: "100"},
		{Burned: "2000utoken", Height: "200"},
	}

	for _, entry := range burnedCoins {
		suite.k.SetBurnedCoins(suite.ctx, entry)
	}

	req := &types.QueryAllBurnedCoinsRequest{}
	res, err := suite.k.AllBurnedCoins(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.BurnedCoins, 2)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestQuery_AllBurnedCoins_NilRequest() {
	res, err := suite.k.AllBurnedCoins(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Equal(codes.InvalidArgument, status.Code(err))
	suite.Require().Contains(err.Error(), "invalid request")
}

func (suite *IntegrationTestSuite) TestQuery_AllBurnedCoins_WithPagination() {
	// Create multiple burned coins entries
	for i := 0; i < 5; i++ {
		entry := types.BurnedCoins{
			Burned: "1000utoken",
			Height: string(rune('1' + i)),
		}
		suite.k.SetBurnedCoins(suite.ctx, entry)
	}

	req := &types.QueryAllBurnedCoinsRequest{
		Pagination: &query.PageRequest{
			Limit: 3,
		},
	}
	res, err := suite.k.AllBurnedCoins(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.BurnedCoins, 3)
	suite.Require().NotNil(res.Pagination)
}

func (suite *IntegrationTestSuite) TestQuery_AllBurnedCoins_EmptyStore() {
	req := &types.QueryAllBurnedCoinsRequest{}
	res, err := suite.k.AllBurnedCoins(suite.ctx, req)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.BurnedCoins, 0)
	suite.Require().NotNil(res.Pagination)
}
