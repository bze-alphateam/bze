package keeper_test

import (
	"errors"

	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_NilMessage() {
	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, nil)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "invalid message")
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_LpTokenDenom() {
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   "ulp_some_pool",
	}

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "cannot burn LP tokens")
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_DenomNoSupply() {
	denom := "ibc/ABC123"
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(false).Times(1)

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "denom does not exist")
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_NilLockAccount() {
	denom := "ibc/ABC123"
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.BlackHoleModuleName).Return(nil).Times(1)

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "could not get lock account")
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_ZeroLockedBalance() {
	denom := "ibc/ABC123"
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	lockAddr := sdk.AccAddress("blackhole___________")
	lockAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: lockAddr.String()},
		Name:        types.BlackHoleModuleName,
	}

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.BlackHoleModuleName).Return(&lockAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, lockAddr, denom).Return(sdk.NewInt64Coin(denom, 0)).Times(1)

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "no coins to move for this denom")
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_CannotSwap() {
	denom := "ibc/ABC123"
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	lockAddr := sdk.AccAddress("blackhole___________")
	lockAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: lockAddr.String()},
		Name:        types.BlackHoleModuleName,
	}

	lockedBalance := sdk.NewInt64Coin(denom, 1000)

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.BlackHoleModuleName).Return(&lockAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, lockAddr, denom).Return(lockedBalance).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(false).Times(1)

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "cannot move the locked coins due to liquidity availability")
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_AddLiquidityError() {
	denom := "ibc/ABC123"
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	lockAddr := sdk.AccAddress("blackhole___________")
	lockAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: lockAddr.String()},
		Name:        types.BlackHoleModuleName,
	}

	lockedBalance := sdk.NewInt64Coin(denom, 1000)

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.BlackHoleModuleName).Return(&lockAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, lockAddr, denom).Return(lockedBalance).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
		Return(sdk.Coins{}, sdk.Coins{}, errors.New("liquidity error")).
		Times(1)

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "failed to move locked coins to liquidity pair")
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_NoLiquidityAdded() {
	denom := "ibc/ABC123"
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	lockAddr := sdk.AccAddress("blackhole___________")
	lockAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: lockAddr.String()},
		Name:        types.BlackHoleModuleName,
	}

	lockedBalance := sdk.NewInt64Coin(denom, 1000)
	emptyCoins := sdk.NewCoins()

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.BlackHoleModuleName).Return(&lockAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, lockAddr, denom).Return(lockedBalance).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
		Return(emptyCoins, emptyCoins, nil).
		Times(1)

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "no liquidity was added")
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_SuccessNoRefund() {
	denom := "ibc/ABC123"
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	lockAddr := sdk.AccAddress("blackhole___________")
	lockAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: lockAddr.String()},
		Name:        types.BlackHoleModuleName,
	}

	lockedBalance := sdk.NewInt64Coin(denom, 1000)
	addedCoins := sdk.NewCoins(
		sdk.NewInt64Coin(denom, 500),
		sdk.NewInt64Coin("ubze", 500),
	)
	refundedCoins := sdk.NewCoins()

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.BlackHoleModuleName).Return(&lockAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, lockAddr, denom).Return(lockedBalance).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
		Return(addedCoins, refundedCoins, nil).
		Times(1)

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(addedCoins.String(), res.Added)
	suite.Require().Equal(refundedCoins.String(), res.Refunded)
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_SuccessWithNativeRefund() {
	denom := "ibc/ABC123"
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	lockAddr := sdk.AccAddress("blackhole___________")
	lockAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: lockAddr.String()},
		Name:        types.BlackHoleModuleName,
	}

	lockedBalance := sdk.NewInt64Coin(denom, 1000)
	addedCoins := sdk.NewCoins(
		sdk.NewInt64Coin(denom, 450),
		sdk.NewInt64Coin("ubze", 450),
	)
	nativeRefund := sdk.NewInt64Coin("ubze", 50)
	ibcRefund := sdk.NewInt64Coin(denom, 10)
	refundedCoins := sdk.NewCoins(ibcRefund, nativeRefund)

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.BlackHoleModuleName).Return(&lockAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, lockAddr, denom).Return(lockedBalance).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
		Return(addedCoins, refundedCoins, nil).
		Times(1)

	// The loop iterates over refunded coins. IBC denom is not native, so it's skipped.
	// Native denom is found and sent to burner module.
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, ibcRefund.Denom).Return(false).Times(1)
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, nativeRefund.Denom).Return(true).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.BlackHoleModuleName, types.ModuleName, sdk.NewCoins(nativeRefund)).
		Return(nil).
		Times(1)

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(addedCoins.String(), res.Added)
	suite.Require().Equal(refundedCoins.String(), res.Refunded)
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_SuccessRefundOnlyNonNative() {
	denom := "ibc/ABC123"
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	lockAddr := sdk.AccAddress("blackhole___________")
	lockAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: lockAddr.String()},
		Name:        types.BlackHoleModuleName,
	}

	lockedBalance := sdk.NewInt64Coin(denom, 1000)
	addedCoins := sdk.NewCoins(
		sdk.NewInt64Coin(denom, 490),
		sdk.NewInt64Coin("ubze", 490),
	)
	ibcRefund := sdk.NewInt64Coin(denom, 20)
	refundedCoins := sdk.NewCoins(ibcRefund)

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.BlackHoleModuleName).Return(&lockAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, lockAddr, denom).Return(lockedBalance).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
		Return(addedCoins, refundedCoins, nil).
		Times(1)

	// Only non-native refund, no SendCoinsFromModuleToModule expected
	suite.trade.EXPECT().IsNativeDenom(suite.ctx, ibcRefund.Denom).Return(false).Times(1)

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(addedCoins.String(), res.Added)
	suite.Require().Equal(refundedCoins.String(), res.Refunded)
}

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_SendRefundError() {
	denom := "ibc/ABC123"
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   denom,
	}

	lockAddr := sdk.AccAddress("blackhole___________")
	lockAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: lockAddr.String()},
		Name:        types.BlackHoleModuleName,
	}

	lockedBalance := sdk.NewInt64Coin(denom, 1000)
	addedCoins := sdk.NewCoins(
		sdk.NewInt64Coin(denom, 450),
		sdk.NewInt64Coin("ubze", 450),
	)
	nativeRefund := sdk.NewInt64Coin("ubze", 50)
	refundedCoins := sdk.NewCoins(nativeRefund)

	suite.bank.EXPECT().HasSupply(suite.ctx, denom).Return(true).Times(1)
	suite.acc.EXPECT().GetModuleAccount(suite.ctx, types.BlackHoleModuleName).Return(&lockAcc).Times(1)
	suite.bank.EXPECT().GetBalance(suite.ctx, lockAddr, denom).Return(lockedBalance).Times(1)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		ModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
		Return(addedCoins, refundedCoins, nil).
		Times(1)

	suite.trade.EXPECT().IsNativeDenom(suite.ctx, nativeRefund.Denom).Return(true).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.BlackHoleModuleName, types.ModuleName, sdk.NewCoins(nativeRefund)).
		Return(errors.New("send error")).
		Times(1)

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "failed to send refunded native coins to burner module")
}
