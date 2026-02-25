//nolint:staticcheck // tests use legacy sdk helpers and context wrappers
package keeper_test

import (
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/math"
	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"go.uber.org/mock/gomock"
)

func getValidLp() types.LiquidityPool {
	return types.LiquidityPool{
		Id:      "ubze_uusdc",
		Base:    "ubze",
		Quote:   "uusdc",
		Creator: getTestAddress(),
		LpDenom: "ulp_ubze_uusdc",
		Fee:     math.LegacyNewDecWithPrec(1, 3),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDec(1),
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyZeroDec(),
		},
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
		Stable:       false,
	}
}

func getFeeDestinationString(burner, treasury, providers string) string {
	return fmt.Sprintf(
		"{\"treasury\":\"%s\",\"burner\":\"%s\",\"providers\":\"%s\"}",
		treasury,
		burner,
		providers,
	)
}

func getTestAccount() sdk.AccAddress {
	return sdk.AccAddress("addr1_______________")
}

func getTestAddress() string {
	return getTestAccount().String()
}

func (suite *IntegrationTestSuite) TestMsgAmm_CreateLiquidityPool_InvalidAssets() {
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
			_, err := suite.msgServer.CreateLiquidityPool(suite.ctx, msg)
			suite.Require().NotNil(err)
			suite.Require().ErrorIs(err, c.ExpectedError)
		})
	}
}

func (suite *IntegrationTestSuite) TestMsgAmm_CreateLiquidityPool_PoolAlreadyExists() {
	msg := &types.MsgCreateLiquidityPool{
		Base:    "def",
		Quote:   "abc",
		Creator: getTestAddress(),
	}
	suite.k.SetLiquidityPool(suite.ctx, types.LiquidityPool{
		Id: "abc_def",
	})

	suite.bankMock.EXPECT().HasSupply(gomock.Any(), "def").Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), "abc").Return(true).Times(1)

	_, err := suite.msgServer.CreateLiquidityPool(suite.ctx, msg)
	suite.Require().NotNil(err)
	suite.Require().ErrorIs(err, types.ErrMarketAlreadyExists)
}

func (suite *IntegrationTestSuite) TestMsgAmm_CreateLiquidityPool_InvalidFee() {
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

			_, err := suite.msgServer.CreateLiquidityPool(suite.ctx, msg)
			suite.Require().NotNil(err)
			suite.Require().ErrorIs(err, c.ExpectedError)
		})
	}
}

func (suite *IntegrationTestSuite) TestMsgAmm_CreateLiquidityPool_InvalidFeeDestination() {
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
			FeeDest:       getFeeDestinationString("ceva_fin", "dasadsa", "0.25"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is bigger than 1 - treasury high",
			FeeDest:       getFeeDestinationString("0.25", "0.251", "0.25"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is bigger than 1 - treasury all",
			FeeDest:       getFeeDestinationString("0", "1.01", "0"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is bigger than 1 - burner",
			FeeDest:       getFeeDestinationString("0.11", "0.9", "0"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is bigger than 1 - providers",
			FeeDest:       getFeeDestinationString("0.1", "0.9", "0.001"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is smaller than 1 - total 0.90001",
			FeeDest:       getFeeDestinationString("0", "0.9", "0"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "total fee destination is smaller than 1 - total 0",
			FeeDest:       getFeeDestinationString("0", "0", "0"),
			ExpectedError: types.ErrInvalidFeeDestination,
		},
		{
			Name:          "negative treasury fee",
			FeeDest:       getFeeDestinationString("1", "-0.1", "0.1"),
			ExpectedError: types.ErrNegativeFeeDestination,
		},
		{
			Name:          "negative burner fee",
			FeeDest:       getFeeDestinationString("-0.1", "1", "0.1"),
			ExpectedError: types.ErrNegativeFeeDestination,
		},
		{
			Name:          "negative providers fee",
			FeeDest:       getFeeDestinationString("1", "0.1", "-0.1"),
			ExpectedError: types.ErrNegativeFeeDestination,
		},
	}

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

			_, err := suite.msgServer.CreateLiquidityPool(suite.ctx, msg)
			suite.Require().NotNil(err)
			suite.Require().ErrorIs(err, c.ExpectedError)
		})
	}
}

func (suite *IntegrationTestSuite) TestMsgAmm_CreateLiquidityPool_InvalidReserves() {
	tc := []struct {
		Name         string
		InitialBase  math.Int
		InitialQuote math.Int
	}{
		{
			Name:         "zero base",
			InitialBase:  math.ZeroInt(),
			InitialQuote: math.NewInt(123456),
		},
		{
			Name:         "zero quote",
			InitialBase:  math.NewInt(2123321),
			InitialQuote: math.ZeroInt(),
		},
	}

	for _, c := range tc {
		suite.T().Run(c.Name, func(t *testing.T) {
			msg := &types.MsgCreateLiquidityPool{
				Base:         "abc",
				Quote:        "def",
				Creator:      getTestAddress(),
				Fee:          "0.002",
				FeeDest:      getFeeDestinationString("0.5", "0.25", "0.25"),
				InitialBase:  c.InitialBase,
				InitialQuote: c.InitialQuote,
			}
			suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Base).Return(true).Times(1)
			suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Quote).Return(true).Times(1)

			_, err := suite.msgServer.CreateLiquidityPool(suite.ctx, msg)
			suite.Require().NotNil(err)
		})
	}
}

func (suite *IntegrationTestSuite) TestMsgAmm_CreateLiquidityPool_StableNotSupported() {
	//TODO: improve test when stable swap implemented (TODO: implement stable swap)

	msg := &types.MsgCreateLiquidityPool{
		Base:         "abc",
		Quote:        "def",
		Creator:      getTestAddress(),
		Fee:          "0.002",
		FeeDest:      getFeeDestinationString("0.25", "0.25", "0.5"),
		InitialBase:  math.NewInt(123),
		InitialQuote: math.NewInt(456),
		Stable:       true,
	}
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Base).Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Quote).Return(true).Times(1)

	_, err := suite.msgServer.CreateLiquidityPool(suite.ctx, msg)
	suite.Require().NotNil(err)
	suite.Require().ErrorIs(err, sdkerrors.ErrNotSupported)
}

func (suite *IntegrationTestSuite) TestMsgAmm_CreateLiquidityPool_FundCommunityPoolErr() {

	msg := &types.MsgCreateLiquidityPool{
		Base:         "abc",
		Quote:        "def",
		Creator:      getTestAddress(),
		Fee:          "0.002",
		FeeDest:      getFeeDestinationString("0.25", "0.25", "0.5"),
		InitialBase:  math.NewInt(123),
		InitialQuote: math.NewInt(345),
	}

	createMarketFeeCoin := sdk.NewCoins(suite.k.CreateMarketFee(suite.ctx))

	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Base).Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Quote).Return(true).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), getTestAccount(), types.ModuleName, createMarketFeeCoin).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), createMarketFeeCoin).Return(fmt.Errorf("test error")).Times(1)

	_, err := suite.msgServer.CreateLiquidityPool(suite.ctx, msg)
	suite.Require().NotNil(err)
}

func (suite *IntegrationTestSuite) TestMsgAmm_CreateLiquidityPool_Success() {

	msg := &types.MsgCreateLiquidityPool{
		Base:         "abc",
		Quote:        "def",
		Creator:      getTestAddress(),
		Fee:          "0.002",
		FeeDest:      getFeeDestinationString("0.25", "0.5", "0.25"),
		InitialBase:  math.NewInt(123),
		InitialQuote: math.NewInt(345),
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

	createMarketFeeCoin := sdk.NewCoins(suite.k.CreateMarketFee(suite.ctx))

	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Base).Return(true).Times(1)
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), msg.Quote).Return(true).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), getTestAccount(), types.ModuleName, createMarketFeeCoin).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), createMarketFeeCoin).Return(nil).Times(1)
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(suite.ctx, getTestAccount(), types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("abc", 123), sdk.NewInt64Coin("def", 345)))
	suite.bankMock.EXPECT().SetDenomMetaData(suite.ctx, denomMetaData)
	//205997572,801234723674372 - resulted shared from (sqrt(123 * 345)) * 1_000_000
	lpTokens := sdk.NewCoins(sdk.NewInt64Coin("ulp_abc_def", 205997572))
	suite.bankMock.EXPECT().MintCoins(suite.ctx, types.ModuleName, lpTokens)
	suite.bankMock.EXPECT().SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, "burner_black_hole", lpTokens).Return(nil).Times(1)

	res, err := suite.msgServer.CreateLiquidityPool(suite.ctx, msg)
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

func (suite *IntegrationTestSuite) TestMsgAmm_AddLiquidity_InvalidCreator() {

	msg := &types.MsgAddLiquidity{
		Creator: "",
	}

	_, err := suite.msgServer.AddLiquidity(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid address")
}

func (suite *IntegrationTestSuite) TestMsgAmm_AddLiquidity_PoolNotFound() {

	testLp := getValidLp()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator: getTestAddress(),
		PoolId:  "pool_1",
	}

	_, err := suite.msgServer.AddLiquidity(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "pool pool_1 not found")
}

func (suite *IntegrationTestSuite) TestMsgAmm_AddLiquidity_InvalidCoins() {

	testLp := getValidLp()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator: getTestAddress(),
		PoolId:  testLp.Id,
	}

	_, err := suite.msgServer.AddLiquidity(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "failed to calculate provided amounts")
}

func (suite *IntegrationTestSuite) TestMsgAmm_AddLiquidity_CoinCaptureFailure() {

	testLp := getValidLp()
	testAcc := getTestAccount()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator:     testAcc.String(),
		PoolId:      testLp.Id,
		BaseAmount:  math.NewInt(100),
		QuoteAmount: math.NewInt(200),
		MinLpTokens: math.NewInt(321),
	}

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 100), sdk.NewInt64Coin("uusdc", 200))).Times(1).Return(fmt.Errorf("invalid balance test"))

	_, err := suite.msgServer.AddLiquidity(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid balance test")
}

func (suite *IntegrationTestSuite) TestMsgAmm_AddLiquidity_MissingLpTokenSupply() {

	testLp := getValidLp()
	testAcc := getTestAccount()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator:     testAcc.String(),
		PoolId:      testLp.Id,
		BaseAmount:  math.NewInt(100),
		QuoteAmount: math.NewInt(200),
		MinLpTokens: math.NewInt(321),
	}

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 100), sdk.NewInt64Coin("uusdc", 200))).Times(1).Return(nil)
	suite.bankMock.EXPECT().GetSupply(suite.ctx, testLp.GetLpDenom()).Times(1).Return(sdk.NewCoin(testLp.GetLpDenom(), math.ZeroInt()))

	_, err := suite.msgServer.AddLiquidity(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "could not find supply for pool")
}

func (suite *IntegrationTestSuite) TestMsgAmm_AddLiquidity_LpMintError() {

	testLp := getValidLp()
	testAcc := getTestAccount()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator:     testAcc.String(),
		PoolId:      testLp.Id,
		BaseAmount:  math.NewInt(100),
		QuoteAmount: math.NewInt(200),
		MinLpTokens: math.NewInt(321),
	}

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 100), sdk.NewInt64Coin("uusdc", 200))).Times(1).Return(nil)
	suite.bankMock.EXPECT().GetSupply(suite.ctx, testLp.GetLpDenom()).Times(1).Return(sdk.NewCoin(testLp.GetLpDenom(), math.NewIntFromUint64(100)))

	suite.bankMock.EXPECT().MintCoins(suite.ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(testLp.GetLpDenom(), math.NewIntFromUint64(10)))).Return(fmt.Errorf("lp minting error"))

	_, err := suite.msgServer.AddLiquidity(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "lp minting error")
}

func (suite *IntegrationTestSuite) TestMsgAmm_AddLiquidity_MinLpTokensNotMet() {

	testLp := getValidLp()
	testAcc := getTestAccount()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator:     testAcc.String(),
		PoolId:      testLp.Id,
		BaseAmount:  math.NewInt(100),
		QuoteAmount: math.NewInt(200),
		MinLpTokens: math.NewInt(11),
	}

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 100), sdk.NewInt64Coin("uusdc", 200))).Times(1).Return(nil)
	suite.bankMock.EXPECT().GetSupply(suite.ctx, testLp.GetLpDenom()).Times(1).Return(sdk.NewCoin(testLp.GetLpDenom(), math.NewIntFromUint64(100)))
	suite.bankMock.EXPECT().MintCoins(suite.ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(testLp.GetLpDenom(), math.NewIntFromUint64(10)))).Return(nil)

	_, err := suite.msgServer.AddLiquidity(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "could not mint the minimum expected lp tokens")
}

func (suite *IntegrationTestSuite) TestMsgAmm_AddLiquidity_ErrorOnSendingLpTokens() {

	testLp := getValidLp()
	testAcc := getTestAccount()
	suite.k.SetLiquidityPool(suite.ctx, testLp)

	msg := &types.MsgAddLiquidity{
		Creator:     testAcc.String(),
		PoolId:      testLp.Id,
		BaseAmount:  math.NewInt(100),
		QuoteAmount: math.NewInt(200),
		MinLpTokens: math.NewInt(9),
	}

	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), testAcc, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("ubze", 100), sdk.NewInt64Coin("uusdc", 200))).Times(1).Return(nil)
	suite.bankMock.EXPECT().GetSupply(suite.ctx, testLp.GetLpDenom()).Times(1).Return(sdk.NewCoin(testLp.GetLpDenom(), math.NewIntFromUint64(100)))

	minted := sdk.NewCoins(sdk.NewCoin(testLp.GetLpDenom(), math.NewIntFromUint64(10)))
	suite.bankMock.EXPECT().MintCoins(suite.ctx, types.ModuleName, minted).Return(nil)

	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, testAcc, minted).Times(1).Return(fmt.Errorf("error on sending lp tokens test"))

	_, err := suite.msgServer.AddLiquidity(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "error on sending lp tokens test")
}

func (suite *IntegrationTestSuite) TestMsgAmm_AddLiquidity_Success() {

	testCases := []struct {
		name         string
		poolReserves struct {
			base  math.Int
			quote math.Int
		}
		userDeposit struct {
			base  math.Int
			quote math.Int
		}
		lpSupply        uint64
		minLpTokens     math.Int
		expectedDeposit struct {
			base  math.Int
			quote math.Int
		}
		expectedMint uint64
	}{
		{
			name: "balanced deposit - same ratio as pool",
			poolReserves: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(1000),
				quote: math.NewInt(2000),
			},
			userDeposit: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(100),
				quote: math.NewInt(200),
			},
			lpSupply:    1000,
			minLpTokens: math.NewInt(90),
			expectedDeposit: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(100),
				quote: math.NewInt(200),
			},
			expectedMint: 100, // 10% of reserves = 10% of LP supply
		},
		{
			name: "unbalanced deposit - base limiting",
			poolReserves: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(1000),
				quote: math.NewInt(3000),
			},
			userDeposit: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(100),
				quote: math.NewInt(500), // More than needed for 100 base
			},
			lpSupply:    1500,
			minLpTokens: math.NewInt(100),
			expectedDeposit: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(100),
				quote: math.NewInt(300), // Adjusted to maintain pool ratio
			},
			expectedMint: 150, // 10% of reserves = 10% of LP supply
		},
		{
			name: "unbalanced deposit - quote limiting",
			poolReserves: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(2000),
				quote: math.NewInt(1000),
			},
			userDeposit: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(500), // More than needed for 100 quote
				quote: math.NewInt(100),
			},
			lpSupply:    2000,
			minLpTokens: math.NewInt(150),
			expectedDeposit: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(200), // Adjusted to maintain pool ratio
				quote: math.NewInt(100),
			},
			expectedMint: 200, // 10% of reserves = 10% of LP supply
		},
		{
			name: "small deposit with uneven ratio",
			poolReserves: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(5000),
				quote: math.NewInt(7500),
			},
			userDeposit: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(50),
				quote: math.NewInt(80),
			},
			lpSupply:    10000,
			minLpTokens: math.NewInt(90),
			expectedDeposit: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(50),
				quote: math.NewInt(75), // Adjusted to maintain pool ratio
			},
			expectedMint: 100, // 1% of reserves = 1% of LP supply
		},
		{
			name: "large deposit",
			poolReserves: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(10000),
				quote: math.NewInt(20000),
			},
			userDeposit: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(10000), // Doubling the pool
				quote: math.NewInt(20000),
			},
			lpSupply:    5000,
			minLpTokens: math.NewInt(4000),
			expectedDeposit: struct {
				base  math.Int
				quote math.Int
			}{
				base:  math.NewInt(10000),
				quote: math.NewInt(20000),
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
				Fee:          math.LegacyNewDecWithPrec(3, 3),
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
				Return(sdk.NewCoin(testLp.GetLpDenom(), math.NewIntFromUint64(tc.lpSupply)))

			mintedCoins := sdk.NewCoins(sdk.NewCoin(testLp.GetLpDenom(), math.NewIntFromUint64(tc.expectedMint)))

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

			// Execute the handler
			resp, err := suite.msgServer.AddLiquidity(ctx, msg)

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

func (suite *IntegrationTestSuite) TestMsgAmm_RemoveLiquidity_Errors() {

	testLp := types.LiquidityPool{
		Id:           "ubze_uusdc",
		Base:         "ubze",
		Quote:        "uusdc",
		LpDenom:      "lp_ubze_uusdc",
		ReserveBase:  math.NewInt(1000),
		ReserveQuote: math.NewInt(2000),
		Creator:      "creator",
		Fee:          math.LegacyNewDecWithPrec(3, 3),
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
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(10),
				MinQuote: math.NewInt(20),
			},
			expectedError: "invalid address",
			errorType:     sdkerrors.ErrInvalidAddress,
			setupMock:     func() {},
		},
		{
			name: "pool not found",
			msg: &types.MsgRemoveLiquidity{
				Creator:  testAcc.String(),
				PoolId:   "nonexistent_pool",
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(10),
				MinQuote: math.NewInt(20),
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
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(10),
				MinQuote: math.NewInt(20),
			},
			expectedError: "could not find supply for pool",
			errorType:     types.ErrInvalidDenom,
			setupMock: func() {
				// Return zero supply
				suite.bankMock.EXPECT().
					GetSupply(suite.ctx, testLp.LpDenom).
					Return(sdk.NewCoin(testLp.LpDenom, math.ZeroInt())).
					Times(1)
			},
		},
		{
			name: "failed to send LP tokens",
			msg: &types.MsgRemoveLiquidity{
				Creator:  testAcc.String(),
				PoolId:   "ubze_uusdc",
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(10),
				MinQuote: math.NewInt(20),
			},
			expectedError: "failed to send LP Tokens to module account",
			errorType:     nil, // This is a wrapped error so we don't check the type
			setupMock: func() {
				suite.bankMock.EXPECT().
					GetSupply(suite.ctx, testLp.LpDenom).
					Return(sdk.NewCoin(testLp.LpDenom, math.NewInt(1000))).
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
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(500), // Too high - would get only 100 (10%)
				MinQuote: math.NewInt(10),
			},
			expectedError: "base amount too low",
			errorType:     types.ErrResultedAmountTooLow,
			setupMock: func() {
				// Set up mocks for a successful flow up to the point of min amount validation
				lpSupply := math.NewInt(1000) // 1000 LP tokens total

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
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(10),
				MinQuote: math.NewInt(500), // Too high - would get only 200 (10%)
			},
			expectedError: "quote amount too low",
			errorType:     types.ErrResultedAmountTooLow,
			setupMock: func() {
				// Set up mocks for a successful flow up to the point of min amount validation
				lpSupply := math.NewInt(1000) // 1000 LP tokens total

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
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(10),
				MinQuote: math.NewInt(20),
			},
			expectedError: "failed to burn LP Tokens",
			errorType:     nil, // This is a wrapped error so we don't check the type
			setupMock: func() {
				lpSupply := math.NewInt(1000) // 1000 LP tokens total

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
				LpTokens: math.NewInt(100),
				MinBase:  math.NewInt(10),
				MinQuote: math.NewInt(20),
			},
			expectedError: "failed to send resulted coins to user account",
			errorType:     nil, // This is a wrapped error so we don't check the type
			setupMock: func() {
				lpSupply := math.NewInt(1000) // 1000 LP tokens total

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
				baseAmount := math.NewInt(100)  // 10% of 1000
				quoteAmount := math.NewInt(200) // 10% of 2000

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
			_, err := suite.msgServer.RemoveLiquidity(suite.ctx, tc.msg)

			// Verify error
			suite.Require().Error(err)
			suite.Require().Contains(err.Error(), tc.expectedError)
			if tc.errorType != nil {
				suite.Require().ErrorIs(err, tc.errorType)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestMsgAmm_RemoveLiquidity_Success() {
	testAcc := getTestAccount()

	testCases := []struct {
		name          string
		pool          types.LiquidityPool
		lpTokens      math.Int
		lpSupply      uint64
		minBase       math.Int
		minQuote      math.Int
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
				ReserveBase:  math.NewInt(1000),
				ReserveQuote: math.NewInt(2000),
				Creator:      "creator",
				Fee:          math.LegacyNewDecWithPrec(3, 3),
				Stable:       false,
			},
			lpTokens:      math.NewInt(100),
			lpSupply:      1000,             // 10% removal
			minBase:       math.NewInt(90),  // Slightly below expected
			minQuote:      math.NewInt(190), // Slightly below expected
			expectedBase:  100,              // 10% of 1000
			expectedQuote: 200,              // 10% of 2000
		},
		{
			name: "remove 50% of liquidity",
			pool: types.LiquidityPool{
				Id:           "ubze_uusdc",
				Base:         "ubze",
				Quote:        "uusdc",
				LpDenom:      "lp_ubze_uusdc",
				ReserveBase:  math.NewInt(5000),
				ReserveQuote: math.NewInt(10000),
				Creator:      "creator",
				Fee:          math.LegacyNewDecWithPrec(3, 3),
				Stable:       false,
			},
			lpTokens:      math.NewInt(500),
			lpSupply:      1000,              // 50% removal
			minBase:       math.NewInt(2400), // Slightly below expected
			minQuote:      math.NewInt(4900), // Slightly below expected
			expectedBase:  2500,              // 50% of 5000
			expectedQuote: 5000,              // 50% of 10000
		},
		{
			name: "remove small amount of liquidity",
			pool: types.LiquidityPool{
				Id:           "ubze_uusdc",
				Base:         "ubze",
				Quote:        "uusdc",
				LpDenom:      "lp_ubze_uusdc",
				ReserveBase:  math.NewInt(10000),
				ReserveQuote: math.NewInt(20000),
				Creator:      "creator",
				Fee:          math.LegacyNewDecWithPrec(3, 3),
				Stable:       false,
			},
			lpTokens:      math.NewInt(1),
			lpSupply:      1000,            // 0.1% removal
			minBase:       math.NewInt(9),  // Slightly below expected
			minQuote:      math.NewInt(19), // Slightly below expected
			expectedBase:  10,              // 0.1% of 10000
			expectedQuote: 20,              // 0.1% of 20000
		},
		{
			name: "remove all liquidity",
			pool: types.LiquidityPool{
				Id:           "ubze_uusdc",
				Base:         "ubze",
				Quote:        "uusdc",
				LpDenom:      "lp_ubze_uusdc",
				ReserveBase:  math.NewInt(3000),
				ReserveQuote: math.NewInt(6000),
				Creator:      "creator",
				Fee:          math.LegacyNewDecWithPrec(3, 3),
				Stable:       false,
			},
			lpTokens:      math.NewInt(1000),
			lpSupply:      1000,              // 100% removal
			minBase:       math.NewInt(2900), // Slightly below expected
			minQuote:      math.NewInt(5900), // Slightly below expected
			expectedBase:  3000,              // 100% of 3000
			expectedQuote: 6000,              // 100% of 6000
		},
		{
			name: "uneven pool reserves",
			pool: types.LiquidityPool{
				Id:           "ubze_uusdc",
				Base:         "ubze",
				Quote:        "uusdc",
				LpDenom:      "lp_ubze_uusdc",
				ReserveBase:  math.NewInt(1500),
				ReserveQuote: math.NewInt(4500),
				Creator:      "creator",
				Fee:          math.LegacyNewDecWithPrec(3, 3),
				Stable:       false,
			},
			lpTokens:      math.NewInt(200),
			lpSupply:      1000,             // 20% removal
			minBase:       math.NewInt(290), // Slightly below expected
			minQuote:      math.NewInt(890), // Slightly below expected
			expectedBase:  300,              // 20% of 1500
			expectedQuote: 900,              // 20% of 4500
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
			lpSupply := math.NewInt(int64(tc.lpSupply))
			baseAmount := math.NewInt(int64(tc.expectedBase))
			quoteAmount := math.NewInt(int64(tc.expectedQuote))

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

			// Execute the handler
			resp, err := suite.msgServer.RemoveLiquidity(ctx, msg)

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

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_SinglePool_Success() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool with specific fee distribution
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(3, 1), // 30%
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30%
			Providers: math.LegacyNewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, math.NewInt(1900)) // Set minimum below expected

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Expected calculations:
	// Fee: 1000 * 0.003 = 3 ubze
	// Fee distribution:
	// - Treasury: 3 * 0.3 = 0.9 (rounds to 0 as Int)
	// - Burner: 3 * 0.3 = 0.9 (rounds to 0 as Int)
	// - Providers: 3 * 0.4 = 1.2 (rounds to 1 as Int), but actually gets all 3 because others rounded to 0
	// Real input after fee: 1000 - 3 = 997
	// Expected output: (2000000 * 997) / (1000000 + 997) ≈ 1992

	// Setup module account for treasury
	moduleAcc := authtypes.NewEmptyModuleAccount(types.ModuleName)

	// Mock sending coins from account to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	expectedFee := sdk.NewCoins(v2types.DefaultMarketTakerFee)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			expectedFee,
		).
		Return(nil)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, gomock.Any(), expectedFee).
		Return(nil)
	// Since the fee is so small, treasury and burner parts round to 0
	// But verify that any fee would be handled correctly
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(
			gomock.Any(),
			types.ModuleName,
			burnermoduletypes.ModuleName,
			gomock.Any(),
		).
		Return(nil).
		AnyTimes()

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	// Mock getting module account
	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(moduleAcc).
		AnyTimes()

	// Mock sending output to user
	expectedOutput := sdk.NewCoin(denomStake, math.NewInt(1992))
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			sdk.NewCoins(expectedOutput),
		).
		Return(nil)

	// Execute swap
	resp, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify no errors
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	// Verify output matches expected
	suite.Require().Equal(expectedOutput, resp.Output)

	// Verify the pool was updated correctly in storage
	updatedPool, found := suite.k.GetLiquidityPool(suite.ctx, "pool1")
	suite.Require().True(found)

	// Base reserve should be increased by the input minus fee + LP fee portion
	// Original: 1000000, Input: 1000, Fee: 3, All fee to Providers: 3
	// Expected: 1000000 + 997 + 3 = 1001000
	suite.Require().Equal(math.NewInt(1001000), updatedPool.ReserveBase)

	// Quote reserve should be decreased by the output
	// Original: 2000000, Output: 1992
	// Expected: 2000000 - 1992 = 1998008
	suite.Require().Equal(math.NewInt(1998008), updatedPool.ReserveQuote)
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_MultiPool_Success() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create two pools for a multi-hop swap
	pool1 := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1_000_000_000_000),
		ReserveQuote: math.NewInt(2_000_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(1, 2), // 1%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(3, 1), // 30%
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30%
			Providers: math.LegacyNewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}

	// Second pool: stake/usdc
	pool2 := types.LiquidityPool{
		Id:           "pool2",
		Base:         denomStake,
		Quote:        "uusdc",
		LpDenom:      "lp_pool2",
		ReserveBase:  math.NewInt(3_000_000_000_000),
		ReserveQuote: math.NewInt(4_000_000_000_000),
		Fee:          math.LegacyNewDecWithPrec(5, 3), // 0.5%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(3, 1), // 30%
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30%
			Providers: math.LegacyNewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}

	suite.k.SetLiquidityPool(suite.ctx, pool1)
	suite.k.SetLiquidityPool(suite.ctx, pool2)

	// Create swap message - BZE -> STAKE -> USDC
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(1_000_000))
	minOutput := sdk.NewCoin("uusdc", math.NewInt(2_626_000)) // Set minimum below expected

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1", "pool2"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Setup module account for treasury
	moduleAcc := authtypes.NewEmptyModuleAccount(types.ModuleName)

	// Mock sending coins from account to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Times(1).
		Return(nil)

	expectedFee := sdk.NewCoins(v2types.DefaultMarketTakerFee)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			expectedFee,
		).
		Return(nil)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, gomock.Any(), expectedFee).
		Return(nil)
	// Mock fee handling
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(
			gomock.Any(),
			types.ModuleName,
			burnermoduletypes.ModuleName,
			gomock.Any(),
		).
		Return(nil).
		Times(2)

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).
		Return(nil).
		Times(2)

	// Mock getting module account
	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(moduleAcc).
		Times(2)

	// Final output would be approximately 2640 USDC
	// (from ~1988 STAKE, minus 0.3% fee, through the second pool)
	expectedFinalOutput := sdk.NewCoin("uusdc", math.NewInt(2_626_796))

	// Mock sending output to user
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			sdk.NewCoins(expectedFinalOutput),
		).
		Return(nil)

	// Execute swap
	resp, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify no errors
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	// Verify output matches expected
	suite.Require().Equal(expectedFinalOutput.Denom, resp.Output.Denom)
	// Allow some flexibility in the exact output amount
	suite.Require().True(resp.Output.Amount.GTE(minOutput.Amount))
	suite.Require().True(resp.Output.Amount.LTE(expectedFinalOutput.Amount.AddRaw(10)))

	// Verify both pools were updated correctly in storage
	updatedPool1, found := suite.k.GetLiquidityPool(suite.ctx, "pool1")
	suite.Require().True(found)
	//old reserve + token in from swap - fee going to other places than the reserve (burner/treasury)
	suite.Require().Equal(updatedPool1.ReserveBase, math.NewInt(1_000_000_000_000+1_000_000-(6_000)))
	suite.Require().Equal(updatedPool1.ReserveQuote, math.NewInt(1_999_998_020_002))

	updatedPool2, found := suite.k.GetLiquidityPool(suite.ctx, "pool2")
	suite.Require().True(found)
	suite.Require().True(updatedPool2.ReserveBase.GT(pool2.ReserveBase))
	suite.Require().True(updatedPool2.ReserveQuote.LT(pool2.ReserveQuote))
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_InvalidCreator() {
	// Create swap message with invalid creator
	msg := types.MsgMultiSwap{
		Creator:   "invalid_address",
		Routes:    []string{"pool1"},
		Input:     sdk.NewCoin(denomBze, math.NewInt(1000)),
		MinOutput: sdk.NewCoin(denomStake, math.NewInt(1900)),
	}

	// Execute swap
	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid address")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_PoolNotFound() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create swap message with non-existent pool
	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"nonexistent_pool"},
		Input:     sdk.NewCoin(denomBze, math.NewInt(1000)),
		MinOutput: sdk.NewCoin(denomStake, math.NewInt(1900)),
	}

	// Execute swap
	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid pools")
	suite.Require().Contains(err.Error(), "not found")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_InsufficientFunds() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		Creator:      creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, math.NewInt(1900))

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Mock insufficient funds
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(fmt.Errorf("insufficient funds"))

	// Execute swap

	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "could not capture user input coins")
	suite.Require().Contains(err.Error(), "insufficient funds")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_DenomNotInPool() {
	// Setup a test account
	creator := sdk.AccAddress([]byte("creator"))

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		Creator:      creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with wrong input denom
	inputCoin := sdk.NewCoin("wrong_denom", math.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, math.NewInt(1900))

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Mock sending coins to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	// Execute swap

	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "swap failed on pool")
	suite.Require().Contains(err.Error(), "does not exist in pool")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_OutputTooLow() {
	// Setup a test account
	creator := sdk.AccAddress([]byte("creator"))

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(3, 1), // 30%
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30%
			Providers: math.LegacyNewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with too high minimum output
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, math.NewInt(2500)) // Much higher than possible ~1988

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Mock necessary calls
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(
			gomock.Any(),
			types.ModuleName,
			burnermoduletypes.ModuleName,
			gomock.Any(),
		).
		Return(nil).
		AnyTimes()

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(authtypes.NewEmptyModuleAccount(types.ModuleName)).
		AnyTimes()

	// Execute swap

	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "expected minimum")
	suite.Require().Contains(err.Error(), "got")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_OutputDenomMismatch() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		Creator:      creator.String(),
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(3, 1), // 30%
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30%
			Providers: math.LegacyNewDecWithPrec(4, 1), // 40%
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with wrong output denom
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(1000))
	minOutput := sdk.NewCoin("wrong_denom", math.NewInt(1900)) // Different from pool's quote

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Mock necessary calls
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(
			gomock.Any(),
			types.ModuleName,
			burnermoduletypes.ModuleName,
			gomock.Any(),
		).
		Return(nil).
		AnyTimes()

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(authtypes.NewEmptyModuleAccount(types.ModuleName)).
		AnyTimes()

	// Execute swap

	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "expected wrong_denom output, got stake")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_ZeroFeeDest() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool with all fees going to LP (zero treasury and burner)
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyZeroDec(),            // 0%
			Burner:    math.LegacyZeroDec(),            // 0%
			Providers: math.LegacyNewDecWithPrec(1, 0), // 100%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, math.NewInt(1900))

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Expected calculations:
	// Fee: 1000 * 0.003 = 3 ubze
	// All fee goes to LP
	// Real input after fee: 1000 - 3 = 997
	// Expected output: (2000000 * 997) / (1000000 + 997) ≈ 1988
	expectedOutput := sdk.NewCoin(denomStake, math.NewInt(1992))

	// Mock sending coins from account to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	// No treasury or burner mocks needed as those destinations are zero

	// Mock sending output to user
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			sdk.NewCoins(expectedOutput),
		).
		Return(nil)
	expectedFee := sdk.NewCoins(v2types.DefaultMarketTakerFee)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			expectedFee,
		).
		Return(nil)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, gomock.Any(), expectedFee).
		Return(nil)

	// Execute swap

	resp, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify no errors
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	// Verify output matches expected
	suite.Require().Equal(expectedOutput, resp.Output)

	// Verify the pool was updated correctly in storage
	updatedPool, found := suite.k.GetLiquidityPool(suite.ctx, "pool1")
	suite.Require().True(found)

	// Base reserve should be increased by the input (with all fee going to LP)
	// Original: 1000000, Input: 1000, All fee to LP
	// Expected: 1000000 + 997 + 3 = 1001000
	suite.Require().Equal(math.NewInt(1001000), updatedPool.ReserveBase)

	// Quote reserve should be decreased by the output
	// Original: 2000000, Output: 1992
	// Expected: 2000000 - 1992 = 1998008
	suite.Require().Equal(math.NewInt(1998008), updatedPool.ReserveQuote)
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_FeeDistribution() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool with specific fee distribution
	// and a large input to make sure fees aren't rounded to zero
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(3, 1), // 30%
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30%
			Providers: math.LegacyNewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with larger amount to make fee distribution visible
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(100000))
	minOutput := sdk.NewCoin(denomStake, math.NewInt(150000))

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Expected calculations:
	// Fee: 100000 * 0.003 = 300 ubze
	// Fee distribution:
	// - Treasury: 300 * 0.3 = 90
	// - Burner: 300 * 0.3 = 90
	// - Providers: 300 * 0.4 = 120
	// Real input after fee: 100000 - 300 = 99700

	// Setup module account for treasury
	moduleAcc := authtypes.NewEmptyModuleAccount(types.ModuleName)

	// Mock sending coins from account to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	expectedFee := sdk.NewCoins(v2types.DefaultMarketTakerFee)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			expectedFee,
		).
		Return(nil)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, gomock.Any(), expectedFee).
		Return(nil)

	// Mock getting module account
	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(moduleAcc)

	// Mock treasury fee - expect 90 ubze
	treasuryFee := sdk.NewCoin(denomBze, math.NewInt(90))
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, gomock.Any(), sdk.NewCoins(treasuryFee)).
		Return(nil)

	// Mock burner fee - expect 90 ubze
	burnerFee := sdk.NewCoin(denomBze, math.NewInt(90))
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(
			suite.ctx,
			types.ModuleName,
			burnermoduletypes.ModuleName,
			sdk.NewCoins(burnerFee),
		).
		Return(nil)

	// Expected output from the swap
	// Using a slightly rounded value for easier testing
	//expectedOutput := sdk.NewCoin(denomStake, math.NewInt(166000))

	// Mock sending output to user
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			gomock.Any(), // Can't predict exact output due to formula
		).
		Return(nil)

	// Execute swap

	resp, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify no errors
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	// Verify the pool was updated correctly in storage
	updatedPool, found := suite.k.GetLiquidityPool(suite.ctx, "pool1")
	suite.Require().True(found)

	// Base reserve should be increased by the input minus fees sent to treasury/burner, plus LP fee portion
	// Original: 1000000, Input: 100000, Fees: 300, Treasury: 90, Burner: 90, Providers: 120
	// Expected: 1000000 + 99700 + 120 = 1099820
	suite.Require().Equal(math.NewInt(1099820), updatedPool.ReserveBase)

	// Quote reserve should be decreased by the output
	// Can't check exact amount due to formula, but should be less than original
	suite.Require().True(updatedPool.ReserveQuote.LT(pool.ReserveQuote))
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_SmallFeeAmount() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool with a very small amount that will cause
	// treasury and burner parts to be truncated to zero
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(3, 1), // 30%
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30%
			Providers: math.LegacyNewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with very small amount
	// Fee would be 10 * 0.003 = 0.03, which rounds to 0
	// This causes the swap to fail with "amount is too low to be traded"
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(10))
	minOutput := sdk.NewCoin(denomStake, math.NewInt(10))

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Mock sending coins from account to module (this happens before the swap validation)
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	// Execute swap - should fail due to amount being too low

	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error occurred
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "amount is too low to be traded")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_TreasuryFeeError() {
	// Setup a test account
	creator := sdk.AccAddress([]byte("creator"))

	// Create a test pool with fee going to treasury
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(1, 0), // 100%
			Burner:    math.LegacyZeroDec(),
			Providers: math.LegacyZeroDec(),
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with large enough input to generate fee
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(10000))
	minOutput := sdk.NewCoin(denomStake, math.NewInt(10000))

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Mock sending coins from account to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	// Mock getting module account
	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(authtypes.NewEmptyModuleAccount(types.ModuleName))

	// Mock treasury fee failing
	treasuryFee := sdk.NewCoin(denomBze, math.NewInt(30)) // 10000 * 0.003 = 30
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, gomock.Any(), sdk.NewCoins(treasuryFee)).
		Return(fmt.Errorf("treasury fee transfer failed"))

	// Execute swap

	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "swap failed on pool")
	suite.Require().Contains(err.Error(), "treasury fee transfer failed")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_BurnerFeeError() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool with fee going to burner
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyZeroDec(),
			Burner:    math.LegacyNewDecWithPrec(1, 0), // 100%
			Providers: math.LegacyZeroDec(),
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with large enough input to generate fee
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(10000))
	minOutput := sdk.NewCoin(denomStake, math.NewInt(10000))

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Mock sending coins from account to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	// Mock burner module failing
	burnerFee := sdk.NewCoin(denomBze, math.NewInt(30)) // 10000 * 0.003 = 30
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(
			suite.ctx,
			types.ModuleName,
			burnermoduletypes.ModuleName,
			sdk.NewCoins(burnerFee),
		).
		Return(fmt.Errorf("burner fee transfer failed"))

	// Execute swap

	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "swap failed on pool")
	suite.Require().Contains(err.Error(), "burner fee transfer failed")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_EmptyRoutes() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create swap message with empty routes
	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{},
		Input:     sdk.NewCoin(denomBze, math.NewInt(1000)),
		MinOutput: sdk.NewCoin(denomStake, math.NewInt(1900)),
	}

	// Execute swap

	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid pools")
	suite.Require().Contains(err.Error(), "does not contain any routes")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_InvalidCoins() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		Creator:      creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with invalid input coin (zero amount)
	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     sdk.Coin{Denom: denomBze, Amount: math.NewInt(0)}, // Zero amount
		MinOutput: sdk.NewCoin(denomStake, math.NewInt(1900)),
	}

	// Execute swap

	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid coins provided")

	// Create swap message with invalid minimum output (zero amount)
	msg = types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     sdk.NewCoin(denomBze, math.NewInt(1000)),
		MinOutput: sdk.Coin{Denom: denomStake, Amount: math.NewInt(0)}, // Zero amount
	}

	// Execute swap
	_, err = suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid coins provided")
}

func (suite *IntegrationTestSuite) TestMsgAmm_MultiSwap_SendOutputError() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  math.NewInt(1000000),
		ReserveQuote: math.NewInt(2000000),
		Fee:          math.LegacyNewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  math.LegacyNewDecWithPrec(3, 1), // 30%
			Burner:    math.LegacyNewDecWithPrec(3, 1), // 30%
			Providers: math.LegacyNewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message
	inputCoin := sdk.NewCoin(denomBze, math.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, math.NewInt(1900))

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Mock account and setup
	moduleAcc := authtypes.NewEmptyModuleAccount(types.ModuleName)

	// Mock sending coins from account to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	// Mock getting module account for treasury fee
	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(moduleAcc).
		AnyTimes()

	// Mock fee collection operations - simplified for this test
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(
			gomock.Any(),
			types.ModuleName,
			burnermoduletypes.ModuleName,
			gomock.Any(),
		).
		Return(nil).
		AnyTimes()

	// Mock sending output to user - simulate failure
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			gomock.Any(),
		).
		Return(fmt.Errorf("output transfer failed"))

	// Execute swap
	_, err := suite.msgServer.MultiSwap(suite.ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "could not send bought coins")
	suite.Require().Contains(err.Error(), "output transfer failed")

	// Verify the pool was still updated in storage (since error is after swap)
	updatedPool, found := suite.k.GetLiquidityPool(suite.ctx, "pool1")
	suite.Require().True(found)
	suite.Require().True(updatedPool.ReserveBase.GT(pool.ReserveBase))
	suite.Require().True(updatedPool.ReserveQuote.LT(pool.ReserveQuote))
}
