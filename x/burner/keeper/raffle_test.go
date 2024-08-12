package keeper_test

import "github.com/bze-alphateam/bze/x/burner/types"

func (suite *IntegrationTestSuite) Test_Raffle() {
	raffle := types.Raffle{
		Pot:         "1000",
		Duration:    1,
		Chances:     11,
		Ratio:       "0.1",
		EndAt:       5,
		Winners:     1,
		TicketPrice: "1",
		Denom:       "asd1",
	}

	suite.k.SetRaffle(suite.ctx, raffle)

	r1, found := suite.k.GetRaffle(suite.ctx, "asd1")
	suite.Require().True(found)
	suite.Require().Equal(raffle, r1)

	raffle2 := types.Raffle{
		Pot:         "1000",
		Duration:    1,
		Chances:     11,
		Ratio:       "0.1",
		EndAt:       5,
		Winners:     1,
		TicketPrice: "1",
		Denom:       "asd2",
	}

	suite.k.SetRaffle(suite.ctx, raffle2)
	r2, found := suite.k.GetRaffle(suite.ctx, "asd2")
	suite.Require().True(found)
	suite.Require().Equal(raffle2, r2)

	list := suite.k.GetAllRaffle(suite.ctx)
	suite.Require().Len(list, 2)
	suite.Require().Contains(list, raffle)
	suite.Require().Contains(list, raffle2)

	suite.k.RemoveRaffle(suite.ctx, raffle.Denom)
	list = suite.k.GetAllRaffle(suite.ctx)
	suite.Require().Len(list, 1)
	suite.Require().Contains(list, raffle2)

	suite.k.RemoveRaffle(suite.ctx, raffle2.Denom)
	list = suite.k.GetAllRaffle(suite.ctx)
	suite.Require().Len(list, 0)
}

func (suite *IntegrationTestSuite) Test_RaffleDeleteHook() {
	rdh := types.RaffleDeleteHook{
		Denom: "asd1",
		EndAt: 12,
	}
	suite.k.SetRaffleDeleteHook(suite.ctx, rdh)

	list := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, 12)
	suite.Require().Len(list, 1)
	suite.Require().Equal(rdh, list[0])

	rdh2 := types.RaffleDeleteHook{
		Denom: "asd2",
		EndAt: 12,
	}
	suite.k.SetRaffleDeleteHook(suite.ctx, rdh2)
	list = suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, 12)
	suite.Require().Len(list, 2)
	suite.Require().Contains(list, rdh2)
	suite.Require().Contains(list, rdh)

	rdh3 := types.RaffleDeleteHook{
		Denom: "asd3",
		EndAt: 123,
	}
	suite.k.SetRaffleDeleteHook(suite.ctx, rdh3)
	list = suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, 123)
	suite.Require().Len(list, 1)
	suite.Require().Contains(list, rdh3)

	suite.k.RemoveRaffleDeleteHook(suite.ctx, rdh)
	list = suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, 12)
	suite.Require().Len(list, 1)
	suite.Require().Contains(list, rdh2)

	suite.k.RemoveRaffleDeleteHook(suite.ctx, rdh2)
	list = suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, 12)
	suite.Require().Len(list, 0)

	suite.k.RemoveRaffleDeleteHook(suite.ctx, rdh3)
	list = suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, 123)
	suite.Require().Len(list, 0)
}

func (suite *IntegrationTestSuite) Test_RaffleWinner() {
	rw := types.RaffleWinner{
		Index:  "1",
		Denom:  "asd1",
		Amount: "1200",
		Winner: "addr1",
	}

	suite.k.SetRaffleWinner(suite.ctx, rw)
	list := suite.k.GetRaffleWinners(suite.ctx, "asd1")
	suite.Require().Len(list, 1)
	suite.Require().Equal(rw, list[0])

	rw2 := types.RaffleWinner{
		Index:  "2",
		Denom:  "asd1",
		Amount: "1200",
		Winner: "addr1",
	}

	suite.k.SetRaffleWinner(suite.ctx, rw2)
	list = suite.k.GetRaffleWinners(suite.ctx, "asd1")
	suite.Require().Len(list, 2)
	suite.Require().Contains(list, rw2)
	suite.Require().Contains(list, rw)

	rw3 := types.RaffleWinner{
		Index:  "2",
		Denom:  "asd1",
		Amount: "12100",
		Winner: "addr4",
	}
	//override previous set winner
	suite.k.SetRaffleWinner(suite.ctx, rw3)
	list = suite.k.GetRaffleWinners(suite.ctx, "asd1")
	suite.Require().Len(list, 2)
	suite.Require().Contains(list, rw3)

	//random index
	rw2.Index = "321"
	suite.k.RemoveRaffleWinner(suite.ctx, rw2)
	list = suite.k.GetRaffleWinners(suite.ctx, "asd1")
	suite.Require().Len(list, 2)
	suite.Require().Contains(list, rw)
	suite.Require().Contains(list, rw3)

	suite.k.RemoveRaffleWinner(suite.ctx, rw)
	list = suite.k.GetRaffleWinners(suite.ctx, "asd1")
	suite.Require().Len(list, 1)
	suite.Require().Contains(list, rw3)

	suite.k.RemoveRaffleWinner(suite.ctx, rw3)
	list = suite.k.GetRaffleWinners(suite.ctx, "asd1")
	suite.Require().Len(list, 0)
}
