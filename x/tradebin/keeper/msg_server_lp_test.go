package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"go.uber.org/mock/gomock"
	"strings"
	"testing"
)

func getValidLp() types.LiquidityPool {
	return types.LiquidityPool{
		Id:      "ubze_uusdc",
		Base:    "ubze",
		Quote:   "uusdc",
		Creator: getTestAddress(),
		LpDenom: "ulp_ubze_uusdc",
		Fee:     sdk.NewDecWithPrec(1, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.NewDec(1),
			Burner:    sdk.ZeroDec(),
			Providers: sdk.ZeroDec(),
			Liquidity: sdk.ZeroDec(),
		},
		ReserveBase:  sdk.NewInt(1000),
		ReserveQuote: sdk.NewInt(2000),
		Stable:       false,
	}
}

func getFeeDestinationString(burner, treasury, providers, liquidity string) string {
	return fmt.Sprintf(
		"{\"treasury\":\"%s\",\"burner\":\"%s\",\"providers\":\"%s\",\"liquidity\":\"%s\"}",
		treasury,
		burner,
		providers,
		liquidity,
	)
}

func getTestAccount() sdk.AccAddress {
	return sdk.AccAddress("addr1_______________")
}

func getTestAddress() string {
	return getTestAccount().String()
}

func (suite *IntegrationTestSuite) TestCreateLiquidityPool_InvalidAssets() {
	tc := []struct {
		Name          string
		Base          string
		Quote         string
		BaseExists    bool
		QuoteExists   bool
		ExpectedError error
	}{
		{
			Name:          "Base same as quote",
			Base:          "test",
			Quote:         "test",
			BaseExists:    false,
			QuoteExists:   false,
			ExpectedError: types.ErrInvalidDenom,
		},
		{
			Name:          "Base same as quote",
			Base:          "test",
			Quote:         "test2",
			BaseExists:    false,
			QuoteExists:   false,
			ExpectedError: types.ErrDenomHasNoSupply,
		},
		{
			Name:          "Base same as quote",
			Base:          "test",
			Quote:         "test2",
			BaseExists:    true,
			QuoteExists:   false,
			ExpectedError: types.ErrDenomHasNoSupply,
		},
	}

	goCtx := sdk.WrapSDKContext(suite.ctx)
	for _, c := range tc {
		suite.T().Run(c.Name, func(t *testing.T) {
			msg := &types.MsgCreateLiquidityPool{
				Base:    c.Base,
				Quote:   c.Quote,
				Creator: getTestAddress(),
			}
			if c.Base != c.Quote {
				suite.bankMock.EXPECT().HasSupply(gomock.Any(), c.Base).Return(c.BaseExists)
				if c.BaseExists {
					suite.bankMock.EXPECT().HasSupply(gomock.Any(), c.Quote).Return(c.QuoteExists)
				}
			}
			_, err := suite.msgServer.CreateLiquidityPool(goCtx, msg)
			suite.Require().NotNil(err)
			suite.Require().ErrorIs(err, c.ExpectedError)
		})
	}
}

func (suite *IntegrationTestSuite) TestCreateLiquidityPool_PoolAlreadyExists() {
	msg := &types.MsgCreateLiquidityPool{
		Base:    "def",
		Quote:   "abc",
		Creator: getTestAddress(),
	}
	suite.k.SetLiquidityPool(suite.ctx, types.LiquidityPool{
		Id: "abc_def",
	})
	goCtx := sdk.WrapSDKContext(suite.ctx)

	suite.bankMock.EXPECT().HasSupply(gomock.Any(), "def").Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), "abc").Return(true).Times(1)

	_, err := suite.msgServer.CreateLiquidityPool(goCtx, msg)
	suite.Require().NotNil(err)
	suite.Require().ErrorIs(err, types.ErrMarketAlreadyExists)
}

func (suite *IntegrationTestSuite) TestCreateLiquidityPool_InvalidFee() {
	tc := []struct {
		Name          string
		Fee           string
		ExpectedError error
	}{
		{
			Name:          "empty fee",
			Fee:           "",
			ExpectedError: sdkerrors.ErrInvalidCoins,
		},
		{
			Name:          "negative fee",
			Fee:           "-0.001",
			ExpectedError: sdkerrors.ErrInvalidCoins,
		},
		{
			Name:          "zero fee",
			Fee:           "0",
			ExpectedError: sdkerrors.ErrInvalidCoins,
		},
		{
			Name:          "fee too low",
			Fee:           "0.000999",
			ExpectedError: sdkerrors.ErrInvalidCoins,
		},
		{
			Name:          "fee too high",
			Fee:           "0.05009",
			ExpectedError: sdkerrors.ErrInvalidCoins,
		},
	}

	goCtx := sdk.WrapSDKContext(suite.ctx)
	for _, c := range tc {
		suite.T().Run(c.Name, func(t *testing.T) {
			msg := &types.MsgCreateLiquidityPool{
				Base:    "abc",
				Quote:   "def",
				Creator: getTestAddress(),
				Fee:     c.Fee,
			}
			suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Base).Return(true).Times(1)
			suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Quote).Return(true).Times(1)

			_, err := suite.msgServer.CreateLiquidityPool(goCtx, msg)
			suite.Require().NotNil(err)
			suite.Require().ErrorIs(err, c.ExpectedError)
		})
	}
}

func (suite *IntegrationTestSuite) TestCreateLiquidityPool_InvalidFeeDestination() {
	tc := []struct {
		Name          string
		FeeDest       string
		ExpectedError error
	}{
		{
			Name:          "parse fee error",
			FeeDest:       "",
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "parse fee error - NaN",
			FeeDest:       getFeeDestinationString("ceva_fin", "dasadsa", "0.25", "0.25"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is bigger than 1 - treasury high",
			FeeDest:       getFeeDestinationString("0.25", "0.251", "0.25", "0.25"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is bigger than 1 - treasury all",
			FeeDest:       getFeeDestinationString("0", "1.01", "0", "0"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is bigger than 1 - burner",
			FeeDest:       getFeeDestinationString("0.11", "0.9", "0", "0"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is bigger than 1 - providers",
			FeeDest:       getFeeDestinationString("0.1", "0.9", "0.001", "0"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is bigger than 1 - providers",
			FeeDest:       getFeeDestinationString("0.05", "0.9", "0.05", "0.0001"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is smaller than 1 - total 0.90001",
			FeeDest:       getFeeDestinationString("0", "0.9", "0", "0.0001"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is smaller than 1 - total 0.1",
			FeeDest:       getFeeDestinationString("0", "0", "0", "0.1"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is smaller than 1 - total 0",
			FeeDest:       getFeeDestinationString("0", "0", "0", "0.00"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "negative treasury fee",
			FeeDest:       getFeeDestinationString("1", "-0.1", "0.1", "0"),
			ExpectedError: types.ErrNegativeFeeDestination,
		},
		{
			Name:          "negative burner fee",
			FeeDest:       getFeeDestinationString("-0.1", "1", "0.1", "0"),
			ExpectedError: types.ErrNegativeFeeDestination,
		},
		{
			Name:          "negative providers fee",
			FeeDest:       getFeeDestinationString("1", "0.1", "-0.1", "0"),
			ExpectedError: types.ErrNegativeFeeDestination,
		},
		{
			Name:          "negative liquidity fee",
			FeeDest:       getFeeDestinationString("0.1", "1", "0.1", "-0.20"),
			ExpectedError: types.ErrNegativeFeeDestination,
		},
	}

	goCtx := sdk.WrapSDKContext(suite.ctx)
	for _, c := range tc {
		suite.T().Run(c.Name, func(t *testing.T) {
			msg := &types.MsgCreateLiquidityPool{
				Base:    "abc",
				Quote:   "def",
				Creator: getTestAddress(),
				Fee:     "0.002",
				FeeDest: c.FeeDest,
			}
			suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Base).Return(true).Times(1)
			suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Quote).Return(true).Times(1)

			_, err := suite.msgServer.CreateLiquidityPool(goCtx, msg)
			suite.Require().NotNil(err)
			suite.Require().ErrorIs(err, c.ExpectedError)
		})
	}
}

func (suite *IntegrationTestSuite) TestCreateLiquidityPool_InvalidReserves() {
	tc := []struct {
		Name         string
		InitialBase  sdk.Int
		InitialQuote sdk.Int
	}{
		{
			Name:         "zero base",
			InitialBase:  sdk.ZeroInt(),
			InitialQuote: sdk.NewInt(123456),
		},
		{
			Name:         "zero quote",
			InitialBase:  sdk.NewInt(2123321),
			InitialQuote: sdk.ZeroInt(),
		},
	}

	goCtx := sdk.WrapSDKContext(suite.ctx)
	for _, c := range tc {
		suite.T().Run(c.Name, func(t *testing.T) {
			msg := &types.MsgCreateLiquidityPool{
				Base:         "abc",
				Quote:        "def",
				Creator:      getTestAddress(),
				Fee:          "0.002",
				FeeDest:      getFeeDestinationString("0.25", "0.25", "0.25", "0.25"),
				InitialBase:  c.InitialBase,
				InitialQuote: c.InitialQuote,
			}
			suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Base).Return(true).Times(1)
			suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Quote).Return(true).Times(1)

			_, err := suite.msgServer.CreateLiquidityPool(goCtx, msg)
			suite.Require().NotNil(err)
		})
	}
}

func (suite *IntegrationTestSuite) TestCreateLiquidityPool_StableNotSupported() {
	//TODO: improve test when stable swap implemented (TODO: implement stable swap)
	goCtx := sdk.WrapSDKContext(suite.ctx)
	msg := &types.MsgCreateLiquidityPool{
		Base:         "abc",
		Quote:        "def",
		Creator:      getTestAddress(),
		Fee:          "0.002",
		FeeDest:      getFeeDestinationString("0.25", "0.25", "0.25", "0.25"),
		InitialBase:  sdk.NewInt(123),
		InitialQuote: sdk.NewInt(456),
		Stable:       true,
	}
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Base).Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Quote).Return(true).Times(1)

	_, err := suite.msgServer.CreateLiquidityPool(goCtx, msg)
	suite.Require().NotNil(err)
	suite.Require().ErrorIs(err, sdkerrors.ErrNotSupported)
}

func (suite *IntegrationTestSuite) TestCreateLiquidityPool_FundCommunityPoolErr() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	msg := &types.MsgCreateLiquidityPool{
		Base:         "abc",
		Quote:        "def",
		Creator:      getTestAddress(),
		Fee:          "0.002",
		FeeDest:      getFeeDestinationString("0.25", "0.25", "0.25", "0.25"),
		InitialBase:  sdk.NewInt(123),
		InitialQuote: sdk.NewInt(345),
	}

	createMarketFeeCoin, err := sdk.ParseCoinsNormalized(suite.k.CreateMarketFee(suite.ctx))
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Base).Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Quote).Return(true).Times(1)
	suite.distrMock.EXPECT().FundCommunityPool(gomock.Any(), createMarketFeeCoin, msg.GetCreatorAcc()).Return(fmt.Errorf("test error")).Times(1)

	_, err = suite.msgServer.CreateLiquidityPool(goCtx, msg)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestCreateLiquidityPool_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	msg := &types.MsgCreateLiquidityPool{
		Base:         "abc",
		Quote:        "def",
		Creator:      getTestAddress(),
		Fee:          "0.002",
		FeeDest:      getFeeDestinationString("0.25", "0.25", "0.25", "0.25"),
		InitialBase:  sdk.NewInt(123),
		InitialQuote: sdk.NewInt(345),
	}

	denomMetaData := banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "ulp_abc_def",
				Exponent: 0,
			},
			{
				Denom:    "lp_abc_def",
				Exponent: 6,
			},
		},
		Base:    "ulp_abc_def",
		Display: "lp_abc_def",
	}

	createMarketFeeCoin, err := sdk.ParseCoinsNormalized(suite.k.CreateMarketFee(suite.ctx))
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Base).Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Quote).Return(true).Times(1)
	suite.distrMock.EXPECT().FundCommunityPool(gomock.Any(), createMarketFeeCoin, msg.GetCreatorAcc()).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(suite.ctx, msg.GetCreatorAcc(), types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("abc", 123), sdk.NewInt64Coin("def", 345)))
	suite.bankMock.EXPECT().SetDenomMetaData(suite.ctx, denomMetaData)
	//205997572,801234723674372 - resulted shared from (sqrt(123 * 345)) * 1_000_000
	suite.bankMock.EXPECT().MintCoins(suite.ctx, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ulp_abc_def", 205997572)))

	res, err := suite.msgServer.CreateLiquidityPool(goCtx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NotEmpty(res.GetId())

	stored, found := suite.k.GetLiquidityPool(suite.ctx, res.GetId())
	suite.Require().True(found)
	suite.Require().NotNil(stored)
	suite.Require().Equal(res.GetId(), stored.GetId())
	suite.Require().Equal(msg.GetCreator(), stored.GetCreator())
	suite.Require().EqualValues(stored.ReserveBase.Int64(), 123)
	suite.Require().EqualValues(stored.ReserveQuote.Int64(), 345)
}

func (suite *IntegrationTestSuite) TestAddLiquidity_InvalidCreator() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	msg := &types.MsgAddLiquidity{
		Creator: "",
	}

	_, err := suite.msgServer.AddLiquidity(goCtx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), fmt.Sprintf("creator"))
}

func (suite *IntegrationTestSuite) TestAddLiquidity_PoolNotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	testLp := getValidLp()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator: getTestAddress(),
		PoolId:  "pool_1",
	}

	_, err := suite.msgServer.AddLiquidity(goCtx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "pool pool_1 not found")
}

func (suite *IntegrationTestSuite) TestAddLiquidity_InvalidCoins() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	testLp := getValidLp()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator: getTestAddress(),
		PoolId:  testLp.Id,
	}

	_, err := suite.msgServer.AddLiquidity(goCtx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "failed to calculate provided amounts")
}

func (suite *IntegrationTestSuite) TestAddLiquidity_CoinCaptureFailure() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	testLp := getValidLp()
	testAcc := getTestAccount()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator:     testAcc.String(),
		PoolId:      testLp.Id,
		BaseAmount:  sdk.NewInt(100),
		QuoteAmount: sdk.NewInt(200),
		MinLpTokens: sdk.NewInt(321),
	}

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 100), sdk.NewInt64Coin("uusdc", 200))).Times(1).Return(fmt.Errorf("invalid balance test"))

	_, err := suite.msgServer.AddLiquidity(goCtx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid balance test")
}

func (suite *IntegrationTestSuite) TestAddLiquidity_MissingLpTokenSupply() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	testLp := getValidLp()
	testAcc := getTestAccount()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator:     testAcc.String(),
		PoolId:      testLp.Id,
		BaseAmount:  sdk.NewInt(100),
		QuoteAmount: sdk.NewInt(200),
		MinLpTokens: sdk.NewInt(321),
	}

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 100), sdk.NewInt64Coin("uusdc", 200))).Times(1).Return(nil)
	suite.bankMock.EXPECT().GetSupply(suite.ctx, testLp.GetLpDenom()).Times(1).Return(sdk.NewCoin(testLp.GetLpDenom(), sdk.ZeroInt()))

	_, err := suite.msgServer.AddLiquidity(goCtx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "could not find supply for pool")
}

func (suite *IntegrationTestSuite) TestAddLiquidity_LpMintError() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	testLp := getValidLp()
	testAcc := getTestAccount()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator:     testAcc.String(),
		PoolId:      testLp.Id,
		BaseAmount:  sdk.NewInt(100),
		QuoteAmount: sdk.NewInt(200),
		MinLpTokens: sdk.NewInt(321),
	}

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 100), sdk.NewInt64Coin("uusdc", 200))).Times(1).Return(nil)
	suite.bankMock.EXPECT().GetSupply(suite.ctx, testLp.GetLpDenom()).Times(1).Return(sdk.NewCoin(testLp.GetLpDenom(), sdk.NewIntFromUint64(100)))

	suite.bankMock.EXPECT().MintCoins(suite.ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(testLp.GetLpDenom(), sdk.NewIntFromUint64(10)))).Return(fmt.Errorf("lp minting error"))

	_, err := suite.msgServer.AddLiquidity(goCtx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "lp minting error")
}

func (suite *IntegrationTestSuite) TestAddLiquidity_MinLpTokensNotMet() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	testLp := getValidLp()
	testAcc := getTestAccount()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator:     testAcc.String(),
		PoolId:      testLp.Id,
		BaseAmount:  sdk.NewInt(100),
		QuoteAmount: sdk.NewInt(200),
		MinLpTokens: sdk.NewInt(11),
	}

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 100), sdk.NewInt64Coin("uusdc", 200))).Times(1).Return(nil)
	suite.bankMock.EXPECT().GetSupply(suite.ctx, testLp.GetLpDenom()).Times(1).Return(sdk.NewCoin(testLp.GetLpDenom(), sdk.NewIntFromUint64(100)))
	suite.bankMock.EXPECT().MintCoins(suite.ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(testLp.GetLpDenom(), sdk.NewIntFromUint64(10)))).Return(nil)

	_, err := suite.msgServer.AddLiquidity(goCtx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "could not mint the minimum expected lp tokens")
}

func (suite *IntegrationTestSuite) TestAddLiquidity_ErrorOnSendingLpTokens() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	testLp := getValidLp()
	testAcc := getTestAccount()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator:     testAcc.String(),
		PoolId:      testLp.Id,
		BaseAmount:  sdk.NewInt(100),
		QuoteAmount: sdk.NewInt(200),
		MinLpTokens: sdk.NewInt(9),
	}

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 100), sdk.NewInt64Coin("uusdc", 200))).Times(1).Return(nil)
	suite.bankMock.EXPECT().GetSupply(suite.ctx, testLp.GetLpDenom()).Times(1).Return(sdk.NewCoin(testLp.GetLpDenom(), sdk.NewIntFromUint64(100)))

	minted := sdk.NewCoins(sdk.NewCoin(testLp.GetLpDenom(), sdk.NewIntFromUint64(10)))
	suite.bankMock.EXPECT().MintCoins(suite.ctx, types.ModuleName, minted).Return(nil)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, testAcc, minted).Times(1).Return(fmt.Errorf("error on sending lp tokens test"))

	_, err := suite.msgServer.AddLiquidity(goCtx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "error on sending lp tokens test")
}

func (suite *IntegrationTestSuite) TestAddLiquidity_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	testCases := []struct {
		name         string
		poolReserves struct {
			base  sdk.Int
			quote sdk.Int
		}
		userDeposit struct {
			base  sdk.Int
			quote sdk.Int
		}
		lpSupply        uint64
		minLpTokens     sdk.Int
		expectedDeposit struct {
			base  sdk.Int
			quote sdk.Int
		}
		expectedMint uint64
	}{
		{
			name: "balanced deposit - same ratio as pool",
			poolReserves: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(1000),
				quote: sdk.NewInt(2000),
			},
			userDeposit: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(100),
				quote: sdk.NewInt(200),
			},
			lpSupply:    1000,
			minLpTokens: sdk.NewInt(90),
			expectedDeposit: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(100),
				quote: sdk.NewInt(200),
			},
			expectedMint: 100, // 10% of reserves = 10% of LP supply
		},
		{
			name: "unbalanced deposit - base limiting",
			poolReserves: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(1000),
				quote: sdk.NewInt(3000),
			},
			userDeposit: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(100),
				quote: sdk.NewInt(500), // More than needed for 100 base
			},
			lpSupply:    1500,
			minLpTokens: sdk.NewInt(100),
			expectedDeposit: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(100),
				quote: sdk.NewInt(300), // Adjusted to maintain pool ratio
			},
			expectedMint: 150, // 10% of reserves = 10% of LP supply
		},
		{
			name: "unbalanced deposit - quote limiting",
			poolReserves: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(2000),
				quote: sdk.NewInt(1000),
			},
			userDeposit: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(500), // More than needed for 100 quote
				quote: sdk.NewInt(100),
			},
			lpSupply:    2000,
			minLpTokens: sdk.NewInt(150),
			expectedDeposit: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(200), // Adjusted to maintain pool ratio
				quote: sdk.NewInt(100),
			},
			expectedMint: 200, // 10% of reserves = 10% of LP supply
		},
		{
			name: "small deposit with uneven ratio",
			poolReserves: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(5000),
				quote: sdk.NewInt(7500),
			},
			userDeposit: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(50),
				quote: sdk.NewInt(80),
			},
			lpSupply:    10000,
			minLpTokens: sdk.NewInt(90),
			expectedDeposit: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(50),
				quote: sdk.NewInt(75), // Adjusted to maintain pool ratio
			},
			expectedMint: 100, // 1% of reserves = 1% of LP supply
		},
		{
			name: "large deposit",
			poolReserves: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(10000),
				quote: sdk.NewInt(20000),
			},
			userDeposit: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(10000), // Doubling the pool
				quote: sdk.NewInt(20000),
			},
			lpSupply:    5000,
			minLpTokens: sdk.NewInt(4000),
			expectedDeposit: struct {
				base  sdk.Int
				quote sdk.Int
			}{
				base:  sdk.NewInt(10000),
				quote: sdk.NewInt(20000),
			},
			expectedMint: 5000, // 100% increase = 100% of LP supply
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			testAcc := getTestAccount()
			// Reset the keeper and set up a fresh pool for each test
			testLp := types.LiquidityPool{
				Id:           "ubze_uusdc",
				Base:         "ubze",
				Quote:        "uusdc",
				LpDenom:      "lp_ubze_uusdc",
				ReserveBase:  tc.poolReserves.base,
				ReserveQuote: tc.poolReserves.quote,
				Creator:      testAcc.String(),
				Fee:          sdk.NewDecWithPrec(3, 3),
				Stable:       false,
			}

			suite.k.SetLiquidityPool(suite.ctx, testLp)

			msg := &types.MsgAddLiquidity{
				Creator:     testAcc.String(),
				PoolId:      testLp.Id,
				BaseAmount:  tc.userDeposit.base,
				QuoteAmount: tc.userDeposit.quote,
				MinLpTokens: tc.minLpTokens,
			}

			// Expected coins that will be sent from user to module
			expectedBaseCoin := sdk.NewCoin("ubze", tc.expectedDeposit.base)
			expectedQuoteCoin := sdk.NewCoin("uusdc", tc.expectedDeposit.quote)
			expectedCoins := sdk.NewCoins(expectedBaseCoin, expectedQuoteCoin)

			// Mock the bank keeper methods
			suite.bankMock.EXPECT().
				SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, expectedCoins).
				Times(1).
				Return(nil)

			suite.bankMock.EXPECT().
				GetSupply(suite.ctx, testLp.GetLpDenom()).
				Times(1).
				Return(sdk.NewCoin(testLp.GetLpDenom(), sdk.NewIntFromUint64(tc.lpSupply)))

			mintedCoins := sdk.NewCoins(sdk.NewCoin(testLp.GetLpDenom(), sdk.NewIntFromUint64(tc.expectedMint)))

			suite.bankMock.EXPECT().
				MintCoins(suite.ctx, types.ModuleName, mintedCoins).
				Times(1).
				Return(nil)

			suite.bankMock.EXPECT().
				SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, testAcc, mintedCoins).
				Times(1).
				Return(nil)

			// Capture the event that should be emitted
			eventManager := sdk.NewEventManager()
			ctx := suite.ctx.WithEventManager(eventManager)
			goCtx = sdk.WrapSDKContext(ctx)

			// Execute the handler
			resp, err := suite.msgServer.AddLiquidity(goCtx, msg)

			// Verify no errors and correct response
			suite.Require().NoError(err)
			suite.Require().NotNil(resp)
			suite.Require().EqualValues(tc.expectedMint, resp.MintedAmount.Uint64())

			// Verify the pool has been updated correctly
			updatedPool, found := suite.k.GetLiquidityPool(ctx, testLp.Id)
			suite.Require().True(found)
			suite.Require().Equal(tc.poolReserves.base.Add(tc.expectedDeposit.base).String(), updatedPool.ReserveBase.String())
			suite.Require().Equal(tc.poolReserves.quote.Add(tc.expectedDeposit.quote).String(), updatedPool.ReserveQuote.String())

			// Verify that the event was emitted correctly
			events := ctx.EventManager().Events()
			hasLiquidityAddedEvent := false

			for _, event := range events {
				if strings.Contains(event.Type, "LiquidityAddedEvent") {
					hasLiquidityAddedEvent = true
					for _, attr := range event.Attributes {
						switch string(attr.Key) {
						case "creator":
							suite.Require().Contains(string(attr.Value), msg.Creator)
						case "base_amount":
							suite.Require().Contains(fmt.Sprintf("%d", tc.expectedDeposit.base), string(attr.Value))
						case "quote_amount":
							suite.Require().Contains(fmt.Sprintf("%d", tc.expectedDeposit.quote), string(attr.Value))
						case "minted_amount":
							suite.Require().Contains(fmt.Sprintf("%d", tc.expectedMint), string(attr.Value))
						}
					}
				}
			}

			suite.Require().True(hasLiquidityAddedEvent, "LiquidityAddedEvent should be emitted")
		})
	}
}

func (suite *IntegrationTestSuite) TestRemoveLiquidity_Errors() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	testLp := types.LiquidityPool{
		Id:           "ubze_uusdc",
		Base:         "ubze",
		Quote:        "uusdc",
		LpDenom:      "lp_ubze_uusdc",
		ReserveBase:  sdk.NewInt(1000),
		ReserveQuote: sdk.NewInt(2000),
		Creator:      "creator",
		Fee:          sdk.NewDecWithPrec(3, 3),
		Stable:       false,
	}
	testAcc := getTestAccount()

	testCases := []struct {
		name          string
		setupMock     func()
		msg           *types.MsgRemoveLiquidity
		expectedError string
		errorType     error
		skipPoolSetup bool
	}{
		{
			name: "invalid creator address",
			msg: &types.MsgRemoveLiquidity{
				Creator:  "invalid_address",
				PoolId:   "ubze_uusdc",
				LpTokens: sdk.NewInt(100),
				MinBase:  sdk.NewInt(10),
				MinQuote: sdk.NewInt(20),
			},
			expectedError: "invalid creator address",
			errorType:     sdkerrors.ErrUnauthorized,
			setupMock:     func() {},
		},
		{
			name: "pool not found",
			msg: &types.MsgRemoveLiquidity{
				Creator:  testAcc.String(),
				PoolId:   "nonexistent_pool",
				LpTokens: sdk.NewInt(100),
				MinBase:  sdk.NewInt(10),
				MinQuote: sdk.NewInt(20),
			},
			expectedError: "pool nonexistent_pool not found",
			errorType:     types.ErrMarketNotFound,
			setupMock:     func() {},
			skipPoolSetup: true,
		},
		{
			name: "zero LP supply",
			msg: &types.MsgRemoveLiquidity{
				Creator:  testAcc.String(),
				PoolId:   "ubze_uusdc",
				LpTokens: sdk.NewInt(100),
				MinBase:  sdk.NewInt(10),
				MinQuote: sdk.NewInt(20),
			},
			expectedError: "could not find supply for pool",
			errorType:     types.ErrInvalidDenom,
			setupMock: func() {
				// Return zero supply
				suite.bankMock.EXPECT().
					GetSupply(suite.ctx, testLp.LpDenom).
					Return(sdk.NewCoin(testLp.LpDenom, sdk.ZeroInt())).
					Times(1)
			},
		},
		{
			name: "failed to send LP tokens",
			msg: &types.MsgRemoveLiquidity{
				Creator:  testAcc.String(),
				PoolId:   "ubze_uusdc",
				LpTokens: sdk.NewInt(100),
				MinBase:  sdk.NewInt(10),
				MinQuote: sdk.NewInt(20),
			},
			expectedError: "failed to send LP Tokens to module account",
			errorType:     nil, // This is a wrapped error so we don't check the type
			setupMock: func() {
				suite.bankMock.EXPECT().
					GetSupply(suite.ctx, testLp.LpDenom).
					Return(sdk.NewCoin(testLp.LpDenom, sdk.NewInt(1000))).
					Times(1)

				// Simulate failure when sending coins
				suite.bankMock.EXPECT().
					SendCoinsFromAccountToModule(
						suite.ctx,
						testAcc,
						types.ModuleName,
						sdk.NewCoins(sdk.NewInt64Coin(testLp.LpDenom, 100)),
					).
					Return(fmt.Errorf("insufficient funds")).
					Times(1)
			},
		},
		{
			name: "base amount too low",
			msg: &types.MsgRemoveLiquidity{
				Creator:  testAcc.String(),
				PoolId:   "ubze_uusdc",
				LpTokens: sdk.NewInt(100),
				MinBase:  sdk.NewInt(500), // Too high - would get only 100 (10%)
				MinQuote: sdk.NewInt(10),
			},
			expectedError: "base amount too low",
			errorType:     types.ErrResultedAmountTooLow,
			setupMock: func() {
				// Set up mocks for a successful flow up to the point of min amount validation
				lpSupply := sdk.NewInt(1000) // 1000 LP tokens total

				suite.bankMock.EXPECT().
					GetSupply(suite.ctx, testLp.LpDenom).
					Return(sdk.NewCoin(testLp.LpDenom, lpSupply)).
					Times(1)

				suite.bankMock.EXPECT().
					SendCoinsFromAccountToModule(
						suite.ctx,
						testAcc,
						types.ModuleName,
						sdk.NewCoins(sdk.NewInt64Coin(testLp.LpDenom, 100)),
					).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "quote amount too low",
			msg: &types.MsgRemoveLiquidity{
				Creator:  testAcc.String(),
				PoolId:   "ubze_uusdc",
				LpTokens: sdk.NewInt(100),
				MinBase:  sdk.NewInt(10),
				MinQuote: sdk.NewInt(500), // Too high - would get only 200 (10%)
			},
			expectedError: "quote amount too low",
			errorType:     types.ErrResultedAmountTooLow,
			setupMock: func() {
				// Set up mocks for a successful flow up to the point of min amount validation
				lpSupply := sdk.NewInt(1000) // 1000 LP tokens total

				suite.bankMock.EXPECT().
					GetSupply(suite.ctx, testLp.LpDenom).
					Return(sdk.NewCoin(testLp.LpDenom, lpSupply)).
					Times(1)

				suite.bankMock.EXPECT().
					SendCoinsFromAccountToModule(
						suite.ctx,
						testAcc,
						types.ModuleName,
						sdk.NewCoins(sdk.NewInt64Coin(testLp.LpDenom, 100)),
					).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "failed to burn LP tokens",
			msg: &types.MsgRemoveLiquidity{
				Creator:  testAcc.String(),
				PoolId:   "ubze_uusdc",
				LpTokens: sdk.NewInt(100),
				MinBase:  sdk.NewInt(10),
				MinQuote: sdk.NewInt(20),
			},
			expectedError: "failed to burn LP Tokens",
			errorType:     nil, // This is a wrapped error so we don't check the type
			setupMock: func() {
				lpSupply := sdk.NewInt(1000) // 1000 LP tokens total

				suite.bankMock.EXPECT().
					GetSupply(suite.ctx, testLp.LpDenom).
					Return(sdk.NewCoin(testLp.LpDenom, lpSupply)).
					Times(1)

				suite.bankMock.EXPECT().
					SendCoinsFromAccountToModule(
						suite.ctx,
						testAcc,
						types.ModuleName,
						sdk.NewCoins(sdk.NewInt64Coin(testLp.LpDenom, 100)),
					).
					Return(nil).
					Times(1)

				// Simulate failure when burning coins
				suite.bankMock.EXPECT().
					BurnCoins(
						suite.ctx,
						types.ModuleName,
						sdk.NewCoins(sdk.NewInt64Coin(testLp.LpDenom, 100)),
					).
					Return(fmt.Errorf("failed to burn")).
					Times(1)
			},
		},
		{
			name: "failed to send tokens to user",
			msg: &types.MsgRemoveLiquidity{
				Creator:  testAcc.String(),
				PoolId:   "ubze_uusdc",
				LpTokens: sdk.NewInt(100),
				MinBase:  sdk.NewInt(10),
				MinQuote: sdk.NewInt(20),
			},
			expectedError: "failed to send resulted coins to user account",
			errorType:     nil, // This is a wrapped error so we don't check the type
			setupMock: func() {
				lpSupply := sdk.NewInt(1000) // 1000 LP tokens total

				suite.bankMock.EXPECT().
					GetSupply(suite.ctx, testLp.LpDenom).
					Return(sdk.NewCoin(testLp.LpDenom, lpSupply)).
					Times(1)

				suite.bankMock.EXPECT().
					SendCoinsFromAccountToModule(
						suite.ctx,
						testAcc,
						types.ModuleName,
						sdk.NewCoins(sdk.NewInt64Coin(testLp.LpDenom, 100)),
					).
					Return(nil).
					Times(1)

				suite.bankMock.EXPECT().
					BurnCoins(
						suite.ctx,
						types.ModuleName,
						sdk.NewCoins(sdk.NewInt64Coin(testLp.LpDenom, 100)),
					).
					Return(nil).
					Times(1)

				// Calculate expected amounts (10% of pool)
				baseAmount := sdk.NewInt(100)  // 10% of 1000
				quoteAmount := sdk.NewInt(200) // 10% of 2000

				// Simulate failure when sending tokens to user
				suite.bankMock.EXPECT().
					SendCoinsFromModuleToAccount(
						suite.ctx,
						types.ModuleName,
						testAcc,
						sdk.NewCoins(
							sdk.NewCoin(testLp.Base, baseAmount),
							sdk.NewCoin(testLp.Quote, quoteAmount),
						),
					).
					Return(fmt.Errorf("insufficient module balance")).
					Times(1)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Set up pool in the keeper if needed
			if !tc.skipPoolSetup {
				suite.k.SetLiquidityPool(suite.ctx, testLp)
			}

			// Setup mocks
			tc.setupMock()

			// Execute the handler
			_, err := suite.msgServer.RemoveLiquidity(goCtx, tc.msg)

			// Verify error
			suite.Require().Error(err)
			suite.Require().Contains(err.Error(), tc.expectedError)
			if tc.errorType != nil {
				suite.Require().ErrorIs(err, tc.errorType)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestRemoveLiquidity_Success() {
	testAcc := getTestAccount()

	testCases := []struct {
		name          string
		pool          types.LiquidityPool
		lpTokens      sdk.Int
		lpSupply      uint64
		minBase       sdk.Int
		minQuote      sdk.Int
		expectedBase  uint64
		expectedQuote uint64
	}{
		{
			name: "remove 10% of liquidity",
			pool: types.LiquidityPool{
				Id:           "ubze_uusdc",
				Base:         "ubze",
				Quote:        "uusdc",
				LpDenom:      "lp_ubze_uusdc",
				ReserveBase:  sdk.NewInt(1000),
				ReserveQuote: sdk.NewInt(2000),
				Creator:      "creator",
				Fee:          sdk.NewDecWithPrec(3, 3),
				Stable:       false,
			},
			lpTokens:      sdk.NewInt(100),
			lpSupply:      1000,            // 10% removal
			minBase:       sdk.NewInt(90),  // Slightly below expected
			minQuote:      sdk.NewInt(190), // Slightly below expected
			expectedBase:  100,             // 10% of 1000
			expectedQuote: 200,             // 10% of 2000
		},
		{
			name: "remove 50% of liquidity",
			pool: types.LiquidityPool{
				Id:           "ubze_uusdc",
				Base:         "ubze",
				Quote:        "uusdc",
				LpDenom:      "lp_ubze_uusdc",
				ReserveBase:  sdk.NewInt(5000),
				ReserveQuote: sdk.NewInt(10000),
				Creator:      "creator",
				Fee:          sdk.NewDecWithPrec(3, 3),
				Stable:       false,
			},
			lpTokens:      sdk.NewInt(500),
			lpSupply:      1000,             // 50% removal
			minBase:       sdk.NewInt(2400), // Slightly below expected
			minQuote:      sdk.NewInt(4900), // Slightly below expected
			expectedBase:  2500,             // 50% of 5000
			expectedQuote: 5000,             // 50% of 10000
		},
		{
			name: "remove small amount of liquidity",
			pool: types.LiquidityPool{
				Id:           "ubze_uusdc",
				Base:         "ubze",
				Quote:        "uusdc",
				LpDenom:      "lp_ubze_uusdc",
				ReserveBase:  sdk.NewInt(10000),
				ReserveQuote: sdk.NewInt(20000),
				Creator:      "creator",
				Fee:          sdk.NewDecWithPrec(3, 3),
				Stable:       false,
			},
			lpTokens:      sdk.NewInt(1),
			lpSupply:      1000,           // 0.1% removal
			minBase:       sdk.NewInt(9),  // Slightly below expected
			minQuote:      sdk.NewInt(19), // Slightly below expected
			expectedBase:  10,             // 0.1% of 10000
			expectedQuote: 20,             // 0.1% of 20000
		},
		{
			name: "remove all liquidity",
			pool: types.LiquidityPool{
				Id:           "ubze_uusdc",
				Base:         "ubze",
				Quote:        "uusdc",
				LpDenom:      "lp_ubze_uusdc",
				ReserveBase:  sdk.NewInt(3000),
				ReserveQuote: sdk.NewInt(6000),
				Creator:      "creator",
				Fee:          sdk.NewDecWithPrec(3, 3),
				Stable:       false,
			},
			lpTokens:      sdk.NewInt(1000),
			lpSupply:      1000,             // 100% removal
			minBase:       sdk.NewInt(2900), // Slightly below expected
			minQuote:      sdk.NewInt(5900), // Slightly below expected
			expectedBase:  3000,             // 100% of 3000
			expectedQuote: 6000,             // 100% of 6000
		},
		{
			name: "uneven pool reserves",
			pool: types.LiquidityPool{
				Id:           "ubze_uusdc",
				Base:         "ubze",
				Quote:        "uusdc",
				LpDenom:      "lp_ubze_uusdc",
				ReserveBase:  sdk.NewInt(1500),
				ReserveQuote: sdk.NewInt(4500),
				Creator:      "creator",
				Fee:          sdk.NewDecWithPrec(3, 3),
				Stable:       false,
			},
			lpTokens:      sdk.NewInt(200),
			lpSupply:      1000,            // 20% removal
			minBase:       sdk.NewInt(290), // Slightly below expected
			minQuote:      sdk.NewInt(890), // Slightly below expected
			expectedBase:  300,             // 20% of 1500
			expectedQuote: 900,             // 20% of 4500
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup the pool
			suite.k.SetLiquidityPool(suite.ctx, tc.pool)

			// Create message
			msg := &types.MsgRemoveLiquidity{
				Creator:  testAcc.String(),
				PoolId:   tc.pool.Id,
				LpTokens: tc.lpTokens,
				MinBase:  tc.minBase,
				MinQuote: tc.minQuote,
			}

			// Setup mocks
			lpSupply := sdk.NewInt(int64(tc.lpSupply))
			baseAmount := sdk.NewInt(int64(tc.expectedBase))
			quoteAmount := sdk.NewInt(int64(tc.expectedQuote))

			suite.bankMock.EXPECT().
				GetSupply(suite.ctx, tc.pool.LpDenom).
				Return(sdk.NewCoin(tc.pool.LpDenom, lpSupply)).
				Times(1)

			suite.bankMock.EXPECT().
				SendCoinsFromAccountToModule(
					suite.ctx,
					testAcc,
					types.ModuleName,
					sdk.NewCoins(sdk.NewCoin(tc.pool.LpDenom, tc.lpTokens)),
				).
				Return(nil).
				Times(1)

			suite.bankMock.EXPECT().
				BurnCoins(
					suite.ctx,
					types.ModuleName,
					sdk.NewCoins(sdk.NewCoin(tc.pool.LpDenom, tc.lpTokens)),
				).
				Return(nil).
				Times(1)

			suite.bankMock.EXPECT().
				SendCoinsFromModuleToAccount(
					suite.ctx,
					types.ModuleName,
					testAcc,
					sdk.NewCoins(
						sdk.NewCoin(tc.pool.Base, baseAmount),
						sdk.NewCoin(tc.pool.Quote, quoteAmount),
					),
				).
				Return(nil).
				Times(1)

			// Capture events
			eventManager := sdk.NewEventManager()
			ctx := suite.ctx.WithEventManager(eventManager)
			wrappedCtx := sdk.WrapSDKContext(ctx)

			// Execute the handler
			resp, err := suite.msgServer.RemoveLiquidity(wrappedCtx, msg)

			// Verify success
			suite.Require().NoError(err)
			suite.Require().NotNil(resp)
			suite.Require().Equal(tc.expectedBase, resp.Base.Uint64())
			suite.Require().Equal(tc.expectedQuote, resp.Quote.Uint64())

			// Verify the pool has been updated correctly
			updatedPool, found := suite.k.GetLiquidityPool(ctx, tc.pool.Id)
			suite.Require().True(found)
			suite.Require().Equal(tc.pool.ReserveBase.Uint64()-tc.expectedBase, updatedPool.ReserveBase.Uint64())
			suite.Require().Equal(tc.pool.ReserveQuote.Uint64()-tc.expectedQuote, updatedPool.ReserveQuote.Uint64())

			// Verify that the event was emitted correctly
			events := ctx.EventManager().Events()
			hasLiquidityRemovedEvent := false

			for _, event := range events {
				if strings.Contains(event.Type, "LiquidityRemovedEvent") {
					hasLiquidityRemovedEvent = true
					for _, attr := range event.Attributes {
						switch string(attr.Key) {
						case "creator":
							suite.Require().Contains(string(attr.Value), msg.Creator)
						case "base_amount":
							suite.Require().Equal(fmt.Sprintf("%d", tc.expectedBase), string(attr.Value))
						case "quote_amount":
							suite.Require().Equal(fmt.Sprintf("%d", tc.expectedQuote), string(attr.Value))
						}
					}
				}
			}

			suite.Require().True(hasLiquidityRemovedEvent, "LiquidityRemovedEvent should be emitted")
		})
	}
}
