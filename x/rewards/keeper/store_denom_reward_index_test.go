package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestStoreDenomRewardIndex_SetGetRemove_ExactKey() {
	idx := types.DenomRewardParticipantIndex{
		Address:      "bze1a",
		StakingDenom: "ubze",
		PrizeDenom:   "uprize",
		Index:        math.LegacyMustNewDecFromStr("1.5"),
	}

	_, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, "bze1a", "ubze", "uprize")
	suite.Require().False(found)

	suite.k.SetDenomRewardParticipantIndex(suite.ctx, idx)

	got, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, "bze1a", "ubze", "uprize")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("1.5").Equal(got.Index))

	// exact-key: a different prize denom is a different record
	_, found = suite.k.GetDenomRewardParticipantIndex(suite.ctx, "bze1a", "ubze", "other")
	suite.Require().False(found)

	suite.k.RemoveDenomRewardParticipantIndex(suite.ctx, "bze1a", "ubze", "uprize")
	_, found = suite.k.GetDenomRewardParticipantIndex(suite.ctx, "bze1a", "ubze", "uprize")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestStoreDenomRewardIndex_RemoveAll_RemovesExactlyOneSlice() {
	// (bze1a, ubze): p1, p2 ; (bze1a, other): p1 ; (bze1b, ubze): p1
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{Address: "bze1a", StakingDenom: "ubze", PrizeDenom: "p1", Index: math.LegacyMustNewDecFromStr("0")})
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{Address: "bze1a", StakingDenom: "ubze", PrizeDenom: "p2", Index: math.LegacyMustNewDecFromStr("0")})
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{Address: "bze1a", StakingDenom: "other", PrizeDenom: "p1", Index: math.LegacyMustNewDecFromStr("0")})
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{Address: "bze1b", StakingDenom: "ubze", PrizeDenom: "p1", Index: math.LegacyMustNewDecFromStr("0")})

	// sanity: iterate the (bze1a, ubze) slice
	var slice []string
	suite.k.IterateParticipantIndexes(suite.ctx, "bze1a", "ubze", func(_ sdk.Context, idx types.DenomRewardParticipantIndex) bool {
		suite.Require().Equal("bze1a", idx.Address)
		suite.Require().Equal("ubze", idx.StakingDenom)
		slice = append(slice, idx.PrizeDenom)
		return false
	})
	suite.Require().Equal([]string{"p1", "p2"}, slice)

	// remove exactly the (bze1a, ubze) slice
	suite.k.RemoveAllParticipantIndexes(suite.ctx, "bze1a", "ubze")

	// (bze1a, ubze) is gone
	_, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, "bze1a", "ubze", "p1")
	suite.Require().False(found)
	_, found = suite.k.GetDenomRewardParticipantIndex(suite.ctx, "bze1a", "ubze", "p2")
	suite.Require().False(found)

	// the other two slices are untouched
	_, found = suite.k.GetDenomRewardParticipantIndex(suite.ctx, "bze1a", "other", "p1")
	suite.Require().True(found)
	_, found = suite.k.GetDenomRewardParticipantIndex(suite.ctx, "bze1b", "ubze", "p1")
	suite.Require().True(found)

	suite.Require().Len(suite.k.GetAllDenomRewardParticipantIndex(suite.ctx), 2)
}
