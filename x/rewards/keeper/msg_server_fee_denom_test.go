package keeper_test

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Reward creation fees are ubze params, but the trade keeper may take them in the tx fee denom and
// swap them to ubze. These tests pin that the rewards module never asks for the fee in ubze up front:
// a payer holding the budget and only the fee token (no ubze at all) can create every kind of reward,
// and a payer who can pay the fee in neither denom is refused by the fee capture itself, with an
// insufficient-funds error naming the fee and no reward record written.

// feeTokenOnlyBalance is a payer with the budget denom and the fee token, but zero ubze.
func (suite *IntegrationTestSuite) feeTokenOnlyBalance(acc sdk.AccAddress, budget sdk.Coin) {
	suite.bank.EXPECT().SpendableCoins(suite.ctx, acc).
		Return(sdk.NewCoins(budget, sdk.NewCoin("ufeetoken", math.NewInt(1_000_000_000)))).
		AnyTimes()
}

// swapRoundUp is how many ubze above the fee param a swap may hand back: tradebin rounds the
// optimal input up, so the captured amount can exceed the param by a few units.
const swapRoundUp = 3

// expectFeePaidInFeeToken wires a capture that succeeds the way tradebin does when the payer
// prefers another fee denom: it takes the fee token, swaps it and hands the module a few ubze more
// than the param. Exactly what the capture returned must reach the community pool fee collector,
// so the module keeps no ubze from the swap.
func (suite *IntegrationTestSuite) expectFeePaidInFeeToken(acc sdk.AccAddress, fee sdk.Coins) {
	captured := fee.Add(sdk.NewCoin("ubze", math.NewInt(swapRoundUp)))
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, acc, fee, types.ModuleName).
		Return(captured, nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, captured).
		Return(nil).Times(1)
}

// expectFeeUnpayable wires a capture that fails the way the bank does when neither denom covers it.
func (suite *IntegrationTestSuite) expectFeeUnpayable(acc sdk.AccAddress, fee sdk.Coins) {
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, acc, fee, types.ModuleName).
		Return(nil, errors.Wrapf(sdkerrors.ErrInsufficientFunds, "spendable balance 0ubze is smaller than %s", fee)).
		Times(1)
}

func (suite *IntegrationTestSuite) requireFeeRefusal(err error, fee sdk.Coins) {
	suite.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
	suite.Require().Contains(err.Error(), "could not pay the fee "+fee.String())
}

var (
	srCreateFee = sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(25_000)))
	trCreateFee = sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(25_000)))
	drCreateFee = sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(25_000)))
)

// --- CreateStakingReward ---

func (suite *IntegrationTestSuite) setStakingRewardFee() {
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{CreateStakingRewardFee: srCreateFee[0]}))
	suite.bank.EXPECT().HasSupply(suite.ctx, "ustake").Return(true).Times(1)
	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
}

func (suite *IntegrationTestSuite) stakingRewardMsg(creator sdk.AccAddress) *types.MsgCreateStakingReward {
	return &types.MsgCreateStakingReward{
		Creator:      creator.String(),
		PrizeAmount:  "1000",
		PrizeDenom:   "ufoo",
		StakingDenom: "ustake",
		Duration:     "5",
		MinStake:     "0",
		Lock:         "1",
	}
}

func (suite *IntegrationTestSuite) TestFeeDenom_CreateStakingReward_PaysFeeWithoutUbze() {
	creator := sdk.AccAddress("fee-denom-sr-01")
	suite.setStakingRewardFee()
	suite.feeTokenOnlyBalance(creator, sdk.NewCoin("ufoo", math.NewInt(5000)))
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(5000)))).
		Return(nil).Times(1)
	suite.expectFeePaidInFeeToken(creator, srCreateFee)

	res, err := suite.msgServer.CreateStakingReward(suite.ctx, suite.stakingRewardMsg(creator))
	suite.Require().NoError(err)

	_, found := suite.k.GetStakingReward(suite.ctx, res.RewardId)
	suite.Require().True(found)
}

func (suite *IntegrationTestSuite) TestFeeDenom_CreateStakingReward_FeeUnpayable() {
	creator := sdk.AccAddress("fee-denom-sr-02")
	suite.setStakingRewardFee()
	suite.feeTokenOnlyBalance(creator, sdk.NewCoin("ufoo", math.NewInt(5000)))
	// the budget leaves first; a refused fee fails the tx and the SDK reverts that transfer with it
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(5000)))).
		Return(nil).Times(1)
	suite.expectFeeUnpayable(creator, srCreateFee)

	counter := suite.k.GetStakingRewardsCounter(suite.ctx)
	_, err := suite.msgServer.CreateStakingReward(suite.ctx, suite.stakingRewardMsg(creator))
	suite.requireFeeRefusal(err, srCreateFee)
	suite.Require().Empty(suite.k.GetAllStakingReward(suite.ctx))
	suite.Require().Equal(counter, suite.k.GetStakingRewardsCounter(suite.ctx))
}

// --- CreateTradingReward ---

func (suite *IntegrationTestSuite) setTradingRewardFee() {
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{CreateTradingRewardFee: trCreateFee[0]}))
	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	suite.trade.EXPECT().MarketExists(suite.ctx, "ufoo/ubze").Return(true).Times(1)
}

func (suite *IntegrationTestSuite) tradingRewardMsg(creator sdk.AccAddress) *types.MsgCreateTradingReward {
	return &types.MsgCreateTradingReward{
		Creator:     creator.String(),
		PrizeAmount: "1000",
		PrizeDenom:  "ufoo",
		Duration:    "30",
		MarketId:    "ufoo/ubze",
		Slots:       "5",
	}
}

func (suite *IntegrationTestSuite) TestFeeDenom_CreateTradingReward_PaysFeeWithoutUbze() {
	creator := sdk.AccAddress("fee-denom-tr-01")
	suite.setTradingRewardFee()
	suite.feeTokenOnlyBalance(creator, sdk.NewCoin("ufoo", math.NewInt(5000)))
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(5000)))).
		Return(nil).Times(1)
	suite.expectFeePaidInFeeToken(creator, trCreateFee)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "hour").Return(int64(100), nil).Times(1)

	res, err := suite.msgServer.CreateTradingReward(suite.ctx, suite.tradingRewardMsg(creator))
	suite.Require().NoError(err)

	_, found := suite.k.GetPendingTradingReward(suite.ctx, res.RewardId)
	suite.Require().True(found)
}

func (suite *IntegrationTestSuite) TestFeeDenom_CreateTradingReward_FeeUnpayable() {
	creator := sdk.AccAddress("fee-denom-tr-02")
	suite.setTradingRewardFee()
	suite.feeTokenOnlyBalance(creator, sdk.NewCoin("ufoo", math.NewInt(5000)))
	// the budget leaves first; a refused fee fails the tx and the SDK reverts that transfer with it
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(5000)))).
		Return(nil).Times(1)
	suite.expectFeeUnpayable(creator, trCreateFee)

	_, err := suite.msgServer.CreateTradingReward(suite.ctx, suite.tradingRewardMsg(creator))
	suite.requireFeeRefusal(err, trCreateFee)
	suite.Require().Empty(suite.k.GetAllPendingTradingReward(suite.ctx))
}

// --- CreateDenomReward ---

func (suite *IntegrationTestSuite) TestFeeDenom_CreateDenomReward_PaysFeeWithoutUbze() {
	creator := sdk.AccAddress("fee-denom-dr-01")
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{CreateDenomRewardFee: drCreateFee[0]}))
	suite.bank.EXPECT().HasSupply(suite.ctx, "ustake").Return(true).Times(1)
	suite.feeTokenOnlyBalance(creator, sdk.NewCoin("ufoo", math.NewInt(1)))
	suite.expectFeePaidInFeeToken(creator, drCreateFee)

	_, err := suite.msgServer.CreateDenomReward(suite.ctx, types.NewMsgCreateDenomReward(creator.String(), "ustake"))
	suite.Require().NoError(err)
	suite.Require().True(suite.k.HasDenomReward(suite.ctx, "ustake"))
}

func (suite *IntegrationTestSuite) TestFeeDenom_CreateDenomReward_FeeUnpayable() {
	creator := sdk.AccAddress("fee-denom-dr-02")
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{CreateDenomRewardFee: drCreateFee[0]}))
	suite.bank.EXPECT().HasSupply(suite.ctx, "ustake").Return(true).Times(1)
	suite.expectFeeUnpayable(creator, drCreateFee)

	_, err := suite.msgServer.CreateDenomReward(suite.ctx, types.NewMsgCreateDenomReward(creator.String(), "ustake"))
	suite.requireFeeRefusal(err, drCreateFee)
	suite.Require().False(suite.k.HasDenomReward(suite.ctx, "ustake"))
}

// --- CreateDenomRewardSchedule (schedule fee + prize fee for a new prize denom) ---

func (suite *IntegrationTestSuite) TestFeeDenom_CreateDenomRewardSchedule_PaysFeesWithoutUbze() {
	creator := sdk.AccAddress("fee-denom-drs-01")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ustake", Lock: 7, StakedAmount: math.ZeroInt()})
	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	suite.feeTokenOnlyBalance(creator, sdk.NewCoin("ufoo", math.NewInt(30_000)))
	suite.expectFeePaidInFeeToken(creator, drPrizeFee)
	suite.expectFeePaidInFeeToken(creator, drScheduleFee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(30_000)))).
		Return(nil).Times(1)

	res, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ustake", "ufoo", math.NewInt(1000), "30"))
	suite.Require().NoError(err)

	_, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ustake", res.ScheduleId)
	suite.Require().True(found)
}

func (suite *IntegrationTestSuite) TestFeeDenom_CreateDenomRewardSchedule_ScheduleFeeUnpayable() {
	creator := sdk.AccAddress("fee-denom-drs-02")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ustake", Lock: 7, StakedAmount: math.ZeroInt()})
	// the prize denom already exists, so only the schedule fee is due
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ustake", PrizeDenom: "ufoo", DistributedStake: math.LegacyZeroDec()})
	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	suite.feeTokenOnlyBalance(creator, sdk.NewCoin("ufoo", math.NewInt(30_000)))
	suite.expectFeeUnpayable(creator, drScheduleFee)

	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ustake", "ufoo", math.NewInt(1000), "30"))
	suite.requireFeeRefusal(err, drScheduleFee)
	suite.Require().Empty(suite.k.GetAllDenomRewardSchedule(suite.ctx))
}

// --- prize creation fee (ensureDenomRewardPrize), reached through an airdrop in a new prize denom ---

func (suite *IntegrationTestSuite) TestFeeDenom_NewPrizeDenom_PaysPrizeFeeWithoutUbze() {
	sender := sdk.AccAddress("fee-denom-prz-01")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ustake", Lock: 7, StakedAmount: math.NewInt(4)})
	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	suite.feeTokenOnlyBalance(sender, sdk.NewCoin("ufoo", math.NewInt(1000)))
	suite.expectFeePaidInFeeToken(sender, drPrizeFee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, sender, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(1000)))).
		Return(nil).Times(1)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "day").Return(int64(1), nil).Times(1)

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ustake", "ufoo", math.NewInt(1000)))
	suite.Require().NoError(err)

	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ustake", "ufoo")
	suite.Require().True(found)
}

func (suite *IntegrationTestSuite) TestFeeDenom_NewPrizeDenom_PrizeFeeUnpayable() {
	sender := sdk.AccAddress("fee-denom-prz-02")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ustake", Lock: 7, StakedAmount: math.NewInt(4)})
	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	suite.feeTokenOnlyBalance(sender, sdk.NewCoin("ufoo", math.NewInt(1000)))
	suite.expectFeeUnpayable(sender, drPrizeFee)

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ustake", "ufoo", math.NewInt(1000)))
	suite.requireFeeRefusal(err, drPrizeFee)

	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ustake", "ufoo")
	suite.Require().False(found)
}
