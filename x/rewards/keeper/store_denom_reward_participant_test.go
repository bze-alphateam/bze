package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *IntegrationTestSuite) TestStoreDenomRewardParticipant_SetGetRemove_MarkerInSync() {
	p := types.DenomRewardParticipant{
		Address:      "bze1user",
		StakingDenom: "ubze",
		Amount:       math.NewInt(1000),
	}

	// set writes both the record and the address-first marker
	suite.k.SetDenomRewardParticipant(suite.ctx, p)

	got, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", "bze1user")
	suite.Require().True(found)
	suite.Require().Equal(p.Amount, got.Amount)
	suite.Require().True(suite.k.HasDenomRewardParticipantMarker(suite.ctx, "bze1user", "ubze"))

	// remove deletes both the record and the marker
	suite.k.RemoveDenomRewardParticipant(suite.ctx, "ubze", "bze1user")
	_, found = suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", "bze1user")
	suite.Require().False(found)
	suite.Require().False(suite.k.HasDenomRewardParticipantMarker(suite.ctx, "bze1user", "ubze"))
}

func (suite *IntegrationTestSuite) TestStoreDenomRewardParticipant_IterateByDenom_Isolation() {
	// two participants in ubze, one in other
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: "bze1a", StakingDenom: "ubze", Amount: math.NewInt(1)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: "bze1b", StakingDenom: "ubze", Amount: math.NewInt(2)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: "bze1a", StakingDenom: "other", Amount: math.NewInt(3)})

	var ubze []string
	suite.k.IterateDenomRewardParticipants(suite.ctx, "ubze", func(_ sdk.Context, p types.DenomRewardParticipant) bool {
		suite.Require().Equal("ubze", p.StakingDenom)
		ubze = append(ubze, p.Address)
		return false
	})
	suite.Require().Equal([]string{"bze1a", "bze1b"}, ubze)

	var other []string
	suite.k.IterateDenomRewardParticipants(suite.ctx, "other", func(_ sdk.Context, p types.DenomRewardParticipant) bool {
		other = append(other, p.Address)
		return false
	})
	suite.Require().Equal([]string{"bze1a"}, other)

	suite.Require().Len(suite.k.GetAllDenomRewardParticipant(suite.ctx), 3)
}

func (suite *IntegrationTestSuite) TestStoreDenomRewardParticipant_IterateUserDenomRewards_ViaMarker() {
	// bze1a joins ubze and other; bze1b joins only ubze
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: "bze1a", StakingDenom: "ubze", Amount: math.NewInt(1)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: "bze1a", StakingDenom: "other", Amount: math.NewInt(3)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: "bze1b", StakingDenom: "ubze", Amount: math.NewInt(2)})

	var aDenoms []string
	suite.k.IterateUserDenomRewards(suite.ctx, "bze1a", func(_ sdk.Context, p types.DenomRewardParticipant) bool {
		suite.Require().Equal("bze1a", p.Address)
		aDenoms = append(aDenoms, p.StakingDenom)
		return false
	})
	// marker keys sort lexicographically by staking denom
	suite.Require().Equal([]string{"other", "ubze"}, aDenoms)

	// removing one participation drops it from the user's view (marker in sync)
	suite.k.RemoveDenomRewardParticipant(suite.ctx, "other", "bze1a")
	aDenoms = nil
	suite.k.IterateUserDenomRewards(suite.ctx, "bze1a", func(_ sdk.Context, p types.DenomRewardParticipant) bool {
		aDenoms = append(aDenoms, p.StakingDenom)
		return false
	})
	suite.Require().Equal([]string{"ubze"}, aDenoms)
}
