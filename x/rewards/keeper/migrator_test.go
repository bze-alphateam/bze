package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestMigrate4to5 checks the v5 params migration: the seven Denom Rewards params get
// their defaults, every pre-existing param keeps its stored (non-default) value, and the
// rest of the module state is untouched.
func (suite *IntegrationTestSuite) TestMigrate4to5() {
	// pre-migration params: custom non-DR values, garbage DR values (simulating
	// whatever a v4 chain would unmarshal into the new fields)
	pre := types.DefaultParams()
	pre.CreateStakingRewardFee = sdk.NewInt64Coin("ubze", 111)
	pre.CreateTradingRewardFee = sdk.NewInt64Coin("ubze", 222)
	pre.ExtraGasForExitStake = 333
	pre.CreateDenomRewardFee = sdk.NewInt64Coin("ubze", 1)
	pre.CreateDenomRewardPrizeFee = sdk.NewInt64Coin("ubze", 2)
	pre.AddDenomRewardScheduleFee = sdk.NewInt64Coin("ubze", 3)
	pre.MaxPrizeDenomsPerDr = 4
	pre.ExtraGasForDenomExit = 5
	pre.DenomRewardLock = 6
	pre.DenomRewardMinStake = 7
	suite.Require().NoError(suite.k.SetParams(suite.ctx, pre))

	// unrelated state that the migration must not touch
	sr := types.StakingReward{RewardId: "00000001", PrizeDenom: "ubze", StakingDenom: "ubze", PrizeAmount: "10", Duration: 10, StakedAmount: "0", DistributedStake: "0"}
	suite.k.SetStakingReward(suite.ctx, sr)
	dr := suite.seedDenomReward("udenom1", 1000)
	stateBefore := rewards.ExportGenesis(suite.ctx, *suite.k)

	migrator := keeper.NewMigrator(*suite.k, nil)
	suite.Require().NoError(migrator.Migrate4to5(suite.ctx))

	got := suite.k.GetParams(suite.ctx)

	// the seven DR params are reset to their defaults...
	suite.Require().Equal(types.DefaultCreateRewardFee, got.CreateDenomRewardFee)
	suite.Require().Equal(types.DefaultCreateRewardFee, got.CreateDenomRewardPrizeFee)
	suite.Require().Equal(types.DefaultAddDenomRewardScheduleFee, got.AddDenomRewardScheduleFee)
	suite.Require().Equal(types.DefaultMaxPrizeDenomsPerDr, got.MaxPrizeDenomsPerDr)
	suite.Require().Equal(types.DefaultExtraGasForDenomExit, got.ExtraGasForDenomExit)
	suite.Require().Equal(types.DefaultDenomRewardLock, got.DenomRewardLock)
	suite.Require().Equal(types.DefaultDenomRewardMinStake, got.DenomRewardMinStake)

	// ...the pre-existing params keep their stored values...
	suite.Require().Equal(pre.CreateStakingRewardFee, got.CreateStakingRewardFee)
	suite.Require().Equal(pre.CreateTradingRewardFee, got.CreateTradingRewardFee)
	suite.Require().Equal(pre.ExtraGasForExitStake, got.ExtraGasForExitStake)

	// ...and nothing else in the module state changed
	gotSr, found := suite.k.GetStakingReward(suite.ctx, sr.RewardId)
	suite.Require().True(found)
	suite.Require().Equal(sr, gotSr)
	gotDr, found := suite.k.GetDenomReward(suite.ctx, "udenom1")
	suite.Require().True(found)
	suite.Require().Equal(dr, gotDr)

	stateAfter := rewards.ExportGenesis(suite.ctx, *suite.k)
	stateBefore.Params = got // params are the only intended difference
	suite.Require().Equal(stateBefore, stateAfter)
}

// TestConsensusVersionBumpedOnce pins the rewards module consensus version introduced with
// the Denom Rewards params migration: exactly one bump from 4 to 5.
func (suite *IntegrationTestSuite) TestConsensusVersionBumpedOnce() {
	suite.Require().EqualValues(5, rewards.ConsensusVersion)
	suite.Require().EqualValues(5, rewards.AppModule{}.ConsensusVersion())
}
