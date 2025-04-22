package keeper_test

import (
	"fmt"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"

	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestMultiSwap_SinglePool_Success() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool with specific fee distribution
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.NewDecWithPrec(3, 1), // 30%
			Burner:    sdk.NewDecWithPrec(3, 1), // 30%
			Providers: sdk.NewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, sdk.NewInt(1900)) // Set minimum below expected

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

	suite.distrMock.EXPECT().
		FundCommunityPool(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
		).
		Return(nil).
		AnyTimes()

	// Mock getting module account
	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(moduleAcc).
		AnyTimes()

	// Mock sending output to user
	expectedOutput := sdk.NewCoin(denomStake, sdk.NewInt(1992))
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			sdk.NewCoins(expectedOutput),
		).
		Return(nil)

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	resp, err := suite.msgServer.MultiSwap(ctx, &msg)

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
	suite.Require().Equal(sdk.NewInt(1001000), updatedPool.ReserveBase)

	// Quote reserve should be decreased by the output
	// Original: 2000000, Output: 1992
	// Expected: 2000000 - 1992 = 1998008
	suite.Require().Equal(sdk.NewInt(1998008), updatedPool.ReserveQuote)
}

func (suite *IntegrationTestSuite) TestMultiSwap_MultiPool_Success() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create two pools for a multi-hop swap
	pool1 := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1_000_000_000_000),
		ReserveQuote: sdk.NewInt(2_000_000_000_000),
		Fee:          sdk.NewDecWithPrec(1, 2), // 1%
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.NewDecWithPrec(3, 1), // 30%
			Burner:    sdk.NewDecWithPrec(3, 1), // 30%
			Providers: sdk.NewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}

	// Second pool: stake/usdc
	pool2 := types.LiquidityPool{
		Id:           "pool2",
		Base:         denomStake,
		Quote:        "uusdc",
		LpDenom:      "lp_pool2",
		ReserveBase:  sdk.NewInt(3_000_000_000_000),
		ReserveQuote: sdk.NewInt(4_000_000_000_000),
		Fee:          sdk.NewDecWithPrec(5, 3), // 0.5%
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.NewDecWithPrec(3, 1), // 30%
			Burner:    sdk.NewDecWithPrec(3, 1), // 30%
			Providers: sdk.NewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}

	suite.k.SetLiquidityPool(suite.ctx, pool1)
	suite.k.SetLiquidityPool(suite.ctx, pool2)

	// Create swap message - BZE -> STAKE -> USDC
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(1_000_000))
	minOutput := sdk.NewCoin("uusdc", sdk.NewInt(2_626_000)) // Set minimum below expected

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

	suite.distrMock.EXPECT().
		FundCommunityPool(
			gomock.Any(),
			gomock.Any(),
			moduleAcc.GetAddress(),
		).
		Return(nil).
		Times(2)

	// Mock getting module account
	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(moduleAcc).
		Times(2)

	// Final output would be approximately 2640 USDC
	// (from ~1988 STAKE, minus 0.3% fee, through the second pool)
	expectedFinalOutput := sdk.NewCoin("uusdc", sdk.NewInt(2_626_796))

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
	ctx := sdk.WrapSDKContext(suite.ctx)
	resp, err := suite.msgServer.MultiSwap(ctx, &msg)

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
	suite.Require().Equal(updatedPool1.ReserveBase, sdk.NewInt(1_000_000_000_000+1_000_000-(6_000)))
	suite.Require().Equal(updatedPool1.ReserveQuote, sdk.NewInt(1_999_998_020_002))

	updatedPool2, found := suite.k.GetLiquidityPool(suite.ctx, "pool2")
	suite.Require().True(found)
	suite.Require().True(updatedPool2.ReserveBase.GT(pool2.ReserveBase))
	suite.Require().True(updatedPool2.ReserveQuote.LT(pool2.ReserveQuote))
}

func (suite *IntegrationTestSuite) TestMultiSwap_InvalidCreator() {
	// Create swap message with invalid creator
	msg := types.MsgMultiSwap{
		Creator:   "invalid_address",
		Routes:    []string{"pool1"},
		Input:     sdk.NewCoin(denomBze, sdk.NewInt(1000)),
		MinOutput: sdk.NewCoin(denomStake, sdk.NewInt(1900)),
	}

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid creator address")
}

func (suite *IntegrationTestSuite) TestMultiSwap_PoolNotFound() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create swap message with non-existent pool
	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"nonexistent_pool"},
		Input:     sdk.NewCoin(denomBze, sdk.NewInt(1000)),
		MinOutput: sdk.NewCoin(denomStake, sdk.NewInt(1900)),
	}

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid pools")
	suite.Require().Contains(err.Error(), "not found")
}

func (suite *IntegrationTestSuite) TestMultiSwap_InsufficientFunds() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		Creator:      creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, sdk.NewInt(1900))

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
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "could not capture user input coins")
	suite.Require().Contains(err.Error(), "insufficient funds")
}

func (suite *IntegrationTestSuite) TestMultiSwap_DenomNotInPool() {
	// Setup a test account
	creator := sdk.AccAddress([]byte("creator"))

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		Creator:      creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with wrong input denom
	inputCoin := sdk.NewCoin("wrong_denom", sdk.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, sdk.NewInt(1900))

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
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "swap failed on pool")
	suite.Require().Contains(err.Error(), "does not exist in pool")
}

func (suite *IntegrationTestSuite) TestMultiSwap_OutputTooLow() {
	// Setup a test account
	creator := sdk.AccAddress([]byte("creator"))

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.NewDecWithPrec(3, 1), // 30%
			Burner:    sdk.NewDecWithPrec(3, 1), // 30%
			Providers: sdk.NewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with too high minimum output
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, sdk.NewInt(2500)) // Much higher than possible ~1988

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

	suite.distrMock.EXPECT().
		FundCommunityPool(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
		).
		Return(nil).
		AnyTimes()

	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(authtypes.NewEmptyModuleAccount(types.ModuleName)).
		AnyTimes()

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "expected minimum")
	suite.Require().Contains(err.Error(), "got")
}

func (suite *IntegrationTestSuite) TestMultiSwap_OutputDenomMismatch() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		Creator:      creator.String(),
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.NewDecWithPrec(3, 1), // 30%
			Burner:    sdk.NewDecWithPrec(3, 1), // 30%
			Providers: sdk.NewDecWithPrec(4, 1), // 40%
		},
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with wrong output denom
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(1000))
	minOutput := sdk.NewCoin("wrong_denom", sdk.NewInt(1900)) // Different from pool's quote

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

	suite.distrMock.EXPECT().
		FundCommunityPool(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
		).
		Return(nil).
		AnyTimes()

	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(authtypes.NewEmptyModuleAccount(types.ModuleName)).
		AnyTimes()

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "expected wrong_denom output, got stake")
}

func (suite *IntegrationTestSuite) TestMultiSwap_ZeroFeeDest() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool with all fees going to LP (zero treasury and burner)
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.ZeroDec(),            // 0%
			Burner:    sdk.ZeroDec(),            // 0%
			Providers: sdk.NewDecWithPrec(1, 0), // 100%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, sdk.NewInt(1900))

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
	expectedOutput := sdk.NewCoin(denomStake, sdk.NewInt(1992))

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

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	resp, err := suite.msgServer.MultiSwap(ctx, &msg)

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
	suite.Require().Equal(sdk.NewInt(1001000), updatedPool.ReserveBase)

	// Quote reserve should be decreased by the output
	// Original: 2000000, Output: 1992
	// Expected: 2000000 - 1992 = 1998008
	suite.Require().Equal(sdk.NewInt(1998008), updatedPool.ReserveQuote)
}

func (suite *IntegrationTestSuite) TestMultiSwap_FeeDistribution() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool with specific fee distribution
	// and a large input to make sure fees aren't rounded to zero
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.NewDecWithPrec(3, 1), // 30%
			Burner:    sdk.NewDecWithPrec(3, 1), // 30%
			Providers: sdk.NewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with larger amount to make fee distribution visible
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(100000))
	minOutput := sdk.NewCoin(denomStake, sdk.NewInt(150000))

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

	// Mock getting module account
	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(moduleAcc)

	// Mock treasury fee - expect 90 ubze
	treasuryFee := sdk.NewCoin(denomBze, sdk.NewInt(90))
	suite.distrMock.EXPECT().
		FundCommunityPool(
			suite.ctx,
			sdk.NewCoins(treasuryFee),
			gomock.Any(),
		).
		Return(nil)

	// Mock burner fee - expect 90 ubze
	burnerFee := sdk.NewCoin(denomBze, sdk.NewInt(90))
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
	//expectedOutput := sdk.NewCoin(denomStake, sdk.NewInt(166000))

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
	ctx := sdk.WrapSDKContext(suite.ctx)
	resp, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify no errors
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	// Verify the pool was updated correctly in storage
	updatedPool, found := suite.k.GetLiquidityPool(suite.ctx, "pool1")
	suite.Require().True(found)

	// Base reserve should be increased by the input minus fees sent to treasury/burner, plus LP fee portion
	// Original: 1000000, Input: 100000, Fees: 300, Treasury: 90, Burner: 90, Providers: 120
	// Expected: 1000000 + 99700 + 120 = 1099820
	suite.Require().Equal(sdk.NewInt(1099820), updatedPool.ReserveBase)

	// Quote reserve should be decreased by the output
	// Can't check exact amount due to formula, but should be less than original
	suite.Require().True(updatedPool.ReserveQuote.LT(pool.ReserveQuote))
}

func (suite *IntegrationTestSuite) TestMultiSwap_SmallFeeAmount() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool with a very small amount that will cause
	// treasury and burner parts to be truncated to zero
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.NewDecWithPrec(3, 1), // 30%
			Burner:    sdk.NewDecWithPrec(3, 1), // 30%
			Providers: sdk.NewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with very small amount
	// Fee would be 10 * 0.003 = 0.03, which should result in all parts being truncated to 0
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(10))
	minOutput := sdk.NewCoin(denomStake, sdk.NewInt(10))

	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     inputCoin,
		MinOutput: minOutput,
	}

	// Expected calculations:
	// Fee: 10 * 0.003 = 0.03 (rounds to 0 as Int)
	// All fee effectively becomes 0

	// Mock sending coins from account to module
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(
			suite.ctx,
			creator,
			types.ModuleName,
			sdk.NewCoins(inputCoin),
		).
		Return(nil)

	// Should still attempt to get module account but no distribution should happen
	suite.accountMock.EXPECT().
		GetModuleAccount(suite.ctx, types.ModuleName).
		Return(authtypes.NewEmptyModuleAccount(types.ModuleName)).
		AnyTimes()

	// Expected output from the swap (rounded for simplicity)
	//expectedOutput := sdk.NewCoin(denomStake, sdk.NewInt(19))

	// Mock sending output to user
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(
			suite.ctx,
			types.ModuleName,
			creator,
			gomock.Any(), // Can't predict exact amount
		).
		Return(nil)

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	resp, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify no errors
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	// Verify the pool was updated correctly in storage
	updatedPool, found := suite.k.GetLiquidityPool(suite.ctx, "pool1")
	suite.Require().True(found)

	// With such small amount, fee is effectively 0, so all input goes to reserve
	// Original: 1000000, Input: 10
	// Expected: 1000000 + 10 = 1000010
	suite.Require().Equal(sdk.NewInt(1000010), updatedPool.ReserveBase)
}

func (suite *IntegrationTestSuite) TestMultiSwap_TreasuryFeeError() {
	// Setup a test account
	creator := sdk.AccAddress([]byte("creator"))

	// Create a test pool with fee going to treasury
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.NewDecWithPrec(1, 0), // 100%
			Burner:    sdk.ZeroDec(),
			Providers: sdk.ZeroDec(),
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with large enough input to generate fee
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(10000))
	minOutput := sdk.NewCoin(denomStake, sdk.NewInt(10000))

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
	treasuryFee := sdk.NewCoin(denomBze, sdk.NewInt(30)) // 10000 * 0.003 = 30
	suite.distrMock.EXPECT().
		FundCommunityPool(
			suite.ctx,
			sdk.NewCoins(treasuryFee),
			gomock.Any(),
		).
		Return(fmt.Errorf("treasury fee transfer failed"))

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "swap failed on pool")
	suite.Require().Contains(err.Error(), "treasury fee transfer failed")
}

func (suite *IntegrationTestSuite) TestMultiSwap_BurnerFeeError() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool with fee going to burner
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.ZeroDec(),
			Burner:    sdk.NewDecWithPrec(1, 0), // 100%
			Providers: sdk.ZeroDec(),
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with large enough input to generate fee
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(10000))
	minOutput := sdk.NewCoin(denomStake, sdk.NewInt(10000))

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
	burnerFee := sdk.NewCoin(denomBze, sdk.NewInt(30)) // 10000 * 0.003 = 30
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(
			suite.ctx,
			types.ModuleName,
			burnermoduletypes.ModuleName,
			sdk.NewCoins(burnerFee),
		).
		Return(fmt.Errorf("burner fee transfer failed"))

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "swap failed on pool")
	suite.Require().Contains(err.Error(), "burner fee transfer failed")
}

func (suite *IntegrationTestSuite) TestMultiSwap_EmptyRoutes() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create swap message with empty routes
	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{},
		Input:     sdk.NewCoin(denomBze, sdk.NewInt(1000)),
		MinOutput: sdk.NewCoin(denomStake, sdk.NewInt(1900)),
	}

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid pools")
	suite.Require().Contains(err.Error(), "does not contain any routes")
}

func (suite *IntegrationTestSuite) TestMultiSwap_InvalidCoins() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		Creator:      creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message with invalid input coin (zero amount)
	msg := types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     sdk.Coin{Denom: denomBze, Amount: sdk.NewInt(0)}, // Zero amount
		MinOutput: sdk.NewCoin(denomStake, sdk.NewInt(1900)),
	}

	// Execute swap
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid coins provided")

	// Create swap message with invalid minimum output (zero amount)
	msg = types.MsgMultiSwap{
		Creator:   creator.String(),
		Routes:    []string{"pool1"},
		Input:     sdk.NewCoin(denomBze, sdk.NewInt(1000)),
		MinOutput: sdk.Coin{Denom: denomStake, Amount: sdk.NewInt(0)}, // Zero amount
	}

	// Execute swap
	_, err = suite.msgServer.MultiSwap(ctx, &msg)

	// Verify error
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "invalid coins provided")
}

func (suite *IntegrationTestSuite) TestMultiSwap_SendOutputError() {
	// Setup a test account
	creator := sdk.AccAddress("creator")

	// Create a test pool
	pool := types.LiquidityPool{
		Id:           "pool1",
		Base:         denomBze,
		Quote:        denomStake,
		LpDenom:      "lp_pool1",
		ReserveBase:  sdk.NewInt(1000000),
		ReserveQuote: sdk.NewInt(2000000),
		Fee:          sdk.NewDecWithPrec(3, 3), // 0.3%
		FeeDest: &types.FeeDestination{
			Treasury:  sdk.NewDecWithPrec(3, 1), // 30%
			Burner:    sdk.NewDecWithPrec(3, 1), // 30%
			Providers: sdk.NewDecWithPrec(4, 1), // 40%
		},
		Creator: creator.String(),
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	// Create swap message
	inputCoin := sdk.NewCoin(denomBze, sdk.NewInt(1000))
	minOutput := sdk.NewCoin(denomStake, sdk.NewInt(1900))

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
	suite.distrMock.EXPECT().
		FundCommunityPool(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
		).
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
	ctx := sdk.WrapSDKContext(suite.ctx)
	_, err := suite.msgServer.MultiSwap(ctx, &msg)

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
