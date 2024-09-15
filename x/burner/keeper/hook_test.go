package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestGetBurnerRaffleCleanupHook() {
	hook := suite.k.GetBurnerRaffleCleanupHook()
	suite.Require().NotNil(hook)

	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "day", 1321))
	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "week", 132))
	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "hour", 11))
}

func (suite *IntegrationTestSuite) TestBurnerRaffleCleanupHook_MultipleRaffles_DifferentEndAt() {
	hook := suite.k.GetBurnerRaffleCleanupHook()
	suite.Require().NotNil(hook)

	//call it to create the module account
	moduleAcc := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.RaffleModuleName)

	//add to store some random data to delete
	for i := 1; i <= 5; i++ {
		denom := fmt.Sprintf("burner%d", i)
		raffleDeleteHook := types.RaffleDeleteHook{
			Denom: denom,
			EndAt: uint64(i),
		}
		suite.k.SetRaffleDeleteHook(suite.ctx, raffleDeleteHook)

		raffle := types.Raffle{
			Pot:         "5000",
			Duration:    1,
			Chances:     1,
			Ratio:       "0.1",
			EndAt:       uint64(i),
			Winners:     1,
			TicketPrice: "2",
			Denom:       denom,
		}
		suite.k.SetRaffle(suite.ctx, raffle)

		balances := sdk.NewCoins(sdk.NewInt64Coin(denom, 5000))
		suite.Require().NoError(
			simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.RaffleModuleName, balances),
		)

		w1 := types.RaffleWinner{
			Index:  "1",
			Denom:  denom,
			Amount: "231",
			Winner: "addr_1",
		}
		suite.k.SetRaffleWinner(suite.ctx, w1)

		w2 := types.RaffleWinner{
			Index:  "2",
			Denom:  denom,
			Amount: "231",
			Winner: "addr_1",
		}
		suite.k.SetRaffleWinner(suite.ctx, w2)
	}

	//minimal check that we have something in storage
	list := suite.k.GetAllRaffle(suite.ctx)
	suite.Require().Len(list, 5)

	for i := 1; i <= 5; i++ {
		suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "hour", int64(i)))
		denom := fmt.Sprintf("burner%d", i)

		//check raffle was deleted
		_, ok := suite.k.GetRaffle(suite.ctx, denom)
		suite.Require().False(ok)

		//check winners were deleted
		winners := suite.k.GetRaffleWinners(suite.ctx, denom)
		suite.Require().Len(winners, 0)

		delHooksStored := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, uint64(i))
		suite.Require().Len(delHooksStored, 0)

		bal := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAcc.GetAddress(), denom)
		suite.Require().True(bal.IsZero())
	}
}

func (suite *IntegrationTestSuite) TestBurnerRaffleCleanupHook_MultipleRaffles_SameEndAt() {
	hook := suite.k.GetBurnerRaffleCleanupHook()
	suite.Require().NotNil(hook)

	//call it to create the module account
	moduleAcc := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.RaffleModuleName)

	//add to store some random data to delete
	for i := 1; i <= 5; i++ {
		denom := fmt.Sprintf("burner%d", i)
		if i%2 == 0 {
			denom = fmt.Sprintf("factory/%s", denom)
		}
		raffleDeleteHook := types.RaffleDeleteHook{
			Denom: denom,
			EndAt: 1,
		}
		suite.k.SetRaffleDeleteHook(suite.ctx, raffleDeleteHook)

		raffle := types.Raffle{
			Pot:         "5000",
			Duration:    1,
			Chances:     1,
			Ratio:       "0.1",
			EndAt:       1,
			Winners:     1,
			TicketPrice: "2",
			Denom:       denom,
		}
		suite.k.SetRaffle(suite.ctx, raffle)

		balances := sdk.NewCoins(sdk.NewInt64Coin(denom, 5000))
		suite.Require().NoError(
			simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.RaffleModuleName, balances),
		)

		w1 := types.RaffleWinner{
			Index:  "1",
			Denom:  denom,
			Amount: "231",
			Winner: "addr_1",
		}
		suite.k.SetRaffleWinner(suite.ctx, w1)

		w2 := types.RaffleWinner{
			Index:  "2",
			Denom:  denom,
			Amount: "231",
			Winner: "addr_1",
		}
		suite.k.SetRaffleWinner(suite.ctx, w2)
	}

	//minimal check that we have something in storage
	list := suite.k.GetAllRaffle(suite.ctx)
	suite.Require().Len(list, 5)

	suite.Require().NoError(hook.AfterEpochEnd(suite.ctx, "hour", int64(1)))
	for i := 1; i <= 5; i++ {
		isFactoryDenom := false
		denom := fmt.Sprintf("burner%d", i)
		if i%2 == 0 {
			denom = fmt.Sprintf("factory/%s", denom)
			isFactoryDenom = true
		}

		//check raffle was deleted
		_, ok := suite.k.GetRaffle(suite.ctx, denom)
		suite.Require().False(ok)

		//check winners were deleted
		winners := suite.k.GetRaffleWinners(suite.ctx, denom)
		suite.Require().Len(winners, 0)

		delHooksStored := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, uint64(i))
		suite.Require().Len(delHooksStored, 0)

		bal := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAcc.GetAddress(), denom)
		suite.Require().True(bal.IsZero())

		allBurns := suite.k.GetAllBurnedCoins(suite.ctx)
		if !isFactoryDenom {
			suite.Require().NotEmpty(allBurns)
		}
		for _, b := range allBurns {
			if isFactoryDenom {
				suite.Require().NotContains(b.Burned, denom)
			} else {
				suite.Require().Contains(b.Burned, denom)
			}
		}
	}
}
