package keeper_test

import (
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/bze-alphateam/bze/x/rewards/types"
)

// Bech32 is case-insensitive: the all-uppercase spelling of an address decodes to the same account.
// Every position is keyed by the account's standard (lowercase) spelling, whichever spelling the
// message carried, so one account always has exactly one position per reward.

const canonicalSrId = "canonical-sr"

func (suite *IntegrationTestSuite) setCanonicalStakingReward(distributedStake string) {
	stakedAmount := "0"
	if sr, found := suite.k.GetStakingReward(suite.ctx, canonicalSrId); found {
		stakedAmount = sr.StakedAmount
	}
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         canonicalSrId,
		PrizeAmount:      "1000",
		PrizeDenom:       "uprize",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          0,
		MinStake:         0,
		Lock:             0,
		StakedAmount:     stakedAmount,
		DistributedStake: distributedStake,
	})
}

func (suite *IntegrationTestSuite) joinSr(acc sdk.AccAddress, creator string, amount int64) error {
	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", amount))
	suite.bank.EXPECT().SpendableCoins(suite.ctx, acc).Return(coins).Times(1)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(suite.ctx, acc, types.ModuleName, coins).Return(nil).Times(1)

	_, err := suite.msgServer.JoinStaking(suite.ctx, &types.MsgJoinStaking{Creator: creator, RewardId: canonicalSrId, Amount: math.NewInt(amount).String()})

	return err
}

func (suite *IntegrationTestSuite) joinDrAs(acc sdk.AccAddress, creator string, amount int64) error {
	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", amount))
	suite.bank.EXPECT().SpendableCoins(suite.ctx, acc).Return(coins).Times(1)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(suite.ctx, acc, types.ModuleName, coins).Return(nil).Times(1)

	_, err := suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(creator, "ubze", math.NewInt(amount)))

	return err
}

// An uppercase creator joins, tops up, claims and exits the same staking reward position as the
// lowercase spelling (joins alternate upper, lower, upper): one participant record keyed by the lowercase address, hooks and events
// carry the lowercase address.
func (suite *IntegrationTestSuite) TestCanonicalAddress_StakingReward_UppercaseSharesPosition() {
	acc := sdk.AccAddress("canonical-sr-user-01")
	lower := acc.String()
	upper := strings.ToUpper(lower)
	suite.Require().NotEqual(lower, upper)

	hooks := suite.registerHooks()
	suite.setCanonicalStakingReward("0")

	suite.Require().NoError(suite.joinSr(acc, upper, 100))
	suite.Require().NoError(suite.joinSr(acc, lower, 50))
	suite.Require().NoError(suite.joinSr(acc, upper, 25))

	all := suite.k.GetAllStakingRewardParticipant(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal(lower, all[0].Address)
	suite.Require().Equal("175", all[0].Amount)
	_, found := suite.k.GetStakingRewardParticipant(suite.ctx, upper, canonicalSrId)
	suite.Require().False(found)

	suite.Require().Equal([]joinRecord{{rewardId: canonicalSrId, address: lower, denom: "ubze", amount: math.NewInt(100)}}, hooks.joins)
	suite.Require().Equal([]increaseRecord{
		{rewardId: canonicalSrId, address: lower, denom: "ubze", amountAdded: math.NewInt(50), newTotal: math.NewInt(150)},
		{rewardId: canonicalSrId, address: lower, denom: "ubze", amountAdded: math.NewInt(25), newTotal: math.NewInt(175)},
	}, hooks.increases)

	e, ok := suite.findTypedEvent(proto.MessageName(&types.StakingRewardJoinEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "address", lower)

	// a distribution of 2 per staked unit: the whole 175 earns, in one claim
	suite.setCanonicalStakingReward("2")
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("uprize", 350))).
		Return(nil).Times(1)
	res, err := suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{Creator: upper, RewardId: canonicalSrId})
	suite.Require().NoError(err)
	suite.Require().Equal("350", res.Amount)

	e, ok = suite.findTypedEvent(proto.MessageName(&types.StakingRewardClaimEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "address", lower)

	// exit with the uppercase spelling: lock 0 returns the whole stake at once
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("ubze", 175))).
		Return(nil).Times(1)
	_, err = suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{Creator: upper, RewardId: canonicalSrId})
	suite.Require().NoError(err)

	suite.Require().Empty(suite.k.GetAllStakingRewardParticipant(suite.ctx))
	suite.Require().Equal([]exitRecord{{rewardId: canonicalSrId, address: lower, denom: "ubze", unstaked: math.NewInt(175)}}, hooks.exits)
	sr, _ := suite.k.GetStakingReward(suite.ctx, canonicalSrId)
	suite.Require().Equal("0", sr.StakedAmount)

	e, ok = suite.findTypedEvent(proto.MessageName(&types.StakingRewardExitEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "address", lower)
}

// Regression guard: a staking position stored under a lowercase address before this change (seeded
// directly, as on mainnet) keeps its key and amounts, and claim and exit work on it unchanged.
func (suite *IntegrationTestSuite) TestCanonicalAddress_StakingReward_ExistingLowercasePositionUnchanged() {
	acc := sdk.AccAddress("canonical-sr-user-02")
	lower := acc.String()

	hooks := suite.registerHooks()
	suite.setCanonicalStakingReward("3")
	sr, _ := suite.k.GetStakingReward(suite.ctx, canonicalSrId)
	sr.StakedAmount = "200"
	suite.k.SetStakingReward(suite.ctx, sr)
	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{Address: lower, RewardId: canonicalSrId, Amount: "200", JoinedAt: "1"})

	// top-up with the lowercase spelling: settles 200 × (3 − 1) first, then grows the position
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("uprize", 400))).
		Return(nil).Times(1)
	suite.Require().NoError(suite.joinSr(acc, lower, 50))

	p, found := suite.k.GetStakingRewardParticipant(suite.ctx, lower, canonicalSrId)
	suite.Require().True(found)
	suite.Require().Equal(types.StakingRewardParticipant{Address: lower, RewardId: canonicalSrId, Amount: "250", JoinedAt: "3"}, p)
	suite.Require().Len(suite.k.GetAllStakingRewardParticipant(suite.ctx), 1)
	suite.Require().Equal([]increaseRecord{{rewardId: canonicalSrId, address: lower, denom: "ubze", amountAdded: math.NewInt(50), newTotal: math.NewInt(250)}}, hooks.increases)

	suite.setCanonicalStakingReward("5")
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("uprize", 500))).
		Return(nil).Times(1)
	res, err := suite.msgServer.ClaimStakingRewards(suite.ctx, &types.MsgClaimStakingRewards{Creator: lower, RewardId: canonicalSrId})
	suite.Require().NoError(err)
	suite.Require().Equal("500", res.Amount)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("ubze", 250))).
		Return(nil).Times(1)
	_, err = suite.msgServer.ExitStaking(suite.ctx, &types.MsgExitStaking{Creator: lower, RewardId: canonicalSrId})
	suite.Require().NoError(err)

	suite.Require().Empty(suite.k.GetAllStakingRewardParticipant(suite.ctx))
	suite.Require().Equal([]exitRecord{{rewardId: canonicalSrId, address: lower, denom: "ubze", unstaked: math.NewInt(250)}}, hooks.exits)
}

// An uppercase creator joins, tops up, claims and exits the same denom reward position as the
// lowercase spelling.
func (suite *IntegrationTestSuite) TestCanonicalAddress_DenomReward_UppercaseSharesPosition() {
	acc := sdk.AccAddress("canonical-dr-user-01")
	lower := acc.String()
	upper := strings.ToUpper(lower)

	hooks := suite.registerDenomHooks()
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyZeroDec()})

	suite.Require().NoError(suite.joinDrAs(acc, upper, 100))
	suite.Require().NoError(suite.joinDrAs(acc, lower, 50))
	suite.Require().NoError(suite.joinDrAs(acc, upper, 25))

	all := suite.k.GetAllDenomRewardParticipant(suite.ctx)
	suite.Require().Len(all, 1)
	suite.Require().Equal(lower, all[0].Address)
	suite.Require().Equal("175", all[0].Amount.String())
	suite.Require().True(hasDrParticipantMarker(*suite.k, suite.ctx, lower, "ubze"))
	suite.Require().False(hasDrParticipantMarker(*suite.k, suite.ctx, upper, "ubze"))
	_, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, upper, "ubze", "uprize")
	suite.Require().False(found)

	suite.Require().Equal([]drJoinRecord{{denom: "ubze", address: lower, amount: math.NewInt(100)}}, hooks.joins)
	suite.Require().Equal([]drIncreaseRecord{
		{denom: "ubze", address: lower, amountAdded: math.NewInt(50), newTotal: math.NewInt(150)},
		{denom: "ubze", address: lower, amountAdded: math.NewInt(25), newTotal: math.NewInt(175)},
	}, hooks.increases)

	e, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardJoinEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "address", lower)

	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("2")})
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("uprize", 350))).
		Return(nil).Times(1)
	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: upper, Denom: "ubze"})
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin("uprize", 350)), sdk.Coins(res.Amounts))

	e, ok = suite.findTypedEvent(proto.MessageName(&types.DenomRewardClaimEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "address", lower)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("ubze", 175))).
		Return(nil).Times(1)
	_, err = suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: upper, Denom: "ubze"})
	suite.Require().NoError(err)

	suite.Require().Empty(suite.k.GetAllDenomRewardParticipant(suite.ctx))
	suite.Require().False(hasDrParticipantMarker(*suite.k, suite.ctx, lower, "ubze"))
	suite.Require().Equal([]drExitRecord{{denom: "ubze", address: lower, unstaked: math.NewInt(175)}}, hooks.exits)
	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("0", dr.StakedAmount.String())

	e, ok = suite.findTypedEvent(proto.MessageName(&types.DenomRewardExitEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "address", lower)
}

// Regression guard: a denom reward position stored under a lowercase address before this change
// keeps its key and amounts, and top-up, claim and exit work on it unchanged.
func (suite *IntegrationTestSuite) TestCanonicalAddress_DenomReward_ExistingLowercasePositionUnchanged() {
	acc := sdk.AccAddress("canonical-dr-user-02")
	lower := acc.String()

	hooks := suite.registerDenomHooks()
	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	dr.StakedAmount = math.NewInt(200)
	suite.k.SetDenomReward(suite.ctx, dr)
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("3")})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: lower, StakingDenom: "ubze", Amount: math.NewInt(200)})
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{Address: lower, StakingDenom: "ubze", PrizeDenom: "uprize", Index: math.LegacyMustNewDecFromStr("1")})

	// top-up settles 200 × (3 − 1) first
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("uprize", 400))).
		Return(nil).Times(1)
	suite.Require().NoError(suite.joinDrAs(acc, lower, 50))

	p, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", lower)
	suite.Require().True(found)
	suite.Require().Equal(lower, p.Address)
	suite.Require().Equal("250", p.Amount.String())
	suite.Require().Len(suite.k.GetAllDenomRewardParticipant(suite.ctx), 1)
	suite.Require().Equal([]drIncreaseRecord{{denom: "ubze", address: lower, amountAdded: math.NewInt(50), newTotal: math.NewInt(250)}}, hooks.increases)

	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("5")})
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("uprize", 500))).
		Return(nil).Times(1)
	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: lower, Denom: "ubze"})
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin("uprize", 500)), sdk.Coins(res.Amounts))

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, acc, sdk.NewCoins(sdk.NewInt64Coin("ubze", 250))).
		Return(nil).Times(1)
	_, err = suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: lower, Denom: "ubze"})
	suite.Require().NoError(err)

	suite.Require().Empty(suite.k.GetAllDenomRewardParticipant(suite.ctx))
	suite.Require().Equal([]drExitRecord{{denom: "ubze", address: lower, unstaked: math.NewInt(250)}}, hooks.exits)
}

// Trades reported for the uppercase and the lowercase spelling of one account add up in one
// trading reward candidate and one leaderboard entry, keyed by the lowercase address.
func (suite *IntegrationTestSuite) TestCanonicalAddress_TradingReward_UppercaseSharesCandidate() {
	tr := countdownTestReward()
	suite.activateTradingReward(tr)
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(suite.ctx, "hour").Return(int64(100), nil).Times(1)

	acc := sdk.AccAddress("canonical-trader-01")
	lower := acc.String()
	upper := strings.ToUpper(lower)

	hook := suite.k.GetOnOrderFillHook()
	hook(suite.ctx, tr.MarketId, "100", upper)
	hook(suite.ctx, tr.MarketId, "50", lower)

	candidates := suite.k.GetTradingRewardCandidateByRewardId(suite.ctx, tr.RewardId)
	suite.Require().Len(candidates, 1)
	suite.Require().Equal(lower, candidates[0].Address)
	suite.Require().Equal("150", candidates[0].Amount)

	leaderboard, found := suite.k.GetTradingRewardLeaderboard(suite.ctx, tr.RewardId)
	suite.Require().True(found)
	suite.Require().Len(leaderboard.List, 1)
	suite.Require().Equal(lower, leaderboard.List[0].Address)
	suite.Require().Equal("150", leaderboard.List[0].Amount)
}

// Participant queries find an account's positions whichever spelling of its address they are given.
func (suite *IntegrationTestSuite) TestCanonicalAddress_QueriesAcceptUppercase() {
	acc := sdk.AccAddress("canonical-query-01")
	lower := acc.String()
	upper := strings.ToUpper(lower)

	suite.k.SetStakingRewardParticipant(suite.ctx, types.StakingRewardParticipant{Address: lower, RewardId: canonicalSrId, Amount: "10", JoinedAt: "0"})
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(20)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("2")})
	suite.Require().NoError(suite.joinDrAs(acc, lower, 20))
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("3")})

	srRes, err := suite.k.StakingRewardParticipant(suite.ctx, &types.QueryStakingRewardParticipantRequest{Address: upper})
	suite.Require().NoError(err)
	suite.Require().Len(srRes.List, 1)
	suite.Require().Equal(lower, srRes.List[0].Address)

	drRes, err := suite.k.DenomRewardParticipant(suite.ctx, &types.QueryDenomRewardParticipantRequest{Address: upper, Denom: "ubze"})
	suite.Require().NoError(err)
	suite.Require().Equal(lower, drRes.Participant.Address)
	suite.Require().Equal(sdk.NewCoins(sdk.NewInt64Coin("uprize", 20)), sdk.Coins(drRes.Pending))

	listRes, err := suite.k.DenomRewardParticipations(suite.ctx, &types.QueryDenomRewardParticipationsRequest{Address: upper})
	suite.Require().NoError(err)
	suite.Require().Len(listRes.List, 1)
	suite.Require().Equal(lower, listRes.List[0].Address)
}
