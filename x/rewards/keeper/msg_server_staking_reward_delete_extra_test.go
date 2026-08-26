package keeper_test

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/bze-alphateam/bze/x/rewards/types"
)

// janitor is a valid bech32 caller for the permissionless delete message.
func (suite *IntegrationTestSuite) janitor() string {
	return sdk.AccAddress("janitor").String()
}

// lastFinishEventRewardId returns the reward_id attribute of the most recent
// StakingRewardFinishEvent, with the JSON quoting the event manager adds stripped. Empty if
// no such event was emitted.
func (suite *IntegrationTestSuite) lastFinishEventRewardId() string {
	id := ""
	for _, event := range suite.ctx.EventManager().Events() {
		if event.Type != "bze.rewards.StakingRewardFinishEvent" {
			continue
		}
		for _, attr := range event.Attributes {
			if attr.Key == "reward_id" {
				id = strings.Trim(attr.Value, "\"")
			}
		}
	}

	return id
}

// Extra DeleteStakingReward coverage: the msg-server guard branches the per-chunk suite
// (msg_server_staking_reward_delete_test.go) did not exercise. These are permissionless-message
// robustness checks — a malformed or hostile MsgDeleteStakingReward must fail cleanly, never
// panic and never remove a record it should not.

// TestDeleteStakingReward_NilRequest: a nil message is rejected with ErrInvalidRequest before
// any store access (the msg server is reachable directly, not only through the tx pipeline).
func (suite *IntegrationTestSuite) TestDeleteStakingReward_NilRequest() {
	_, err := suite.msgServer.DeleteStakingReward(suite.ctx, nil)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
}

// TestDeleteStakingReward_InvalidCreator: a non-bech32 creator is rejected by the handler's own
// AccAddressFromBech32 check (defence in depth behind ValidateBasic), and nothing is removed.
func (suite *IntegrationTestSuite) TestDeleteStakingReward_InvalidCreator() {
	suite.seedStrandedReward("bad-creator")

	_, err := suite.msgServer.DeleteStakingReward(suite.ctx, &types.MsgDeleteStakingReward{
		Creator:  "not-a-bech32-address",
		RewardId: "bad-creator",
	})
	suite.Require().Error(err)

	_, found := suite.k.GetStakingReward(suite.ctx, "bad-creator")
	suite.Require().True(found, "record must survive a rejected delete")
}

// TestDeleteStakingReward_UnparseableStakedAmount: the StakedAmount store field is a free-form
// string. A record whose StakedAmount cannot be parsed as an int is rejected with a clear error
// (no panic in math.NewIntFromString), and the record is left in place.
func (suite *IntegrationTestSuite) TestDeleteStakingReward_UnparseableStakedAmount() {
	suite.k.SetStakingReward(suite.ctx, types.StakingReward{
		RewardId:         "garbage-staked",
		PrizeAmount:      "1000",
		PrizeDenom:       "ubze",
		StakingDenom:     "ubze",
		Duration:         5,
		Payouts:          5,
		MinStake:         100,
		Lock:             0,
		StakedAmount:     "not-a-number",
		DistributedStake: "0",
	})

	_, err := suite.msgServer.DeleteStakingReward(suite.ctx, &types.MsgDeleteStakingReward{
		Creator:  suite.janitor(),
		RewardId: "garbage-staked",
	})
	suite.Require().Error(err)

	_, found := suite.k.GetStakingReward(suite.ctx, "garbage-staked")
	suite.Require().True(found)
	suite.Require().False(suite.finishEventEmitted())
}

// TestDeleteStakingReward_EmitsFinishEventWithRewardId: the positive path emits a
// StakingRewardFinishEvent that actually carries the deleted reward's id — an indexer reads
// that attribute to mark the reward closed, so the payload (not just the event type) matters.
func (suite *IntegrationTestSuite) TestDeleteStakingReward_EmitsFinishEventWithRewardId() {
	suite.seedStrandedReward("event-payload")

	_, err := suite.msgServer.DeleteStakingReward(suite.ctx, &types.MsgDeleteStakingReward{
		Creator:  suite.janitor(),
		RewardId: "event-payload",
	})
	suite.Require().NoError(err)

	suite.Require().Equal("event-payload", suite.lastFinishEventRewardId())
}
