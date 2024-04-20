package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/types"
	"strconv"
)

func (suite *IntegrationTestSuite) TestPendingUnlockParticipant() {
	list := suite.k.GetAllPendingUnlockParticipant(suite.ctx)
	suite.Require().Empty(list)

	staticEpoch := 500
	max := 10
	for i := 1; i <= max; i++ {
		pup := types.PendingUnlockParticipant{Index: types.CreatePendingUnlockParticipantKey(int64(i), "something")}
		suite.k.SetPendingUnlockParticipant(suite.ctx, pup)

		pup = types.PendingUnlockParticipant{Index: types.CreatePendingUnlockParticipantKey(int64(staticEpoch), strconv.Itoa(i))}
		suite.k.SetPendingUnlockParticipant(suite.ctx, pup)

		list = suite.k.GetAllPendingUnlockParticipant(suite.ctx)
		suite.Require().NotEmpty(list)
		suite.Require().EqualValues(len(list), 2*i)

		list = suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, int64(i))
		suite.Require().NotEmpty(list)
		suite.Require().EqualValues(len(list), 1)
	}

	list = suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, int64(staticEpoch))
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max)

	pup := types.PendingUnlockParticipant{Index: types.CreatePendingUnlockParticipantKey(int64(staticEpoch), "5")}
	suite.k.RemovePendingUnlockParticipant(suite.ctx, pup)

	list = suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, int64(staticEpoch))
	suite.Require().NotEmpty(list)
	suite.Require().EqualValues(len(list), max-1)
}
