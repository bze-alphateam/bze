package keeper_test

import (
	"errors"

	"cosmossdk.io/math"
	tradebinv2types "github.com/bze-alphateam/bze/x/tradebin/v2types"

	"github.com/bze-alphateam/bze/x/burner/types"
	tradebintypes "github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
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

func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_HashedLpTokenDenom() {
	msg := &types.MsgMoveIbcLockedCoins{
		Creator: sdk.AccAddress("creator").String(),
		Denom:   "ulp/0A1B2C3D4E5F",
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
	suite.expectNativePool(denom, deepReserve)
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
	suite.expectNativePool(denom, deepReserve)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		SafeModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
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
	suite.expectNativePool(denom, deepReserve)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		SafeModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
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
	suite.expectNativePool(denom, deepReserve)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		SafeModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
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
	suite.expectNativePool(denom, deepReserve)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		SafeModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
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
	suite.expectNativePool(denom, deepReserve)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		SafeModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
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
	suite.expectNativePool(denom, deepReserve)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		SafeModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
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

// When the pool price drifted, the message fails and the coin stays in the black hole, so it can be moved later.
func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_PriceDrift() {
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
	suite.expectNativePool(denom, deepReserve)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, lockedBalance).Return(true).Times(1)
	suite.trade.EXPECT().
		SafeModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(lockedBalance)).
		Return(nil, nil, tradebintypes.ErrPoolPriceDrift).
		Times(1)
	// no bank movement is expected: a call would fail the test

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, msg)

	suite.Require().Nil(res)
	suite.Require().ErrorIs(err, tradebintypes.ErrPoolPriceDrift)
}

// deepReserve is a pool reserve whose cap (3% by default) is above every locked balance of the tests above.
var deepReserve = math.NewInt(1_000_000)

// expectNativePool expects one lookup of the ubze/denom pool, which holds reserve of denom. The pool is stored with its
// assets in alphabetical order, as tradebin does.
func (suite *IntegrationTestSuite) expectNativePool(denom string, reserve math.Int) {
	suite.expectPool(denom, reserve, true)
}

func (suite *IntegrationTestSuite) expectPool(denom string, reserve math.Int, found bool) {
	base, quote := denom, "ubze"
	if base > quote {
		base, quote = quote, base
	}
	poolId := base + "_" + quote
	pool := tradebintypes.LiquidityPool{Id: poolId, Base: base, Quote: quote, ReserveBase: math.NewInt(7), ReserveQuote: math.NewInt(7)}
	if base == denom {
		pool.ReserveBase = reserve
	} else {
		pool.ReserveQuote = reserve
	}

	suite.trade.EXPECT().GetParams(gomock.Any()).Return(tradebinv2types.Params{NativeDenom: "ubze"}).Times(1)
	suite.trade.EXPECT().CreatePoolId("ubze", denom).Return(base, quote, poolId).Times(1)
	suite.trade.EXPECT().GetLiquidityPool(gomock.Any(), poolId).Return(pool, found).Times(1)
}

// expectLockedBalance expects the existence check and the black hole balance read of a move of denom.
func (suite *IntegrationTestSuite) expectLockedBalance(ctx sdk.Context, locked sdk.Coin) {
	lockAddr := sdk.AccAddress("blackhole___________")
	lockAcc := authtypes.ModuleAccount{
		BaseAccount: &authtypes.BaseAccount{Address: lockAddr.String()},
		Name:        types.BlackHoleModuleName,
	}

	suite.bank.EXPECT().HasSupply(ctx, locked.Denom).Return(true).Times(1)
	suite.acc.EXPECT().GetModuleAccount(ctx, types.BlackHoleModuleName).Return(&lockAcc).Times(1)
	suite.bank.EXPECT().GetBalance(ctx, lockAddr, locked.Denom).Return(locked).Times(1)
}

// expectSuccessfulMove expects the swap check and the zap of exactly toMove, which adds it all.
func (suite *IntegrationTestSuite) expectSuccessfulMove(ctx sdk.Context, toMove sdk.Coin) {
	suite.trade.EXPECT().CanSwapForNativeDenom(ctx, toMove).Return(true).Times(1)
	suite.trade.EXPECT().
		SafeModuleAddLiquidityWithNativeDenom(ctx, types.BlackHoleModuleName, sdk.NewCoins(toMove)).
		Return(sdk.NewCoins(toMove, sdk.NewCoin("ubze", toMove.Amount)), sdk.NewCoins(), nil).
		Times(1)
}

func moveMsg(denom string) *types.MsgMoveIbcLockedCoins {
	return &types.MsgMoveIbcLockedCoins{Creator: sdk.AccAddress("creator").String(), Denom: denom}
}

// A locked balance above the cap moves exactly floor(ratio x reserve); the rest stays in the black hole for later moves.
func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_CapsTheMove() {
	denom := "ibc/ABC123"
	ctx := suite.ctx.WithBlockHeight(100)
	// 3% of 33_333 = 999.99 -> 999
	suite.expectLockedBalance(ctx, sdk.NewInt64Coin(denom, 50_000))
	suite.expectNativePool(denom, math.NewInt(33_333))
	suite.expectSuccessfulMove(ctx, sdk.NewInt64Coin(denom, 999))

	res, err := suite.msgServer.MoveIbcLockedCoins(ctx, moveMsg(denom))
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin(denom, 999), sdk.NewInt64Coin("ubze", 999)).String(), res.Added)
}

// The cap reads the reserve of the IBC denom also when the denom sorts after the native denom (quote side).
func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_CapsTheMoveOnQuoteSide() {
	denom := "zzz/ibc"
	ctx := suite.ctx.WithBlockHeight(100)
	suite.expectLockedBalance(ctx, sdk.NewInt64Coin(denom, 50_000))
	suite.expectNativePool(denom, math.NewInt(10_000))
	suite.expectSuccessfulMove(ctx, sdk.NewInt64Coin(denom, 300))

	_, err := suite.msgServer.MoveIbcLockedCoins(ctx, moveMsg(denom))
	suite.Require().NoError(err)
}

// The cap follows the ibc_move_max_reserve_ratio param.
func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_CapFollowsParam() {
	params := types.DefaultParams()
	params.IbcMoveMaxReserveRatio = math.LegacyNewDecWithPrec(1, 1)
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	denom := "ibc/ABC123"
	ctx := suite.ctx.WithBlockHeight(100)
	suite.expectLockedBalance(ctx, sdk.NewInt64Coin(denom, 50_000))
	suite.expectNativePool(denom, math.NewInt(10_000))
	suite.expectSuccessfulMove(ctx, sdk.NewInt64Coin(denom, 1_000))

	_, err := suite.msgServer.MoveIbcLockedCoins(ctx, moveMsg(denom))
	suite.Require().NoError(err)
}

// A move without a native pool for the denom is refused before any swap check or coin transfer.
func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_NoPoolIsRefused() {
	denom := "ibc/ABC123"
	suite.expectLockedBalance(suite.ctx, sdk.NewInt64Coin(denom, 1_000))
	suite.expectPool(denom, math.ZeroInt(), false)
	// no CanSwapForNativeDenom, no zap, no bank transfer: a call would fail the test

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, moveMsg(denom))
	suite.Require().Nil(res)
	suite.Require().ErrorIs(err, types.ErrIbcMoveNoCapacity)
	_, found := suite.k.GetIbcMoveLock(suite.ctx, denom)
	suite.Require().False(found)
}

// A pool whose reserve caps the move to 0 refuses it before any swap check or coin transfer.
func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_ZeroCapIsRefused() {
	denom := "ibc/ABC123"
	suite.expectLockedBalance(suite.ctx, sdk.NewInt64Coin(denom, 1_000))
	// 3% of 33 = 0.99 -> 0
	suite.expectNativePool(denom, math.NewInt(33))

	res, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, moveMsg(denom))
	suite.Require().Nil(res)
	suite.Require().ErrorIs(err, types.ErrIbcMoveNoCapacity)
	_, found := suite.k.GetIbcMoveLock(suite.ctx, denom)
	suite.Require().False(found)
}

// A successful move writes the lock; a second move of the same denom within the cooldown is refused before any
// tradebin call, and one at the end of the cooldown goes through.
func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_Cooldown() {
	denom := "ibc/ABC123"
	first := suite.ctx.WithBlockHeight(100)
	suite.expectLockedBalance(first, sdk.NewInt64Coin(denom, 1_000))
	suite.expectNativePool(denom, deepReserve)
	suite.expectSuccessfulMove(first, sdk.NewInt64Coin(denom, 1_000))

	_, err := suite.msgServer.MoveIbcLockedCoins(first, moveMsg(denom))
	suite.Require().NoError(err)
	lock, found := suite.k.GetIbcMoveLock(first, denom)
	suite.Require().True(found)
	suite.Require().Equal(types.IbcMoveLock{Denom: denom, LastMoveHeight: 100}, lock)

	// default cooldown is 1000 blocks: height 1099 is still inside it
	within := suite.ctx.WithBlockHeight(1099)
	suite.expectLockedBalance(within, sdk.NewInt64Coin(denom, 1_000))
	// no tradebin call is expected: a call would fail the test
	res, err := suite.msgServer.MoveIbcLockedCoins(within, moveMsg(denom))
	suite.Require().Nil(res)
	suite.Require().ErrorIs(err, types.ErrIbcMoveCooldown)

	after := suite.ctx.WithBlockHeight(1100)
	suite.expectLockedBalance(after, sdk.NewInt64Coin(denom, 500))
	suite.expectNativePool(denom, deepReserve)
	suite.expectSuccessfulMove(after, sdk.NewInt64Coin(denom, 500))

	_, err = suite.msgServer.MoveIbcLockedCoins(after, moveMsg(denom))
	suite.Require().NoError(err)
	lock, _ = suite.k.GetIbcMoveLock(after, denom)
	suite.Require().Equal(int64(1100), lock.LastMoveHeight)
}

// The cooldown is per denom: a move of one denom does not block another.
func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_CooldownIsPerDenom() {
	suite.k.SetIbcMoveLock(suite.ctx, types.IbcMoveLock{Denom: "ibc/OTHER", LastMoveHeight: 100})

	denom := "ibc/ABC123"
	ctx := suite.ctx.WithBlockHeight(101)
	suite.expectLockedBalance(ctx, sdk.NewInt64Coin(denom, 1_000))
	suite.expectNativePool(denom, deepReserve)
	suite.expectSuccessfulMove(ctx, sdk.NewInt64Coin(denom, 1_000))

	_, err := suite.msgServer.MoveIbcLockedCoins(ctx, moveMsg(denom))
	suite.Require().NoError(err)
}

// A zap that fails writes no lock, so the denom can be moved again right away.
func (suite *IntegrationTestSuite) TestMsgServer_MoveIbcLockedCoins_FailedZapWritesNoLock() {
	denom := "ibc/ABC123"
	locked := sdk.NewInt64Coin(denom, 1_000)
	suite.expectLockedBalance(suite.ctx, locked)
	suite.expectNativePool(denom, deepReserve)
	suite.trade.EXPECT().CanSwapForNativeDenom(suite.ctx, locked).Return(true).Times(1)
	suite.trade.EXPECT().
		SafeModuleAddLiquidityWithNativeDenom(suite.ctx, types.BlackHoleModuleName, sdk.NewCoins(locked)).
		Return(nil, nil, tradebintypes.ErrPoolPriceDrift).
		Times(1)

	_, err := suite.msgServer.MoveIbcLockedCoins(suite.ctx, moveMsg(denom))
	suite.Require().ErrorIs(err, tradebintypes.ErrPoolPriceDrift)
	_, found := suite.k.GetIbcMoveLock(suite.ctx, denom)
	suite.Require().False(found)
}
