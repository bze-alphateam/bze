package ante_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/ante"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"go.uber.org/mock/gomock"
)

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_FeeDisabled() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	// Set params with zero CwDeployFee (disabled)
	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		sdk.NewCoins(),
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas: 100000,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       "cosmos1sender",
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled, "next handler should be called when fee is disabled")
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_NoStoreCodeMsg() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	// Set params with active CwDeployFee
	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000)),
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Transaction with no MsgStoreCode
	tx := &mockFeeTx{
		fee: sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas: 100000,
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled, "next handler should be called when no MsgStoreCode")
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_SingleStoreCode_Stakers() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// No fee denom in context => no conversion
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_SingleStoreCode_Burner() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestBurner,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.BurnerFeeCollector, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_SingleStoreCode_CommunityPool() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestCommunityPool,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.CpFeeCollector, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_MultipleStoreCode() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	singleFee := sdk.NewInt64Coin(denomBze, 5000000000)
	deployFee := sdk.NewCoins(singleFee)

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// 3 MsgStoreCode => 3x the fee
	expectedFee := sdk.NewCoins(sdk.NewCoin(denomBze, singleFee.Amount.MulRaw(3)))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, expectedFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{Sender: feePayer.String(), WASMByteCode: []byte("code1")},
			&wasmtypes.MsgStoreCode{Sender: feePayer.String(), WASMByteCode: []byte("code2")},
			&wasmtypes.MsgStoreCode{Sender: feePayer.String(), WASMByteCode: []byte("code3")},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_InsufficientFunds() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, deployFee).
		Return(fmt.Errorf("insufficient funds")).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "insufficient funds")
	suite.Require().False(suite.nextCalled, "next should not be called on error")
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_SimulateMode() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// No bank mock expectation — fee should NOT be deducted during simulation

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, true, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled, "next handler should be called in simulate mode")
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_MixedMsgs() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestBurner,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Only 1 MsgStoreCode among mixed messages
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.BurnerFeeCollector, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{Sender: feePayer.String()},
			&wasmtypes.MsgStoreCode{Sender: feePayer.String(), WASMByteCode: []byte("wasm_code")},
			&wasmtypes.MsgInstantiateContract{Sender: feePayer.String()},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_MultiDenomFee() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(
		sdk.NewInt64Coin(denomBze, 5000000000),
		sdk.NewInt64Coin(denomUsd, 1000000),
	)

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestCommunityPool,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// 2 MsgStoreCode => 2x multi-denom fee
	expectedFee := sdk.NewCoins(
		sdk.NewInt64Coin(denomBze, 10000000000),
		sdk.NewInt64Coin(denomUsd, 2000000),
	)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.CpFeeCollector, expectedFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{Sender: feePayer.String(), WASMByteCode: []byte("code1")},
			&wasmtypes.MsgStoreCode{Sender: feePayer.String(), WASMByteCode: []byte("code2")},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_DefaultParams() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")

	// Use default params (5000 BZE to stakers)
	params := types.DefaultParams()
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	expectedFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, expectedFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_EmitsEvent() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	// Use a fresh event manager to check emitted events
	ctx := suite.ctx.WithEventManager(sdk.NewEventManager())
	_, err = decorator.AnteHandle(ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)

	events := ctx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == "cw_deploy_fee" {
			found = true
			attrs := make(map[string]string)
			for _, attr := range event.Attributes {
				attrs[attr.Key] = attr.Value
			}
			suite.Require().Equal(sdk.AccAddress(feePayer).String(), attrs["fee_payer"])
			suite.Require().Equal(deployFee.String(), attrs["fee"])
			suite.Require().Equal(types.FeeDestStakers, attrs["destination"])
			suite.Require().Equal("1", attrs["store_code_count"])
		}
	}
	suite.Require().True(found, "cw_deploy_fee event should be emitted")
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_CustomFeePayer() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	customPayer := sdk.AccAddress("custompayer_________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), customPayer, types.ModuleName, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: customPayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       customPayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

// --- Non-native denom conversion tests ---

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_NonNativeDenom_ConvertsNativePortion() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000)) // 5000 BZE

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// User pays tx fees in USD. Spot price: 1 USD = 0.5 BZE (in native units)
	// So 5000000000 ubze / 0.5 = 10000000000 usd
	spotPrice := sdk.NewDecCoinFromDec(denomUsd, sdkmath.LegacyNewDecWithPrec(5, 1)) // 0.5

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), denomUsd).
		Return(spotPrice, nil).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomBze).
		Return(true).
		Times(1)

	// Expected: native fee converted to USD
	expectedFee := sdk.NewCoins(sdk.NewInt64Coin(denomUsd, 10000000000))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, expectedFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(100000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	// Set fee denom in context (as ValidateTxFeeDenomsDecorator would)
	ctx := suite.ctx.WithValue(ante.FeeDenomKey, denomUsd)
	_, err = decorator.AnteHandle(ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_NonNativeDenom_SpotPriceFails_FallbackToNative() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Spot price lookup fails
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), denomUsd).
		Return(sdk.DecCoin{}, fmt.Errorf("price unavailable")).
		Times(1)

	// Fallback: charge the original native denom fee
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(100000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	ctx := suite.ctx.WithValue(ante.FeeDenomKey, denomUsd)
	_, err = decorator.AnteHandle(ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_NonNativeDenom_ZeroSpotPrice_FallbackToNative() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Spot price returns zero
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), denomUsd).
		Return(sdk.NewDecCoinFromDec(denomUsd, sdkmath.LegacyZeroDec()), nil).
		Times(1)

	// Fallback: charge the original native denom fee
	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(100000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	ctx := suite.ctx.WithValue(ante.FeeDenomKey, denomUsd)
	_, err = decorator.AnteHandle(ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_NativeDenomInContext_NoConversion() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Fee denom is native => no conversion
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomBze).
		Return(true).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	ctx := suite.ctx.WithValue(ante.FeeDenomKey, denomBze)
	_, err = decorator.AnteHandle(ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_NonNativeDenom_MixedDeployFee_OnlyNativeConverted() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	// Deploy fee has both native and non-native coins
	deployFee := sdk.NewCoins(
		sdk.NewInt64Coin(denomBze, 5000000000),
		sdk.NewInt64Coin(denomOther, 1000000),
	)

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// User pays in USD. Spot price: 1 USD = 2.0 BZE
	// So 5000000000 ubze / 2.0 = 2500000000 usd
	spotPrice := sdk.NewDecCoinFromDec(denomUsd, sdkmath.LegacyNewDec(2))

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), denomUsd).
		Return(spotPrice, nil).
		Times(1)

	// ubze is native, denomOther is not
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomBze).
		Return(true).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomOther).
		Return(false).
		Times(1)

	// Expected: ubze converted to usd, denomOther stays
	expectedFee := sdk.NewCoins(
		sdk.NewInt64Coin(denomOther, 1000000),
		sdk.NewInt64Coin(denomUsd, 2500000000),
	)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, expectedFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(100000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	ctx := suite.ctx.WithValue(ante.FeeDenomKey, denomUsd)
	_, err = decorator.AnteHandle(ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_NonNativeDenom_CeilingRounding() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 100)) // small amount to test rounding

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Spot price: 1 USD = 0.3 BZE => 100 / 0.3 = 333.33... => ceil = 334
	spotPrice := sdk.NewDecCoinFromDec(denomUsd, sdkmath.LegacyNewDecWithPrec(3, 1)) // 0.3

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), denomUsd).
		Return(spotPrice, nil).
		Times(1)

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomBze).
		Return(true).
		Times(1)

	// 100 / 0.3 = 333.333... => ceil => 334
	expectedFee := sdk.NewCoins(sdk.NewInt64Coin(denomUsd, 334))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, expectedFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(100000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	ctx := suite.ctx.WithValue(ante.FeeDenomKey, denomUsd)
	_, err = decorator.AnteHandle(ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

// --- Authz MsgExec tests ---

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_AuthzWrappedStoreCode() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// MsgStoreCode wrapped in authz.MsgExec should still be counted
	innerMsg := &wasmtypes.MsgStoreCode{
		Sender:       feePayer.String(),
		WASMByteCode: []byte("wasm_code"),
	}
	anyMsg, err := cdctypes.NewAnyWithValue(innerMsg)
	suite.Require().NoError(err)

	authzExec := &authz.MsgExec{
		Grantee: feePayer.String(),
		Msgs:    []*cdctypes.Any{anyMsg},
	}

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs:     []sdk.Msg{authzExec},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_AuthzMixedWithDirect() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	singleFee := sdk.NewInt64Coin(denomBze, 5000000000)
	deployFee := sdk.NewCoins(singleFee)

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// 1 direct MsgStoreCode + 1 wrapped in authz = 2x fee
	innerMsg := &wasmtypes.MsgStoreCode{
		Sender:       feePayer.String(),
		WASMByteCode: []byte("wasm_code_2"),
	}
	anyMsg, err := cdctypes.NewAnyWithValue(innerMsg)
	suite.Require().NoError(err)

	authzExec := &authz.MsgExec{
		Grantee: feePayer.String(),
		Msgs:    []*cdctypes.Any{anyMsg},
	}

	expectedFee := sdk.NewCoins(sdk.NewCoin(denomBze, singleFee.Amount.MulRaw(2)))

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, expectedFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{Sender: feePayer.String(), WASMByteCode: []byte("wasm_code_1")},
			authzExec,
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

// --- FeeGranter tests ---

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_FeeGranter_ChargesGranter() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeGranter := sdk.AccAddress("feegranter__________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Feegrant should be checked and the granter should be charged
	suite.feegrantMock.EXPECT().
		UseGrantedFees(gomock.Any(), feeGranter, feePayer, deployFee, gomock.Any()).
		Return(nil).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feeGranter, types.ModuleName, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:        sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:        100000,
		feePayer:   feePayer,
		feeGranter: feeGranter,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_FeeGranter_GrantDenied() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	feeGranter := sdk.AccAddress("feegranter__________")
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomBze, 5000000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	suite.feegrantMock.EXPECT().
		UseGrantedFees(gomock.Any(), feeGranter, feePayer, deployFee, gomock.Any()).
		Return(fmt.Errorf("fee grant not found")).
		Times(1)

	tx := &mockFeeTx{
		fee:        sdk.NewCoins(sdk.NewCoin(denomBze, sdkmath.NewInt(1000))),
		gas:        100000,
		feePayer:   feePayer,
		feeGranter: feeGranter,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	_, err = decorator.AnteHandle(suite.ctx, tx, false, suite.mockNext())
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "does not allow to pay fees for")
	suite.Require().False(suite.nextCalled)
}

func (suite *AnteTestSuite) TestCwDeployFeeDecorator_NonNativeDenom_NoNativeInFee_NoConversion() {
	decorator := ante.NewCwDeployFeeDecorator(suite.tradeMock, suite.bankMock, suite.feegrantMock, &suite.k)

	feePayer := sdk.AccAddress("feepayer____________")
	// Deploy fee is entirely in a non-native denom — no conversion needed
	deployFee := sdk.NewCoins(sdk.NewInt64Coin(denomOther, 1000000))

	params := types.NewParams(
		types.DefaultParams().ValidatorMinGasFee,
		types.DefaultMaxBalanceIterations,
		types.FeeDestStakers,
		deployFee,
	)
	err := suite.k.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomUsd).
		Return(false).
		Times(1)

	suite.tradeMock.EXPECT().
		GetDenomSpotPriceInNativeCoin(gomock.Any(), denomUsd).
		Return(sdk.NewDecCoinFromDec(denomUsd, sdkmath.LegacyNewDec(1)), nil).
		Times(1)

	// denomOther is not native — stays as-is
	suite.tradeMock.EXPECT().
		IsNativeDenom(gomock.Any(), denomOther).
		Return(false).
		Times(1)

	suite.bankMock.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), feePayer, types.ModuleName, deployFee).
		Return(nil).
		Times(1)

	tx := &mockFeeTx{
		fee:      sdk.NewCoins(sdk.NewCoin(denomUsd, sdkmath.NewInt(100000))),
		gas:      100000,
		feePayer: feePayer,
		msgs: []sdk.Msg{
			&wasmtypes.MsgStoreCode{
				Sender:       feePayer.String(),
				WASMByteCode: []byte("wasm_code"),
			},
		},
	}

	ctx := suite.ctx.WithValue(ante.FeeDenomKey, denomUsd)
	_, err = decorator.AnteHandle(ctx, tx, false, suite.mockNext())
	suite.Require().NoError(err)
	suite.Require().True(suite.nextCalled)
}
