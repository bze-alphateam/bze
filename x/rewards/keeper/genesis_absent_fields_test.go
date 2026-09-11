package keeper_test

import (
	"encoding/json"

	"github.com/bze-alphateam/bze/testutil/sample"
	rewards "github.com/bze-alphateam/bze/x/rewards/module"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestGenesisDenomRewards_AbsentNumericFields_ImportAsZero pins the behaviour of the customtype
// non-nullable math.Int / LegacyDec fields (staked_amount, amount, index, distributed_stake,
// daily_amount) when a genesis file omits them: the import must not panic, every later reader
// (queries, settle, the distribution pass) must see zero, and the re-export must render the
// canonical zero strings — so a hand-edited or partially generated genesis cannot brick a chain
// at InitGenesis or, worse, at the first EndBlock that touches the record.
func (suite *IntegrationTestSuite) TestGenesisDenomRewards_AbsentNumericFields_ImportAsZero() {
	addr := sample.AccAddress()
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// start from the default genesis JSON and splice in records that omit every numeric field
	var doc map[string]json.RawMessage
	suite.Require().NoError(json.Unmarshal(cdc.MustMarshalJSON(types.DefaultGenesis()), &doc))
	doc["denom_reward_list"] = json.RawMessage(`[{"staking_denom":"ubze","lock":7}]`)
	doc["denom_reward_prize_list"] = json.RawMessage(`[{"staking_denom":"ubze","prize_denom":"uprize"}]`)
	doc["denom_reward_participant_list"] = json.RawMessage(`[{"address":"` + addr + `","staking_denom":"ubze"}]`)
	doc["denom_reward_participant_index_list"] = json.RawMessage(`[{"address":"` + addr + `","staking_denom":"ubze","prize_denom":"uprize"}]`)
	doc["denom_reward_schedule_list"] = json.RawMessage(`[{"schedule_id":"000000000001","staking_denom":"ubze","prize_denom":"uprize","duration":3}]`)
	doc["denom_reward_schedule_counter"] = json.RawMessage(`"1"`)
	raw, err := json.Marshal(doc)
	suite.Require().NoError(err)

	var genState types.GenesisState
	suite.Require().NotPanics(func() { cdc.MustUnmarshalJSON(raw, &genState) })
	suite.Require().NoError(genState.Validate())

	suite.Require().NotPanics(func() { rewards.InitGenesis(suite.ctx, *suite.k, genState) })

	// readers see zero, not nil
	dr, found := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().True(found)
	suite.Require().False(dr.StakedAmount.IsNil())
	suite.Require().True(dr.StakedAmount.IsZero())
	prize, found := suite.k.GetDenomRewardPrize(suite.ctx, "ubze", "uprize")
	suite.Require().True(found)
	suite.Require().False(prize.DistributedStake.IsNil())
	suite.Require().True(prize.DistributedStake.IsZero())
	participant, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", addr)
	suite.Require().True(found)
	suite.Require().True(participant.Amount.IsZero())
	index, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, addr, "ubze", "uprize")
	suite.Require().True(found)
	suite.Require().True(index.Index.IsZero())
	schedule, found := suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	suite.Require().True(found)
	suite.Require().True(schedule.DailyAmount.IsZero())

	// the read paths and the claim path run the accumulator math on the zero values without panicking
	res, err := suite.k.DenomRewardParticipant(suite.ctx, &types.QueryDenomRewardParticipantRequest{Address: addr, Denom: "ubze"})
	suite.Require().NoError(err)
	suite.Require().Empty(res.Pending)
	_, err = suite.claimDr(sdk.MustAccAddressFromBech32(addr), "ubze")
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)

	// the daily pass meets a zero-staker DR and a zero daily amount and skips both safely
	suite.Require().NotPanics(func() {
		suite.k.EnqueueDenomRewardsDistribution(suite.ctx)
		suite.k.ProcessDenomRewardsDistributionQueue(suite.ctx)
	})
	schedule, _ = suite.k.GetDenomRewardSchedule(suite.ctx, "ubze", "000000000001")
	suite.Require().Equal(uint32(0), schedule.Payouts)

	// the re-export renders canonical zeros, so the file round-trips
	exported := rewards.ExportGenesis(suite.ctx, *suite.k)
	suite.Require().NoError(exported.Validate())
	exportedJSON := string(cdc.MustMarshalJSON(exported))
	suite.Require().Contains(exportedJSON, `"staked_amount":"0"`)
	suite.Require().Contains(exportedJSON, `"amount":"0"`)
	suite.Require().Contains(exportedJSON, `"index":"0.000000000000000000"`)
	suite.Require().Contains(exportedJSON, `"distributed_stake":"0.000000000000000000"`)
	suite.Require().Contains(exportedJSON, `"daily_amount":"0"`)
}
