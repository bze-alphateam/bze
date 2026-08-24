package keeper_test

import (
	"strings"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// --- shared fixtures for the money-in (schedule + airdrop) tests ---

// setDrMoneyInParams sets the two money-in fees (prize 25k, schedule 5k, both ubze) and the
// prize-denom cap. Fees are asserted through the capture-and-swap mock calls: a test that sets up
// no trade expectations proves the flow is free — an unexpected capture call panics.
func (suite *IntegrationTestSuite) setDrMoneyInParams(maxPrizes uint32) {
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
		CreateDenomRewardPrizeFee: sdk.NewCoin("ubze", math.NewInt(25_000)),
		AddDenomRewardScheduleFee: sdk.NewCoin("ubze", math.NewInt(5_000)),
		MaxPrizeDenomsPerDr:       maxPrizes,
	}))
}

// expectFeeCapture wires the exact capture-and-swap -> fee collector pair for one fee.
func (suite *IntegrationTestSuite) expectFeeCapture(acc sdk.AccAddress, fee sdk.Coins) {
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, acc, fee, types.ModuleName).
		Return(fee, nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, fee).
		Return(nil).Times(1)
}

func (suite *IntegrationTestSuite) richBalance(acc sdk.AccAddress) {
	suite.bank.EXPECT().SpendableCoins(suite.ctx, acc).
		Return(sdk.NewCoins(
			sdk.NewCoin("ubze", math.NewInt(1_000_000_000)),
			sdk.NewCoin("ufoo", math.NewInt(1_000_000_000)),
			sdk.NewCoin("uprize", math.NewInt(1_000_000_000)),
		)).AnyTimes()
}

var (
	drPrizeFee    = sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(25_000)))
	drScheduleFee = sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(5_000)))
)

// --- CreateDenomRewardSchedule ---

// Fee-matrix row 1 — first-ever schedule in a prize denom pays schedule fee + prize fee + budget:
// the prize accumulator is created at S = "0", the full budget daily x duration is escrowed, the
// schedule is stored with payouts = 0, and both create events fire.
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_NewPrizeDenom_FullFeeMatrix() {
	creator := sdk.AccAddress("drs-creator-01")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})

	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	suite.richBalance(creator)
	suite.expectFeeCapture(creator, drPrizeFee)
	suite.expectFeeCapture(creator, drScheduleFee)
	// budget escrow is exact: 1000 x 30 days
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(30_000)))).
		Return(nil).Times(1)

	msg := types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "30")
	res, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal("000000000000", res.ScheduleId)

	schedule, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", res.ScheduleId)
	suite.Require().True(found)
	suite.Require().Equal("ubze", schedule.StakingDenom)
	suite.Require().Equal("ufoo", schedule.PrizeDenom)
	suite.Require().Equal("1000", schedule.DailyAmount.String())
	suite.Require().Equal(uint32(30), schedule.Duration)
	suite.Require().Equal(uint32(0), schedule.Payouts)

	prize, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "ufoo")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("0").Equal(prize.DistributedStake))

	e, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardPrizeCreateEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "denom", "ubze")
	suite.requireEventAttr(e, "prize_denom", "ufoo")

	e, ok = suite.findTypedEvent(proto.MessageName(&types.DenomRewardScheduleCreateEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "schedule_id", "000000000000")
	suite.requireEventAttr(e, "daily_amount", "1000")
	suite.requireEventAttr(e, "duration", "30")
}

// Fee-matrix row 2 — a later schedule in a prize denom the DR already has pays only the schedule
// fee + budget: no prize fee is captured (an unexpected capture call would panic) and the existing
// accumulator's S is left untouched.
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_ExistingPrizeDenom_NoPrizeFee() {
	creator := sdk.AccAddress("drs-creator-02")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(100)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ufoo", DistributedStake: math.LegacyMustNewDecFromStr("5"), LastDistributionEpoch: 9})

	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	suite.richBalance(creator)
	suite.expectFeeCapture(creator, drScheduleFee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(7_000)))).
		Return(nil).Times(1)

	msg := types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(700), "10")
	res, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// the shared accumulator is untouched by schedule creation
	prize, _ := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "ufoo")
	suite.Require().True(math.LegacyMustNewDecFromStr("5").Equal(prize.DistributedStake))
	suite.Require().Equal(int64(9), prize.LastDistributionEpoch)
}

// The prize-denom cap rejects a schedule that would introduce accumulator #cap+1: nothing is
// stored, and no fee of any kind is captured (no trade expectations are set).
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_CapReached_Rejected() {
	creator := sdk.AccAddress("drs-creator-03")
	suite.setDrMoneyInParams(2)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uatom", DistributedStake: math.LegacyMustNewDecFromStr("0")})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ubtc", DistributedStake: math.LegacyMustNewDecFromStr("0")})

	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	suite.richBalance(creator)

	msg := types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "30")
	res, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrPrizeDenomCapReached)
	suite.Require().Nil(res)

	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "ufoo")
	suite.Require().False(found)
	suite.Require().Empty(suite.k.GetAllDenomRewardSchedule(suite.ctx))
}

// The cap check uses >= against the LIVE param: a DR left over-cap by a later param decrease
// (2 accumulators, cap now 1) still refuses new prize denoms.
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_CapGteSemantics_OverCapStillRejected() {
	creator := sdk.AccAddress("drs-creator-04")
	suite.setDrMoneyInParams(1)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uatom", DistributedStake: math.LegacyMustNewDecFromStr("0")})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ubtc", DistributedStake: math.LegacyMustNewDecFromStr("0")})

	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	suite.richBalance(creator)

	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "30"))
	suite.Require().ErrorIs(err, types.ErrPrizeDenomCapReached)
}

// Hitting the cap degrades softly: a schedule in a prize denom the DR ALREADY has is still allowed
// at (or over) the cap — only NEW prize denoms consume slots (Business Logic rule 22).
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_ExistingPrizeDenomAllowedAtCap() {
	creator := sdk.AccAddress("drs-creator-05")
	suite.setDrMoneyInParams(2)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ufoo", DistributedStake: math.LegacyMustNewDecFromStr("0")})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ubtc", DistributedStake: math.LegacyMustNewDecFromStr("0")})

	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	suite.richBalance(creator)
	suite.expectFeeCapture(creator, drScheduleFee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(50)))).
		Return(nil).Times(1)

	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(5), "10"))
	suite.Require().NoError(err)
}

// Schedules attach only to existing DRs.
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_DrNotFound() {
	creator := sdk.AccAddress("drs-creator-06")
	msg := types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "30")
	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrDenomRewardNotFound)
}

// The prize denom must have supply.
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_PrizeDenomNoSupply() {
	creator := sdk.AccAddress("drs-creator-07")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})
	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(false).Times(1)

	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "30"))
	suite.Require().ErrorIs(err, types.ErrInvalidPrizeDenom)
}

// Duration is bound to [1, HundredYearsInDays]: 0 and 36501 are rejected, the exact bound 36500 is
// accepted with the full (large) budget escrowed.
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_DurationBounds() {
	creator := sdk.AccAddress("drs-creator-08")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})

	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).AnyTimes()

	for _, bad := range []string{"0", "36501", "-4", "abc"} {
		_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), bad))
		suite.Require().ErrorIs(err, types.ErrInvalidDuration, "duration %q", bad)
	}

	suite.richBalance(creator)
	suite.expectFeeCapture(creator, drPrizeFee)
	suite.expectFeeCapture(creator, drScheduleFee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(36_500_000)))).
		Return(nil).Times(1)

	res, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "36500"))
	suite.Require().NoError(err)

	schedule, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", res.ScheduleId)
	suite.Require().True(found)
	suite.Require().Equal(uint32(36500), schedule.Duration)
}

// A daily_amount so large that daily x duration could overflow math.Int's 256-bit limit is rejected
// with an error instead of panicking (10^75 needs 250 bits; +16 bits of days > 256).
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_BudgetOverflowGuard() {
	creator := sdk.AccAddress("drs-creator-09")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})
	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)

	huge, ok := math.NewIntFromString("1" + strings.Repeat("0", 75))
	suite.Require().True(ok)
	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", huge, "36500"))
	suite.Require().ErrorIs(err, types.ErrInvalidAmount)
}

// A balance below budget + schedule fee is rejected before any capture or escrow.
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_InsufficientBalance() {
	creator := sdk.AccAddress("drs-creator-10")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})

	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(1)
	// covers the budget but not the schedule fee
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(30_000)))).Times(1)

	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "30"))
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "balance is too low")
	suite.Require().Empty(suite.k.GetAllDenomRewardSchedule(suite.ctx))
}

// Schedule ids come from the DR-private zero-filled counter and increment per creation.
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_IdsIncrement() {
	creator := sdk.AccAddress("drs-creator-11")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})

	suite.bank.EXPECT().HasSupply(suite.ctx, "ufoo").Return(true).Times(2)
	suite.richBalance(creator)
	suite.expectFeeCapture(creator, drPrizeFee)
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, creator, drScheduleFee, types.ModuleName).
		Return(drScheduleFee, nil).Times(2)
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, drScheduleFee).
		Return(nil).Times(2)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(10_000)))).
		Return(nil).Times(2)

	first, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "10"))
	suite.Require().NoError(err)
	second, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(creator.String(), "ubze", "ufoo", math.NewInt(1000), "10"))
	suite.Require().NoError(err)

	suite.Require().Equal("000000000000", first.ScheduleId)
	suite.Require().Equal("000000000001", second.ScheduleId)
}

func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Create_NilRequest() {
	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, nil)
	suite.Require().Error(err)
}

// --- UpdateDenomRewardSchedule ---

// Extension escrows exactly daily x extra_days BEFORE the duration is mutated and pays no fee
// beyond the budget (no trade expectations are set: any fee capture would panic).
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Update_ExtendsAndEscrowsExact() {
	creator := sdk.AccAddress("drs-updater-01")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId: "000000000004", StakingDenom: "ubze", PrizeDenom: "ufoo", DailyAmount: math.NewInt(1000), Duration: 30, Payouts: 3,
	})

	suite.richBalance(creator)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(12_000)))).
		Return(nil).Times(1)

	msg := types.NewMsgUpdateDenomRewardSchedule(creator.String(), "ubze", "000000000004", "12")
	res, err := suite.msgServer.UpdateDenomRewardSchedule(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	schedule, _ := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000004")
	suite.Require().Equal(uint32(42), schedule.Duration)
	suite.Require().Equal(uint32(3), schedule.Payouts)

	e, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardScheduleUpdateEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "schedule_id", "000000000004")
	suite.requireEventAttr(e, "duration", "42")
}

// The post-extension duration is still bound by HundredYearsInDays; a violation errors and leaves
// the stored schedule untouched (the whole tx, escrow included, reverts on-chain).
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Update_PostExtensionBoundRejected() {
	creator := sdk.AccAddress("drs-updater-02")
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId: "000000000000", StakingDenom: "ubze", PrizeDenom: "ufoo", DailyAmount: math.NewInt(1000), Duration: 36500, Payouts: 0,
	})

	suite.richBalance(creator)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, creator, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(1000)))).
		Return(nil).Times(1)

	_, err := suite.msgServer.UpdateDenomRewardSchedule(suite.ctx, types.NewMsgUpdateDenomRewardSchedule(creator.String(), "ubze", "000000000000", "1"))
	suite.Require().ErrorIs(err, types.ErrInvalidDuration)

	schedule, _ := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000000")
	suite.Require().Equal(uint32(36500), schedule.Duration)
}

func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Update_NotFound() {
	creator := sdk.AccAddress("drs-updater-03")
	_, err := suite.msgServer.UpdateDenomRewardSchedule(suite.ctx, types.NewMsgUpdateDenomRewardSchedule(creator.String(), "ubze", "000000000009", "10"))
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "not found")
}

func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Update_NonPositiveExtraRejected() {
	creator := sdk.AccAddress("drs-updater-04")
	for _, bad := range []string{"0", "-3", "abc"} {
		_, err := suite.msgServer.UpdateDenomRewardSchedule(suite.ctx, types.NewMsgUpdateDenomRewardSchedule(creator.String(), "ubze", "000000000000", bad))
		suite.Require().ErrorIs(err, types.ErrInvalidDuration, "duration %q", bad)
	}
}

// A balance below the extra budget is rejected before the escrow transfer, leaving the schedule as
// it was.
func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Update_InsufficientBalance() {
	creator := sdk.AccAddress("drs-updater-05")
	suite.k.SetDenomRewardSchedule(suite.ctx, types.DenomRewardSchedule{
		ScheduleId: "000000000000", StakingDenom: "ubze", PrizeDenom: "ufoo", DailyAmount: math.NewInt(1000), Duration: 30, Payouts: 0,
	})

	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ufoo", math.NewInt(11_999)))).Times(1)

	_, err := suite.msgServer.UpdateDenomRewardSchedule(suite.ctx, types.NewMsgUpdateDenomRewardSchedule(creator.String(), "ubze", "000000000000", "12"))
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "balance is too low")

	schedule, _ := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000000")
	suite.Require().Equal(uint32(30), schedule.Duration)
}

func (suite *IntegrationTestSuite) TestMsgServerDrSchedule_Update_NilRequest() {
	_, err := suite.msgServer.UpdateDenomRewardSchedule(suite.ctx, nil)
	suite.Require().Error(err)
}
