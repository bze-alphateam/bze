package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestUserDustStorage() {
	ud1 := types.UserDust{
		Owner:  "addr1",
		Amount: "0.321312231",
		Denom:  "ubze",
	}

	//test save
	suite.k.SetUserDust(suite.ctx, ud1)
	fromStore, ok := suite.k.GetUserDust(suite.ctx, ud1.Owner, ud1.Denom)
	suite.Require().True(ok)
	suite.Require().Equal(fromStore, ud1)

	ud2 := types.UserDust{
		Owner:  "addr2",
		Amount: "0.098721",
		Denom:  "uteststake",
	}
	//test save 2
	suite.k.SetUserDust(suite.ctx, ud2)
	//check previous entry is still there
	fromStore, ok = suite.k.GetUserDust(suite.ctx, ud1.Owner, ud1.Denom)
	suite.Require().True(ok)
	suite.Require().Equal(fromStore, ud1)
	//check this entry was also saved
	fromStore2, ok := suite.k.GetUserDust(suite.ctx, ud2.Owner, ud2.Denom)
	suite.Require().True(ok)
	suite.Require().Equal(fromStore2, ud2)

	//check both are returned by listing method
	all := suite.k.GetAllUserDust(suite.ctx)
	suite.Require().Len(all, 2)
	suite.Require().Contains(all, ud1)
	suite.Require().Contains(all, ud2)

	//check first entry is returned when queried by address
	byAddress := suite.k.GetUserDustByOwner(suite.ctx, ud1.Owner)
	suite.Require().Len(byAddress, 1)
	suite.Require().Contains(byAddress, ud1)

	//check second entry is returned when queried by address
	byAddress2 := suite.k.GetUserDustByOwner(suite.ctx, ud2.Owner)
	suite.Require().Len(byAddress2, 1)
	suite.Require().Contains(byAddress2, ud2)

	//check delete. delete ud1
	suite.k.RemoveUserDust(suite.ctx, ud1)
	fromStore, ok = suite.k.GetUserDust(suite.ctx, ud1.Owner, ud1.Denom)
	suite.Require().False(ok)

	//check that the list does not contain the deleted entry
	all = suite.k.GetAllUserDust(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().NotContains(all, ud1)
	suite.Require().Contains(all, ud2)

	//check list by address is empty now for the deleted entry owner
	byAddress = suite.k.GetUserDustByOwner(suite.ctx, ud1.Owner)
	suite.Require().Len(byAddress, 0)
}

func (suite *IntegrationTestSuite) TestStoreProcessedUserDust_WithNilDustDec() {
	ud1 := types.UserDust{
		Owner:  "addr1",
		Amount: "0.321312231",
		Denom:  "ubze",
	}

	suite.k.StoreProcessedUserDust(suite.ctx, &ud1, nil)

	fromStorage, ok := suite.k.GetUserDust(suite.ctx, ud1.Owner, ud1.Denom)
	suite.Require().True(ok)
	suite.Require().Equal(fromStorage, ud1)
}

func (suite *IntegrationTestSuite) TestStoreProcessedUserDust_WithZeroDust() {
	ud1 := types.UserDust{
		Owner:  "addr1",
		Amount: "0.321312231",
		Denom:  "ubze",
	}
	suite.k.SetUserDust(suite.ctx, ud1)

	zeroDec := sdk.ZeroDec()
	suite.k.StoreProcessedUserDust(suite.ctx, &ud1, &zeroDec)

	_, ok := suite.k.GetUserDust(suite.ctx, ud1.Owner, ud1.Denom)
	suite.Require().False(ok)
}

func (suite *IntegrationTestSuite) TestCollectUserDust_ZeroDust() {
	coin := sdk.NewCoin("ubze", sdk.NewInt(100))
	dust := sdk.ZeroDec()
	addr := "addr1"
	res, userDust, dust, err := suite.k.CollectUserDust(suite.ctx, addr, coin, dust, false)
	suite.Require().NoError(err)
	suite.Require().Nil(userDust)
	suite.Require().Equal(res, coin)
	suite.Require().Equal(dust, sdk.ZeroDec())
}

func (suite *IntegrationTestSuite) TestCollectUserDust_PayerFirstDust() {
	coin := sdk.NewCoin("ubze", sdk.NewInt(100))
	dust, err := sdk.NewDecFromStr("0.032121123123123")
	suite.Require().Nil(err)
	addr := "addr1"
	res, userDust, dustResulted, err := suite.k.CollectUserDust(suite.ctx, addr, coin, dust, false)
	suite.Require().NoError(err)
	suite.Require().NotNil(userDust)
	suite.Require().Equal(res, coin.AddAmount(sdk.OneInt()))
	suite.Require().NotEqual(dustResulted, dust)
	suite.Require().Equal(userDust.Owner, addr)
	suite.Require().Equal(userDust.Denom, coin.Denom)
	suite.Require().Equal(userDust.Amount, sdk.OneDec().Sub(dust).String())
	suite.Require().Equal(dustResulted.String(), sdk.OneDec().Sub(dust).String())
}

func (suite *IntegrationTestSuite) TestCollectUserDust_PayerDust_AddedFromStorage() {
	addr := "addr1"
	storageDust, err := sdk.NewDecFromStr("0.1")
	suite.Require().Nil(err)
	ud1 := types.UserDust{
		Owner:  addr,
		Amount: storageDust.String(),
		Denom:  "ubze",
	}
	suite.k.SetUserDust(suite.ctx, ud1)

	coin := sdk.NewCoin("ubze", sdk.NewInt(100))
	dust, err := sdk.NewDecFromStr("0.35")
	suite.Require().Nil(err)

	res, userDust, dustResulted, err := suite.k.CollectUserDust(suite.ctx, addr, coin, dust, false)
	suite.Require().NoError(err)
	suite.Require().NotNil(userDust)
	suite.Require().Equal(res, coin.AddAmount(sdk.OneInt()))
	suite.Require().NotEqual(dustResulted, dust)
	suite.Require().Equal(userDust.Owner, addr)
	suite.Require().Equal(userDust.Denom, coin.Denom)
	suite.Require().Equal(userDust.Amount, sdk.OneDec().Sub(dust).Add(storageDust).String())
	suite.Require().Equal(dustResulted.String(), sdk.OneDec().Sub(dust).Add(storageDust).String())
}

func (suite *IntegrationTestSuite) TestCollectUserDust_PayerDust_PaidFromStorage() {
	addr := "addr1"
	storageDust, err := sdk.NewDecFromStr("0.36662")
	suite.Require().Nil(err)
	ud1 := types.UserDust{
		Owner:  addr,
		Amount: storageDust.String(),
		Denom:  "ubze",
	}
	suite.k.SetUserDust(suite.ctx, ud1)

	coin := sdk.NewCoin("ubze", sdk.NewInt(100))
	dust, err := sdk.NewDecFromStr("0.35")
	suite.Require().Nil(err)

	res, userDust, dustResulted, err := suite.k.CollectUserDust(suite.ctx, addr, coin, dust, false)
	suite.Require().NoError(err)
	suite.Require().NotNil(userDust)
	suite.Require().Equal(res, coin)
	suite.Require().NotEqual(dustResulted, dust)
	suite.Require().Equal(userDust.Owner, addr)
	suite.Require().Equal(userDust.Denom, coin.Denom)
	suite.Require().Equal(userDust.Amount, storageDust.Sub(dust).String())
	suite.Require().Equal(dustResulted.String(), storageDust.Sub(dust).String())
}

func (suite *IntegrationTestSuite) TestCollectUserDust_ReceiverFirstDust() {
	coin := sdk.NewCoin("ubze", sdk.NewInt(100))
	dust, err := sdk.NewDecFromStr("0.032121123123123")
	suite.Require().Nil(err)
	addr := "addr1"
	res, userDust, dustResulted, err := suite.k.CollectUserDust(suite.ctx, addr, coin, dust, true)
	suite.Require().NoError(err)
	suite.Require().NotNil(userDust)
	suite.Require().Equal(res, coin)
	suite.Require().Equal(dustResulted, dust)
	suite.Require().Equal(userDust.Owner, addr)
	suite.Require().Equal(userDust.Denom, coin.Denom)
	suite.Require().Equal(userDust.Amount, dust.String())
	suite.Require().Equal(dustResulted.String(), dust.String())
}

func (suite *IntegrationTestSuite) TestCollectUserDust_ReceiverDust_AddedFromStorage() {
	addr := "addr1"
	storageDust, err := sdk.NewDecFromStr("0.1")
	suite.Require().Nil(err)
	ud1 := types.UserDust{
		Owner:  addr,
		Amount: storageDust.String(),
		Denom:  "ubze",
	}
	suite.k.SetUserDust(suite.ctx, ud1)

	coin := sdk.NewCoin("ubze", sdk.NewInt(100))
	dust, err := sdk.NewDecFromStr("0.032121123123123")
	suite.Require().Nil(err)
	res, userDust, dustResulted, err := suite.k.CollectUserDust(suite.ctx, addr, coin, dust, true)
	suite.Require().NoError(err)
	suite.Require().NotNil(userDust)
	suite.Require().Equal(res, coin)
	suite.Require().NotEqual(dustResulted, dust)
	suite.Require().Equal(userDust.Owner, addr)
	suite.Require().Equal(userDust.Denom, coin.Denom)
	suite.Require().Equal(userDust.Amount, dust.Add(storageDust).String())
	suite.Require().Equal(dustResulted.String(), dust.Add(storageDust).String())
}

func (suite *IntegrationTestSuite) TestCollectUserDust_ReceiverDust_AddedFromStorageToCoin() {
	addr := "addr1"
	storageDust, err := sdk.NewDecFromStr("0.1")
	suite.Require().Nil(err)
	ud1 := types.UserDust{
		Owner:  addr,
		Amount: storageDust.String(),
		Denom:  "ubze",
	}
	suite.k.SetUserDust(suite.ctx, ud1)

	coin := sdk.NewCoin("ubze", sdk.NewInt(100))
	dust, err := sdk.NewDecFromStr("0.9")
	suite.Require().Nil(err)
	res, userDust, dustResulted, err := suite.k.CollectUserDust(suite.ctx, addr, coin, dust, true)
	suite.Require().NoError(err)
	suite.Require().NotNil(userDust)
	suite.Require().Equal(res, coin.AddAmount(sdk.OneInt()))
	suite.Require().NotEqual(dustResulted, dust)
	suite.Require().Equal(userDust.Owner, addr)
	suite.Require().Equal(userDust.Denom, coin.Denom)
	suite.Require().Equal(userDust.Amount, sdk.OneDec().Sub(dust.Add(storageDust)).String())
	suite.Require().Equal(dustResulted.String(), sdk.OneDec().Sub(dust.Add(storageDust)).String())
}
