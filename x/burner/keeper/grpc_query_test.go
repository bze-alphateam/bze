package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestParams_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.Params(goCtx, nil)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestParams_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.Params(goCtx, &types.QueryParamsRequest{})
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestAllBurnedCoins_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.AllBurnedCoins(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestAllBurnedCoins_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	burned1 := types.BurnedCoins{
		Burned: "123ubze",
		Height: "1",
	}
	suite.k.SetBurnedCoins(suite.ctx, burned1)

	resp, err := suite.k.AllBurnedCoins(goCtx, &types.QueryAllBurnedCoinsRequest{})
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Len(resp.BurnedCoins, 1)
	suite.Require().Equal(resp.BurnedCoins[0], burned1)

	burned2 := types.BurnedCoins{
		Burned: "100ubze",
		Height: "12",
	}
	suite.k.SetBurnedCoins(suite.ctx, burned2)

	resp, err = suite.k.AllBurnedCoins(goCtx, &types.QueryAllBurnedCoinsRequest{})
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Len(resp.BurnedCoins, 2)
	suite.Require().Equal(resp.BurnedCoins[1], burned2)

	//append some coins to the burn
	suite.ctx = suite.ctx.WithBlockHeight(12)
	err = suite.k.SaveBurnedCoins(suite.ctx, sdk.NewCoins(sdk.NewInt64Coin("ubze", 111)))
	suite.Require().NoError(err)
	resp, err = suite.k.AllBurnedCoins(goCtx, &types.QueryAllBurnedCoinsRequest{})
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Len(resp.BurnedCoins, 2)
	suite.Require().Equal(resp.BurnedCoins[1].Burned, "211ubze")
	suite.Require().Equal(resp.BurnedCoins[1].Height, "12")
}

func (suite *IntegrationTestSuite) TestRaffle_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.Raffles(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestRaffle_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	raffle1 := types.Raffle{
		Pot:         "1",
		Duration:    1,
		Chances:     1,
		Ratio:       "0.1",
		EndAt:       1,
		Winners:     1,
		TicketPrice: "1",
		Denom:       "ubze",
		TotalWon:    "0",
	}
	suite.k.SetRaffle(suite.ctx, raffle1)

	resp, err := suite.k.Raffles(goCtx, &types.QueryRafflesRequest{})
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Len(resp.List, 1)

	raffle2 := types.Raffle{
		Pot:         "1",
		Duration:    1,
		Chances:     12,
		Ratio:       "0.1",
		EndAt:       1,
		Winners:     1,
		TicketPrice: "1",
		Denom:       "tbz",
		TotalWon:    "0",
	}
	suite.k.SetRaffle(suite.ctx, raffle2)

	resp, err = suite.k.Raffles(goCtx, &types.QueryRafflesRequest{})
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Len(resp.List, 2)
	suite.Require().Equal(resp.List[1], raffle1)
	suite.Require().Equal(resp.List[0], raffle2)
}

func (suite *IntegrationTestSuite) TestRaffleWinners_InvalidRequest() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.RaffleWinners(goCtx, nil)
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestRaffleWinners_InvalidDenom() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.k.RaffleWinners(goCtx, &types.QueryRaffleWinnersRequest{})
	suite.Require().Error(err)
}

func (suite *IntegrationTestSuite) TestRaffleWinners_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	denom := "ubze"
	raffle := types.Raffle{
		Pot:         "",
		Duration:    0,
		Chances:     0,
		Ratio:       "",
		EndAt:       0,
		Winners:     0,
		TicketPrice: "",
		Denom:       denom,
		TotalWon:    "",
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	resp, err := suite.k.RaffleWinners(goCtx, &types.QueryRaffleWinnersRequest{Denom: denom})
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Len(resp.List, 0)

	w1 := types.RaffleWinner{
		Index:  "1",
		Denom:  denom,
		Amount: "1",
		Winner: "address1",
	}
	suite.k.SetRaffleWinner(suite.ctx, w1)

	resp, err = suite.k.RaffleWinners(goCtx, &types.QueryRaffleWinnersRequest{Denom: denom})
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Len(resp.List, 1)
	suite.Require().Equal(resp.List[0], w1)

	w2 := types.RaffleWinner{
		Index:  "2",
		Denom:  denom,
		Amount: "1",
		Winner: "address2",
	}
	suite.k.SetRaffleWinner(suite.ctx, w2)

	resp, err = suite.k.RaffleWinners(goCtx, &types.QueryRaffleWinnersRequest{Denom: denom})
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Require().Len(resp.List, 2)
	suite.Require().Equal(resp.List[0], w1)
	suite.Require().Equal(resp.List[1], w2)
}
