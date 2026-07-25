package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/sample"
)

func (suite *IntegrationTestSuite) TestBoostIndex_SetHasRemove() {
	reward := "000000000001"
	addr := sample.AccAddress()

	suite.Require().False(suite.k.HasBoostParticipantIndex(suite.ctx, reward, addr))

	suite.k.SetBoostParticipantIndex(suite.ctx, reward, addr)
	suite.Require().True(suite.k.HasBoostParticipantIndex(suite.ctx, reward, addr))

	suite.k.RemoveBoostParticipantIndex(suite.ctx, reward, addr)
	suite.Require().False(suite.k.HasBoostParticipantIndex(suite.ctx, reward, addr))
}

// TestBoostIndex_CursorPagination checks iterate-from-cursor semantics: full
// walk, bounded batch, strict resume after a cursor, and past-the-end.
func (suite *IntegrationTestSuite) TestBoostIndex_CursorPagination() {
	reward := "000000000001"
	// lexicographically ordered opaque addresses to make key order predictable
	addrs := []string{"bze1aaa", "bze1bbb", "bze1ccc", "bze1ddd"}
	for _, a := range addrs {
		suite.k.SetBoostParticipantIndex(suite.ctx, reward, a)
	}

	// full walk (limit <= 0 = all), in key order
	all := suite.k.GetBoostParticipantsFromCursor(suite.ctx, reward, "", 0)
	suite.Require().Equal(addrs, all)

	// bounded batch
	first2 := suite.k.GetBoostParticipantsFromCursor(suite.ctx, reward, "", 2)
	suite.Require().Equal([]string{"bze1aaa", "bze1bbb"}, first2)

	// resume strictly after the cursor
	afterB := suite.k.GetBoostParticipantsFromCursor(suite.ctx, reward, "bze1bbb", 0)
	suite.Require().Equal([]string{"bze1ccc", "bze1ddd"}, afterB)

	// resume with a limit
	afterAlimit := suite.k.GetBoostParticipantsFromCursor(suite.ctx, reward, "bze1aaa", 2)
	suite.Require().Equal([]string{"bze1bbb", "bze1ccc"}, afterAlimit)

	// past the end
	afterLast := suite.k.GetBoostParticipantsFromCursor(suite.ctx, reward, "bze1ddd", 0)
	suite.Require().Empty(afterLast)
}

// TestBoostIndex_RewardIsolation ensures one reward's participants never leak
// into another reward's iteration.
func (suite *IntegrationTestSuite) TestBoostIndex_RewardIsolation() {
	reward1 := "000000000001"
	reward2 := "000000000002"

	suite.k.SetBoostParticipantIndex(suite.ctx, reward1, "bze1aaa")
	suite.k.SetBoostParticipantIndex(suite.ctx, reward1, "bze1bbb")
	suite.k.SetBoostParticipantIndex(suite.ctx, reward2, "bze1zzz")

	got1 := suite.k.GetBoostParticipantsFromCursor(suite.ctx, reward1, "", 0)
	suite.Require().Equal([]string{"bze1aaa", "bze1bbb"}, got1)

	got2 := suite.k.GetBoostParticipantsFromCursor(suite.ctx, reward2, "", 0)
	suite.Require().Equal([]string{"bze1zzz"}, got2)

	suite.Require().False(suite.k.HasBoostParticipantIndex(suite.ctx, reward1, "bze1zzz"))
}

// TestBoostIndex_RealAddressRoundTrip confirms address extraction from the key
// works for real bech32 addresses (which contain no "/").
func (suite *IntegrationTestSuite) TestBoostIndex_RealAddressRoundTrip() {
	reward := "000000000001"
	addr := sample.AccAddress()

	suite.k.SetBoostParticipantIndex(suite.ctx, reward, addr)

	got := suite.k.GetBoostParticipantsFromCursor(suite.ctx, reward, "", 0)
	suite.Require().Equal([]string{addr}, got)
}
