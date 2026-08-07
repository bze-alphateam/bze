package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
)

func newBoostParticipant(address, rewardId, boostId string) types.BoostParticipant {
	return types.BoostParticipant{
		Address:  address,
		RewardId: rewardId,
		BoostId:  boostId,
		JoinedAt: "1.5",
	}
}

func (suite *IntegrationTestSuite) TestBoostParticipant_SetGet() {
	participant := newBoostParticipant("bze1addr1", "000000000001", "000000000001")

	suite.k.SetBoostParticipant(suite.ctx, participant)

	got, found := suite.k.GetBoostParticipant(suite.ctx, "bze1addr1", "000000000001", "000000000001")
	suite.Require().True(found)
	suite.Require().Equal(participant, got)
}

// TestBoostParticipant_GetNonExistent covers the absent-entry rule: a missing
// record is reported as not found (the settle routine then uses S0 = 0).
func (suite *IntegrationTestSuite) TestBoostParticipant_GetNonExistent() {
	_, found := suite.k.GetBoostParticipant(suite.ctx, "bze1addr1", "000000000001", "000000000001")
	suite.Require().False(found)
}

// TestBoostParticipant_Remove verifies the exact-key delete removes one
// (address, reward, boost) stamp — sibling boosts of the same slice, other
// rewards of the same address and other addresses are untouched.
func (suite *IntegrationTestSuite) TestBoostParticipant_Remove() {
	suite.k.SetBoostParticipant(suite.ctx, newBoostParticipant("bze1addr1", "000000000001", "000000000001"))
	// sibling boost of the same (address, reward) slice: must survive
	suite.k.SetBoostParticipant(suite.ctx, newBoostParticipant("bze1addr1", "000000000001", "000000000002"))
	// addr1 on reward2: must survive
	suite.k.SetBoostParticipant(suite.ctx, newBoostParticipant("bze1addr1", "000000000002", "000000000003"))
	// addr2 on reward1: must survive
	suite.k.SetBoostParticipant(suite.ctx, newBoostParticipant("bze1addr2", "000000000001", "000000000001"))

	suite.k.RemoveBoostParticipant(suite.ctx, "bze1addr1", "000000000001", "000000000001")

	_, found := suite.k.GetBoostParticipant(suite.ctx, "bze1addr1", "000000000001", "000000000001")
	suite.Require().False(found)
	_, found = suite.k.GetBoostParticipant(suite.ctx, "bze1addr1", "000000000001", "000000000002")
	suite.Require().True(found)
	_, found = suite.k.GetBoostParticipant(suite.ctx, "bze1addr1", "000000000002", "000000000003")
	suite.Require().True(found)
	_, found = suite.k.GetBoostParticipant(suite.ctx, "bze1addr2", "000000000001", "000000000001")
	suite.Require().True(found)

	suite.Require().Len(suite.k.GetAllBoostParticipant(suite.ctx), 3)
}

// TestBoostParticipant_Remove_NoOp asserts deleting a missing stamp does
// nothing.
func (suite *IntegrationTestSuite) TestBoostParticipant_Remove_NoOp() {
	suite.k.SetBoostParticipant(suite.ctx, newBoostParticipant("bze1addr1", "000000000001", "000000000001"))

	suite.k.RemoveBoostParticipant(suite.ctx, "bze1addr2", "000000000001", "000000000001")

	suite.Require().Len(suite.k.GetAllBoostParticipant(suite.ctx), 1)
}

func (suite *IntegrationTestSuite) TestBoostParticipant_GetAll() {
	suite.Require().Empty(suite.k.GetAllBoostParticipant(suite.ctx))

	suite.k.SetBoostParticipant(suite.ctx, newBoostParticipant("bze1addr1", "000000000001", "000000000001"))
	suite.k.SetBoostParticipant(suite.ctx, newBoostParticipant("bze1addr2", "000000000001", "000000000001"))
	suite.k.SetBoostParticipant(suite.ctx, newBoostParticipant("bze1addr2", "000000000002", "000000000002"))

	suite.Require().Len(suite.k.GetAllBoostParticipant(suite.ctx), 3)
}
