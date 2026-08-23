package keeper_test

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"
)

// --- settleDenomParticipant ---

// A participant with no index for an accumulator settles from index 0 (the lazy-zero rule): the
// full amount × S is paid and the index is stamped to S.
func (suite *IntegrationTestSuite) TestSettleDenomParticipant_AbsentIndex_LazyZero() {
	addr := sdk.AccAddress("dr-participant-01")
	dr := types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(1000)}
	suite.k.SetDenomReward(suite.ctx, dr)
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("2")})

	participant := types.DenomRewardParticipant{Address: addr.String(), StakingDenom: "ubze", Amount: math.NewInt(100)}
	suite.k.SetDenomRewardParticipant(suite.ctx, participant)

	// no index exists -> lazy zero. pending = 100 * (2 - 0) = 200
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(200)))).
		Return(nil).
		Times(1)

	paid, err := suite.k.SettleDenomParticipant(suite.ctx, dr, participant)
	suite.Require().NoError(err)
	suite.Require().Equal("200uprize", paid.String())

	// the index was stamped to S
	idx, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, addr.String(), "ubze", "uprize")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("2").Equal(idx.Index))
}

// A participant with a present index settles only the span since that index: amount × (S − index).
func (suite *IntegrationTestSuite) TestSettleDenomParticipant_PresentIndex() {
	addr := sdk.AccAddress("dr-participant-02")
	dr := types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(1000)}
	suite.k.SetDenomReward(suite.ctx, dr)
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("5")})

	participant := types.DenomRewardParticipant{Address: addr.String(), StakingDenom: "ubze", Amount: math.NewInt(10)}
	suite.k.SetDenomRewardParticipant(suite.ctx, participant)
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{
		Address: addr.String(), StakingDenom: "ubze", PrizeDenom: "uprize", Index: math.LegacyMustNewDecFromStr("2"),
	})

	// pending = 10 * (5 - 2) = 30
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(30)))).
		Return(nil).
		Times(1)

	paid, err := suite.k.SettleDenomParticipant(suite.ctx, dr, participant)
	suite.Require().NoError(err)
	suite.Require().Equal("30uprize", paid.String())

	idx, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, addr.String(), "ubze", "uprize")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("5").Equal(idx.Index))
}

// Dust — a positive pending that truncates to zero whole units — sends nothing and does NOT advance
// the index; the fraction keeps accruing and is paid cumulatively once a whole unit is reachable.
func (suite *IntegrationTestSuite) TestSettleDenomParticipant_DustGuard_NoSendNoStamp_AccruesCumulatively() {
	addr := sdk.AccAddress("dr-participant-03")
	dr := types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(1000)}
	suite.k.SetDenomReward(suite.ctx, dr)

	// S = 0.005, amount 100 -> pending = 0.5 -> truncates to 0 -> dust
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("0.005")})
	participant := types.DenomRewardParticipant{Address: addr.String(), StakingDenom: "ubze", Amount: math.NewInt(100)}
	suite.k.SetDenomRewardParticipant(suite.ctx, participant)

	// no send is set up: any SendCoins call would be an unexpected call and panic
	paid, err := suite.k.SettleDenomParticipant(suite.ctx, dr, participant)
	suite.Require().NoError(err)
	suite.Require().True(paid.IsZero())

	// the index must NOT have been created — the dust is still pending
	_, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, addr.String(), "ubze", "uprize")
	suite.Require().False(found)

	// later the accumulator grows to 0.02; because the index never advanced, the whole span from 0
	// is settled: 100 * (0.02 - 0) = 2 (the earlier 0.5 dust is included — nothing was forfeited)
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("0.02")})
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(2)))).
		Return(nil).
		Times(1)

	paid, err = suite.k.SettleDenomParticipant(suite.ctx, dr, participant)
	suite.Require().NoError(err)
	suite.Require().Equal("2uprize", paid.String())

	idx, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, addr.String(), "ubze", "uprize")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("0.02").Equal(idx.Index))
}

// Multi-accumulator settle pays each prize denom exactly once, in deterministic lexicographic
// prize-denom order regardless of insertion order, and stamps each index to its own S.
func (suite *IntegrationTestSuite) TestSettleDenomParticipant_MultiPrize_EachPaidOnce_DeterministicOrder() {
	addr := sdk.AccAddress("dr-participant-04")
	dr := types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(1000)}
	suite.k.SetDenomReward(suite.ctx, dr)

	// inserted out of lexicographic order on purpose
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uctc", DistributedStake: math.LegacyMustNewDecFromStr("3")})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uatom", DistributedStake: math.LegacyMustNewDecFromStr("1")})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ubtc", DistributedStake: math.LegacyMustNewDecFromStr("2")})

	participant := types.DenomRewardParticipant{Address: addr.String(), StakingDenom: "ubze", Amount: math.NewInt(10)}
	suite.k.SetDenomRewardParticipant(suite.ctx, participant)

	var sentOrder []string
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ sdk.AccAddress, coins sdk.Coins) error {
			sentOrder = append(sentOrder, coins[0].Denom)
			return nil
		}).
		Times(3)

	paid, err := suite.k.SettleDenomParticipant(suite.ctx, dr, participant)
	suite.Require().NoError(err)

	// uatom: 10*(1-0)=10 ; ubtc: 10*(2-0)=20 ; uctc: 10*(3-0)=30
	suite.Require().Equal("10uatom,20ubtc,30uctc", paid.String())
	// deterministic lexicographic prize-denom order
	suite.Require().Equal([]string{"uatom", "ubtc", "uctc"}, sentOrder)

	for _, tc := range []struct{ denom, s string }{{"uatom", "1"}, {"ubtc", "2"}, {"uctc", "3"}} {
		idx, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, addr.String(), "ubze", tc.denom)
		suite.Require().True(found, tc.denom)
		suite.Require().True(math.LegacyMustNewDecFromStr(tc.s).Equal(idx.Index), tc.denom)
	}
}

// Property: settling twice with no distribution in between pays zero for every prize denom the
// second time (all indexes equal S after the first settle).
func (suite *IntegrationTestSuite) TestSettleDenomParticipant_SettleTwice_SecondPaysZero() {
	addr := sdk.AccAddress("dr-participant-05")
	dr := types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(1000)}
	suite.k.SetDenomReward(suite.ctx, dr)
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uatom", DistributedStake: math.LegacyMustNewDecFromStr("1")})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ubtc", DistributedStake: math.LegacyMustNewDecFromStr("2")})

	participant := types.DenomRewardParticipant{Address: addr.String(), StakingDenom: "ubze", Amount: math.NewInt(10)}
	suite.k.SetDenomRewardParticipant(suite.ctx, participant)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("uatom", math.NewInt(10)))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("ubtc", math.NewInt(20)))).
		Return(nil).Times(1)

	paid, err := suite.k.SettleDenomParticipant(suite.ctx, dr, participant)
	suite.Require().NoError(err)
	suite.Require().Equal("10uatom,20ubtc", paid.String())

	// second settle: no distribution happened, so every prize pays zero and no send is made
	// (any SendCoins here would be an unexpected mock call and panic)
	paid, err = suite.k.SettleDenomParticipant(suite.ctx, dr, participant)
	suite.Require().NoError(err)
	suite.Require().True(paid.IsZero())
}

// A bank failure mid-settle aborts with the error; message execution being atomic, callers rely on
// the whole tx rolling back.
func (suite *IntegrationTestSuite) TestSettleDenomParticipant_BankError_Aborts() {
	addr := sdk.AccAddress("dr-participant-06")
	dr := types.DenomReward{StakingDenom: "ubze", StakedAmount: math.NewInt(1000)}
	suite.k.SetDenomReward(suite.ctx, dr)
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("2")})
	participant := types.DenomRewardParticipant{Address: addr.String(), StakingDenom: "ubze", Amount: math.NewInt(100)}
	suite.k.SetDenomRewardParticipant(suite.ctx, participant)

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, gomock.Any()).
		Return(fmt.Errorf("bank refused the transfer")).
		Times(1)

	_, err := suite.k.SettleDenomParticipant(suite.ctx, dr, participant)
	suite.Require().Error(err)
}

// --- stampParticipantIndexes ---

// Stamping covers every existing accumulator of the DR (and only that DR), writing index = S each.
func (suite *IntegrationTestSuite) TestStampParticipantIndexes_CoversEveryAccumulator() {
	addr := sdk.AccAddress("dr-participant-07")

	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uatom", DistributedStake: math.LegacyMustNewDecFromStr("1")})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ubtc", DistributedStake: math.LegacyMustNewDecFromStr("2.5")})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uctc", DistributedStake: math.LegacyMustNewDecFromStr("3")})
	// a different DR's accumulator must remain untouched for this participant
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "uother", PrizeDenom: "uatom", DistributedStake: math.LegacyMustNewDecFromStr("9")})

	suite.k.StampParticipantIndexes(suite.ctx, addr.String(), "ubze")

	for _, tc := range []struct{ denom, s string }{{"uatom", "1"}, {"ubtc", "2.5"}, {"uctc", "3"}} {
		idx, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, addr.String(), "ubze", tc.denom)
		suite.Require().True(found, tc.denom)
		suite.Require().True(math.LegacyMustNewDecFromStr(tc.s).Equal(idx.Index), tc.denom)
	}

	// isolation: the other DR's accumulator was not stamped
	_, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, addr.String(), "uother", "uatom")
	suite.Require().False(found)
}

// Stamping a DR with no accumulators writes nothing and does not panic.
func (suite *IntegrationTestSuite) TestStampParticipantIndexes_NoAccumulators_NoOp() {
	addr := sdk.AccAddress("dr-participant-08")
	suite.Require().NotPanics(func() {
		suite.k.StampParticipantIndexes(suite.ctx, addr.String(), "ubze")
	})
	suite.Require().Len(suite.k.GetAllDenomRewardParticipantIndex(suite.ctx), 0)
}

// --- distributeToDenomPrize ---

// The accumulator bump matches hand-computed S += amount/T across exact, fractional and repeating
// divisions, accumulates onto an existing S, stamps the day-epoch, and persists the prize.
func (suite *IntegrationTestSuite) TestDistributeToDenomPrize_Math() {
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "day").
		Return(int64(42), nil).
		AnyTimes()

	cases := []struct {
		name        string
		prizeDenom  string
		startS      string
		amount      int64
		stakedTotal int64
		expectedS   string
	}{
		{"exact integer division", "uint", "0", 1000, 4, "250"},
		{"exact fractional division", "ufrac", "0", 1, 8, "0.125"},
		{"repeating decimal, 18dp", "urep", "0", 1000, 3, "333.333333333333333333"},
		{"accumulates onto existing S", "uacc", "125", 500, 4, "250"},
	}

	for _, tc := range cases {
		prize := types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: tc.prizeDenom, DistributedStake: math.LegacyMustNewDecFromStr(tc.startS)}
		suite.k.SetDenomRewardPrize(suite.ctx, prize)

		err := suite.k.DistributeToDenomPrize(suite.ctx, prize, math.NewInt(tc.amount), math.NewInt(tc.stakedTotal))
		suite.Require().NoError(err, tc.name)

		got, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", tc.prizeDenom)
		suite.Require().True(found, tc.name)
		suite.Require().True(
			math.LegacyMustNewDecFromStr(tc.expectedS).Equal(got.DistributedStake),
			"%s: expected S=%s got S=%s", tc.name, tc.expectedS, got.DistributedStake,
		)
		suite.Require().Equal(int64(42), got.LastDistributionEpoch, tc.name)
	}
}

// The guards reject a zero/negative stake total (invariant I6) and a non-positive amount, and on any
// guard failure the accumulator is neither saved nor is the epoch read.
func (suite *IntegrationTestSuite) TestDistributeToDenomPrize_Guards() {
	// no epoch expectation: a guard failure must return before reading the epoch
	prize := types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("0")}

	cases := []struct {
		name        string
		amount      int64
		stakedTotal int64
	}{
		{"zero T", 1000, 0},
		{"negative T", 1000, -5},
		{"zero amount", 0, 100},
		{"negative amount", -10, 100},
	}

	for _, tc := range cases {
		err := suite.k.DistributeToDenomPrize(suite.ctx, prize, math.NewInt(tc.amount), math.NewInt(tc.stakedTotal))
		suite.Require().Error(err, tc.name)
	}

	// nothing was ever persisted
	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().False(found)
}

// distributeToDenomPrize propagates an epoch-keeper failure without persisting the bump.
func (suite *IntegrationTestSuite) TestDistributeToDenomPrize_EpochError() {
	prize := types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("0")}

	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "day").
		Return(int64(0), fmt.Errorf("epoch keeper unavailable")).
		Times(1)

	err := suite.k.DistributeToDenomPrize(suite.ctx, prize, math.NewInt(1000), math.NewInt(4))
	suite.Require().Error(err)

	// the accumulator was not persisted
	_, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().False(found)
}
