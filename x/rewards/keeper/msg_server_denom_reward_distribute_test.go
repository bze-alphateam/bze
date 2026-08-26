package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// --- DistributeDenomRewards (airdrop) ---

// Fee-matrix row 3 — an airdrop in a prize denom the DR already has is completely free: no trade
// expectations are set, so ANY fee capture would panic. The amount is escrowed and the shared
// accumulator is bumped by exactly amount/T in the same tx, stamping the distribution epoch.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_ExistingPrize_Free_BumpsAccumulator() {
	sender := sdk.AccAddress("drd-sender-01")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(4)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("0")})

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(sender)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, sender, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(1000)))).
		Return(nil).Times(1)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "day").Return(int64(42), nil).Times(1)

	msg := types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(1000))
	res, err := suite.msgServer.DistributeDenomRewards(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	prize, _ := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().True(math.LegacyMustNewDecFromStr("250.000000000000000000").Equal(prize.DistributedStake))
	suite.Require().Equal(int64(42), prize.LastDistributionEpoch)

	e, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardDistributionEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "denom", "ubze")
	suite.requireEventAttr(e, "prize_denom", "uprize")
	suite.requireEventAttr(e, "amount", "1000")
}

// Fee-matrix row 4 — an airdrop that INTRODUCES a prize denom pays the DR-prize creation fee (and
// only that): the accumulator is created at S = "0" and immediately bumped by amount/T.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_NewPrizeDenom_PaysPrizeFee() {
	sender := sdk.AccAddress("drd-sender-02")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(8)})

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(sender)
	suite.expectFeeCapture(sender, drPrizeFee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, sender, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(2)))).
		Return(nil).Times(1)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "day").Return(int64(7), nil).Times(1)

	msg := types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(2))
	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, msg)
	suite.Require().NoError(err)

	prize, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("0.250000000000000000").Equal(prize.DistributedStake))
	suite.Require().Equal(int64(7), prize.LastDistributionEpoch)

	_, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardPrizeCreateEvent{}))
	suite.Require().True(ok)
}

// Rule 19 / invariant I6: a DR with zero stakers rejects the airdrop — with T = 0 the escrow would
// be stranded with nobody to credit. Nothing is escrowed and no prize is created.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_ZeroStakers_Rejected() {
	sender := sdk.AccAddress("drd-sender-03")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})

	msg := types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(1000))
	res, err := suite.msgServer.DistributeDenomRewards(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrNoStakersInDenomReward)
	suite.Require().Nil(res)

	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().False(found)
}

// Airdrops target only existing DRs.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_DrNotFound() {
	sender := sdk.AccAddress("drd-sender-04")
	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(1000)))
	suite.Require().ErrorIs(err, types.ErrDenomRewardNotFound)
}

// The prize-denom cap applies identically on the airdrop path, with the same >= (over-cap)
// semantics: 2 accumulators with the cap lowered to 1 refuse a new prize denom, fee-free.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_CapReached_Rejected() {
	sender := sdk.AccAddress("drd-sender-05")
	suite.setDrMoneyInParams(1)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(100)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uatom", DistributedStake: math.LegacyMustNewDecFromStr("0")})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ubtc", DistributedStake: math.LegacyMustNewDecFromStr("0")})

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(sender)

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(1000)))
	suite.Require().ErrorIs(err, types.ErrPrizeDenomCapReached)

	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().False(found)
}

// A balance below the airdrop amount is rejected before the prize is ensured: no accumulator is
// created and no fee is captured.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_InsufficientBalance() {
	sender := sdk.AccAddress("drd-sender-06")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(100)})

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, sender).
		Return(sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(999)))).Times(1)

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(1000)))
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "balance is too low")

	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_NilRequest() {
	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, nil)
	suite.Require().Error(err)
}

// BZE-95: a malformed prize denom that slipped past ValidateBasic must come back as a normal
// validation error from the HasSupply guard — never reach coin construction, which would panic.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_MalformedPrizeDenom_NoPanic() {
	sender := sdk.AccAddress("drd-sender-07")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(100)})

	suite.bank.EXPECT().HasSupply(suite.ctx, "!").Return(false).Times(1)

	res, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "!", math.NewInt(1000)))
	suite.Require().ErrorIs(err, types.ErrInvalidPrizeDenom)
	suite.Require().Nil(res)

	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "!")
	suite.Require().False(found)
}

// BZE-95: parity with CreateDenomRewardSchedule — a well-formed prize denom with no supply is
// rejected before any escrow or fee, since it could never be escrowed anyway.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_PrizeDenomWithoutSupply_Rejected() {
	sender := sdk.AccAddress("drd-sender-08")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(100)})

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(false).Times(1)

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(1000)))
	suite.Require().ErrorIs(err, types.ErrInvalidPrizeDenom)

	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().False(found)
}

// The prize fee is charged exactly ONCE per denom lifetime, schedule-then-airdrop order: the
// schedule introduces the denom (25k prize fee + 5k schedule fee + budget), the later airdrop in
// the same denom is completely free — only its escrow moves.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_PrizeFeeOnce_ScheduleThenAirdrop() {
	actor := sdk.AccAddress("drd-lifecycle-01")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(10)})

	// schedule first: full fee matrix
	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(actor)
	suite.expectFeeCapture(actor, drPrizeFee)
	suite.expectFeeCapture(actor, drScheduleFee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, actor, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(5_000)))).
		Return(nil).Times(1)

	_, err := suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(actor.String(), "ubze", "uprize", math.NewInt(500), "10"))
	suite.Require().NoError(err)

	// airdrop second, same prize denom: free — only the escrow send and the accumulator bump
	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, actor, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(100)))).
		Return(nil).Times(1)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "day").Return(int64(3), nil).Times(1)

	_, err = suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(actor.String(), "ubze", "uprize", math.NewInt(100)))
	suite.Require().NoError(err)

	prize, _ := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().True(math.LegacyMustNewDecFromStr("10.000000000000000000").Equal(prize.DistributedStake))
}

// The prize fee is charged exactly ONCE per denom lifetime, airdrop-then-schedule order: the
// airdrop introduces the denom (25k prize fee), the later schedule in the same denom pays only the
// 5k schedule fee + budget.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_PrizeFeeOnce_AirdropThenSchedule() {
	actor := sdk.AccAddress("drd-lifecycle-02")
	suite.setDrMoneyInParams(50)
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(10)})

	// airdrop first: prize fee only
	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(actor)
	suite.expectFeeCapture(actor, drPrizeFee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, actor, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(100)))).
		Return(nil).Times(1)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "day").Return(int64(3), nil).Times(1)

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(actor.String(), "ubze", "uprize", math.NewInt(100)))
	suite.Require().NoError(err)

	// schedule second, same prize denom: schedule fee + budget only (no second prize fee — an
	// unexpected 25k capture would panic)
	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.expectFeeCapture(actor, drScheduleFee)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, actor, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(5_000)))).
		Return(nil).Times(1)

	_, err = suite.msgServer.CreateDenomRewardSchedule(suite.ctx, types.NewMsgCreateDenomRewardSchedule(actor.String(), "ubze", "uprize", math.NewInt(500), "10"))
	suite.Require().NoError(err)
}

// Integration with the settle engine (BZE-86): a staker's claim right after an airdrop pays their
// full share of it. Sole staker with amount = T: the airdrop bumps S by amount/T and the claim pays
// stake x (S - 0) = the whole airdrop.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_SubsequentClaimIncludesAirdrop() {
	sender := sdk.AccAddress("drd-funder-01")
	staker := sdk.AccAddress("drd-staker-01")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(400)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("0")})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: staker.String(), StakingDenom: "ubze", Amount: math.NewInt(400)})
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{Address: staker.String(), StakingDenom: "ubze", PrizeDenom: "uprize", Index: math.LegacyMustNewDecFromStr("0")})

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(sender)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, sender, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(1000)))).
		Return(nil).Times(1)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "day").Return(int64(11), nil).Times(1)

	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(1000)))
	suite.Require().NoError(err)

	// the staker claims: 400 x (2.5 - 0) = 1000 — the entire airdrop
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, staker, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(1000)))).
		Return(nil).Times(1)

	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, types.NewMsgClaimDenomRewards(staker.String(), "ubze"))
	suite.Require().NoError(err)
	suite.Require().Equal("1000uprize", sdk.Coins(res.Amounts).String())
}

// Regression for BZE-104: the airdrop accumulator bump must truncate amount/T (round down). With a
// staked total ~1e18 the ratio 2/3e18 is below one ulp (1e-18); the accumulator must stay at zero,
// not gain a rounded-up ulp. This bounds every downstream claim to floor(deposited·S)=0 ≤ the
// escrowed amount, so the shared module account is never drawn down below what was funded.
func (suite *IntegrationTestSuite) TestMsgServerDrDistribute_SubUlpRatio_TruncatesAccumulator() {
	sender := sdk.AccAddress("drd-sender-03")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(3000000000000000000)}) // 3e18
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("0")})

	suite.bank.EXPECT().HasSupply(suite.ctx, "uprize").Return(true).Times(1)
	suite.richBalance(sender)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, sender, types.ModuleName, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(2)))).
		Return(nil).Times(1)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "day").Return(int64(42), nil).Times(1)

	msg := types.NewMsgDistributeDenomRewards(sender.String(), "ubze", "uprize", math.NewInt(2))
	_, err := suite.msgServer.DistributeDenomRewards(suite.ctx, msg)
	suite.Require().NoError(err)

	prize, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().True(found)
	suite.Require().True(prize.DistributedStake.IsZero(), "accumulator must truncate the sub-ulp ratio to zero, got %s", prize.DistributedStake)
}
