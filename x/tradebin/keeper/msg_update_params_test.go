package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	v2types "github.com/bze-alphateam/bze/x/tradebin/v2types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func validV2Params() v2types.Params {
	return v2types.Params{
		CreateMarketFee:                   sdk.NewInt64Coin(denomBze, 1000),
		MarketMakerFee:                    sdk.NewInt64Coin(denomBze, 100),
		MarketTakerFee:                    sdk.NewInt64Coin(denomBze, 200),
		MakerFeeDestination:               v2types.FeeDestinationCommunityPool,
		TakerFeeDestination:               v2types.FeeDestinationBurnerModule,
		NativeDenom:                       denomBze,
		OrderBookExtraGasWindow:           100,
		OrderBookQueueExtraGas:            25000,
		FillOrdersExtraGas:                5000,
		MinNativeLiquidityForModuleSwap:   math.NewInt(100000000000),
		OrderBookPerBlockMessages:         500,
		OrderBookQueueMessageScanExtraGas: 5000,
	}
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_ValidAuthority() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	// Mock bank supply check
	suite.bankMock.EXPECT().HasSupply(suite.ctx, denomBze).Return(true).Times(1)

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify params were updated using GetParams
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params.CreateMarketFee, retrievedParams.CreateMarketFee)
	suite.Require().Equal(params.MarketMakerFee, retrievedParams.MarketMakerFee)
	suite.Require().Equal(params.MarketTakerFee, retrievedParams.MarketTakerFee)
	suite.Require().Equal(params.MakerFeeDestination, retrievedParams.MakerFeeDestination)
	suite.Require().Equal(params.TakerFeeDestination, retrievedParams.TakerFeeDestination)
	suite.Require().Equal(params.NativeDenom, retrievedParams.NativeDenom)
	suite.Require().Equal(params.MinNativeLiquidityForModuleSwap, retrievedParams.MinNativeLiquidityForModuleSwap)
	suite.Require().Equal(params.OrderBookPerBlockMessages, retrievedParams.OrderBookPerBlockMessages)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_InvalidAuthority() {
	invalidAuthority := "bze1invalidauthority"
	params := validV2Params()

	msg := &types.MsgUpdateParams{
		Authority: invalidAuthority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_EmptyAuthority() {
	params := validV2Params()

	msg := &types.MsgUpdateParams{
		Authority: "",
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid authority")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_InvalidNativeDenom() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()
	params.NativeDenom = "invalidenom"

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	// Mock bank supply check returning false for invalid denom
	suite.bankMock.EXPECT().HasSupply(suite.ctx, "invalidenom").Return(false).Times(1)

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid native denom provided")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_EmptyNativeDenom() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()
	params.NativeDenom = ""

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "native denom cannot be an empty string")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_InvalidFeeDestination() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()
	params.MakerFeeDestination = "validator_rewards"

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid MakerFeeDestination")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_InvalidTakerFeeDestination() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()
	params.TakerFeeDestination = "burn"

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid TakerFeeDestination")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_InvalidCreateMarketFee() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()
	params.CreateMarketFee = sdk.Coin{Denom: "", Amount: math.NewInt(1000)}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "invalid CreateMarketFee")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_NegativeMarketMakerFee() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()
	params.MarketMakerFee = sdk.Coin{Denom: denomBze, Amount: math.NewInt(-100)}

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_ZeroFees() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()
	params.CreateMarketFee = sdk.NewInt64Coin(denomBze, 0)
	params.MarketMakerFee = sdk.NewInt64Coin(denomBze, 0)
	params.MarketTakerFee = sdk.NewInt64Coin(denomBze, 0)

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	// Mock bank supply check
	suite.bankMock.EXPECT().HasSupply(suite.ctx, denomBze).Return(true).Times(1)

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify zero fees were set
	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().True(retrievedParams.CreateMarketFee.IsZero())
	suite.Require().True(retrievedParams.MarketMakerFee.IsZero())
	suite.Require().True(retrievedParams.MarketTakerFee.IsZero())
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_InvalidMinNativeLiquidity() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()
	params.MinNativeLiquidityForModuleSwap = math.NewInt(0)

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "min native liquidity for module swap must be positive")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_InvalidOrderBookPerBlockMessages() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()
	params.OrderBookPerBlockMessages = 0

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(response)
	suite.Require().Contains(err.Error(), "order book per block messages must be at least 1")
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_MultipleUpdates() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// First update
	params1 := validV2Params()
	params1.CreateMarketFee = sdk.NewInt64Coin(denomBze, 500)

	msg1 := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params1,
	}

	suite.bankMock.EXPECT().HasSupply(suite.ctx, denomBze).Return(true).Times(1)

	response1, err := suite.msgServer.UpdateParams(suite.ctx, msg1)
	suite.Require().NoError(err)
	suite.Require().NotNil(response1)

	retrievedParams1 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params1.CreateMarketFee, retrievedParams1.CreateMarketFee)

	// Second update
	params2 := validV2Params()
	params2.CreateMarketFee = sdk.NewInt64Coin(denomBze, 2000)
	params2.MakerFeeDestination = v2types.FeeDestinationBurnerModule
	params2.TakerFeeDestination = v2types.FeeDestinationCommunityPool

	msg2 := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params2,
	}

	suite.bankMock.EXPECT().HasSupply(suite.ctx, denomBze).Return(true).Times(1)

	response2, err := suite.msgServer.UpdateParams(suite.ctx, msg2)
	suite.Require().NoError(err)
	suite.Require().NotNil(response2)

	retrievedParams2 := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(params2.CreateMarketFee, retrievedParams2.CreateMarketFee)
	suite.Require().Equal(params2.MakerFeeDestination, retrievedParams2.MakerFeeDestination)
	suite.Require().Equal(params2.TakerFeeDestination, retrievedParams2.TakerFeeDestination)
}

func (suite *IntegrationTestSuite) TestMsgUpdateParams_DifferentDestinations() {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	params := validV2Params()
	params.MakerFeeDestination = v2types.FeeDestinationBurnerModule
	params.TakerFeeDestination = v2types.FeeDestinationCommunityPool

	msg := &types.MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	suite.bankMock.EXPECT().HasSupply(suite.ctx, denomBze).Return(true).Times(1)

	response, err := suite.msgServer.UpdateParams(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	retrievedParams := suite.k.GetParams(suite.ctx)
	suite.Require().Equal(v2types.FeeDestinationBurnerModule, retrievedParams.MakerFeeDestination)
	suite.Require().Equal(v2types.FeeDestinationCommunityPool, retrievedParams.TakerFeeDestination)
}
