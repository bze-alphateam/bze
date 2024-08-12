package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
)

func (suite *IntegrationTestSuite) TestSetBurnedCoins() {
	toSave := types.BurnedCoins{
		Burned: "1234",
		Height: "3",
	}

	suite.k.SetBurnedCoins(suite.ctx, toSave)

	all := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Equal(len(all), 1)
	suite.Require().Equal(all[0].Burned, toSave.Burned)
	suite.Require().Equal(all[0].Height, toSave.Height)
}

func (suite *IntegrationTestSuite) TestGetAllBurnedCoins() {
	toSave := types.BurnedCoins{
		Burned: "1234",
		Height: "3",
	}

	suite.k.SetBurnedCoins(suite.ctx, toSave)

	all := suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Equal(len(all), 1)
	suite.Require().Equal(all[0].Burned, toSave.Burned)
	suite.Require().Equal(all[0].Height, toSave.Height)

	toSave2 := types.BurnedCoins{
		Burned: "29000000000000ubze",
		Height: "33321",
	}

	suite.k.SetBurnedCoins(suite.ctx, toSave2)
	all = suite.k.GetAllBurnedCoins(suite.ctx)
	suite.Require().Equal(len(all), 2)
	suite.Require().Equal(all[0].Burned, toSave.Burned)
	suite.Require().Equal(all[0].Height, toSave.Height)
	suite.Require().Equal(all[1].Burned, toSave2.Burned)
	suite.Require().Equal(all[1].Height, toSave2.Height)
}
