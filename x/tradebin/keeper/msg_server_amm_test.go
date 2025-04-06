package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"go.uber.org/mock/gomock"
	"testing"
)

func getFeeDestinationString(burner, treasury, providers, liquidity string) string {
	return fmt.Sprintf(
		"{\"treasury\":\"%s\",\"burner\":\"%s\",\"providers\":\"%s\",\"liquidity\":\"%s\"}",
		treasury,
		burner,
		providers,
		liquidity,
	)
}

func getTestAddress() string {
	return sdk.AccAddress("addr1_______________").String()
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
		InitialBase  uint64
		InitialQuote uint64
	}{
		{
			Name:         "zero base",
			InitialBase:  0,
			InitialQuote: 123456,
		},
		{
			Name:         "zero quote",
			InitialBase:  2123321,
			InitialQuote: 0,
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
		InitialBase:  123,
		InitialQuote: 456,
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
		InitialBase:  123,
		InitialQuote: 345,
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
		InitialBase:  123,
		InitialQuote: 345,
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
	suite.Require().EqualValues(stored.GetReserveBase(), 123)
	suite.Require().EqualValues(stored.GetReserveQuote(), 345)
}
