package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	dao "github.com/bze-alphateam/bze/x/dao/module"
	"github.com/bze-alphateam/bze/x/dao/testutil"
	"github.com/bze-alphateam/bze/x/dao/types"
)

// liveState builds state through the messages and hooks: DAO 1 a static list
// with an external admin, a pending nomination and a 40ubze deposit; DAO 2 a
// self-owned stake DAO over SR 1 and the ubze DR. Proposal 1 (DAO 1) and 2
// (DAO 2) are in voting with ballots, 2's with two shares and memberB's
// reduced by an exit; proposal 3 (DAO 2) was voted and cancelled, so its vote
// index entry is a leftover. memberC's join after the submit is recorded.
func (suite *IntegrationTestSuite) liveState() {
	suite.stubRewards(defaultRewardsState())
	s := suite.stubStakes()

	suite.createDao(1, func(msg *types.MsgCreateDao) {
		msg.Admin = adminAddr.String()
		msg.Config.ProposalDeposit = coinPtr(40)
	})
	suite.Require().NoError(suite.nominate(adminAddr, 1, memberC.String()))
	suite.createDao(2, func(msg *types.MsgCreateDao) {
		stake := validStakeDaoMsg([]string{"1"}, stakeDenom)
		msg.Type, msg.Members, msg.Electorate = stake.Type, stake.Members, stake.Electorate
	})

	suite.atBlock(5)
	s.joinSr("1", memberA, 30)
	s.joinDr(stakeDenom, memberA, 20)
	s.joinSr("1", memberB, 10)
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardJoin(suite.ctx, "1", memberB.String(), math.NewInt(10), stakeDenom))

	suite.atBlock(10)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), memberA, types.ModuleName, ubze(40)).Return(nil)
	static, err := suite.submit(memberA, 1, false)
	suite.Require().NoError(err)
	stake, err := suite.submit(memberA, 2, false)
	suite.Require().NoError(err)
	cancelled, err := suite.submit(memberA, 2, true)
	suite.Require().NoError(err)

	suite.Require().NoError(suite.vote(memberA, static, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberB, static, types.VoteOption_NO))
	suite.Require().NoError(suite.vote(memberA, stake, types.VoteOption_YES))
	suite.Require().NoError(suite.vote(memberB, stake, types.VoteOption_NO))
	suite.Require().NoError(suite.vote(memberA, cancelled, types.VoteOption_ABSTAIN))
	suite.Require().NoError(suite.cancel(memberA, cancelled))

	suite.atBlock(11)
	delete(s.sr["1"], memberB.String())
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardExit(suite.ctx, "1", memberB.String(), math.NewInt(10), stakeDenom))
	s.joinSr("1", memberC, 40)
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardJoin(suite.ctx, "1", memberC.String(), math.NewInt(40), stakeDenom))
}

// The state the messages build exports, passes every import check (I1 against
// the escrowed deposit) and imports into a fresh keeper, which exports it
// unchanged.
func (suite *IntegrationTestSuite) TestGenesis_LiveStateRoundTrip() {
	suite.liveState()

	exported := dao.ExportGenesis(suite.ctx, suite.k)
	suite.Require().NoError(exported.Validate())
	suite.Require().Len(exported.Daos, 2)
	suite.Require().Len(exported.Proposals, 3)
	suite.Require().Len(exported.Ballots, 5)
	suite.Require().Equal(memberC.String(), exported.Daos[0].PendingAdmin)
	suite.Require().Equal([]types.VoteIndexEntry{
		{Source: string(srSource), Address: memberA.String(), ProposalId: 2},
		{Source: string(srSource), Address: memberA.String(), ProposalId: 3},
		{Source: string(drSource), Address: memberA.String(), ProposalId: 2},
		{Source: string(drSource), Address: memberA.String(), ProposalId: 3},
	}, exported.VoteIndex)
	suite.Require().Equal([]types.JoinRecord{{Source: string(srSource), Address: memberC.String(), Height: 11}}, exported.JoinRecords)
	// DAO 2 and proposal 2 pin both sources; cancelled 3 no longer does
	suite.Require().Equal([]types.PinCount{{Source: string(drSource), Count: 2}, {Source: string(srSource), Count: 2}}, exported.Pins)

	ballot, _ := suite.k.GetBallot(suite.ctx, 2, memberB)
	suite.Require().Equal(math.ZeroInt(), ballot.Weight)
	suite.Require().Empty(ballot.Shares)

	cdc := keepertest.DaoCodec()
	exportedJSON := cdc.MustMarshalJSON(exported)
	var decoded types.GenesisState
	cdc.MustUnmarshalJSON(exportedJSON, &decoded)

	bank := testutil.NewMockBankKeeper(gomock.NewController(suite.T()))
	bank.EXPECT().GetAllBalances(gomock.Any(), authtypes.NewModuleAddress(types.ModuleName)).Return(ubze(40)).AnyTimes()
	fresh, freshCtx := keepertest.DaoKeeper(suite.T(), cdc, nil, nil, bank, nil, nil)
	dao.InitGenesis(freshCtx, &fresh, decoded)

	suite.Require().Equal(exportedJSON, cdc.MustMarshalJSON(dao.ExportGenesis(freshCtx, &fresh)))

	// the imported state answers the queries as the original does
	for _, voter := range []sdk.AccAddress{memberA, memberB} {
		want, err := suite.k.Ballot(suite.ctx, &types.QueryBallotRequest{ProposalId: 2, Voter: voter.String()})
		suite.Require().NoError(err)
		got, err := fresh.Ballot(freshCtx, &types.QueryBallotRequest{ProposalId: 2, Voter: voter.String()})
		suite.Require().NoError(err)
		suite.Require().Equal(want, got)
	}
	record, err := fresh.JoinRecord(freshCtx, &types.QueryJoinRecordRequest{Source: string(srSource), Address: memberC.String()})
	suite.Require().NoError(err)
	suite.Require().Equal(int64(11), record.Height)
	suite.Require().True(fresh.HasVoteIndex(freshCtx, drSource, memberA, 2))
	suite.Require().True(fresh.HasDaoAdminIndex(freshCtx, adminAddr, 1))
}

// Unpinning a source keeps its join records, so a live chain holds join
// records of sources with pin 0; the import accepts them. They are inert: a
// later proposal's submit height is above every such record.
func (suite *IntegrationTestSuite) TestGenesis_JoinRecordOfUnpinnedSource() {
	suite.stubRewards(defaultRewardsState())
	suite.createAdminStakeDao([]string{"1"}, "")
	suite.atBlock(5)
	suite.Require().NoError(suite.k.StakingRewardHooks().AfterStakingRewardJoin(suite.ctx, "1", memberC.String(), math.NewInt(40), stakeDenom))
	suite.Require().NoError(suite.updateElectorate(daoAddress(1), 1, []string{"5"}, ""))
	suite.Require().Zero(suite.k.GetSourcePins(suite.ctx, srSource))

	exported := dao.ExportGenesis(suite.ctx, suite.k)
	suite.Require().Equal([]types.JoinRecord{{Source: string(srSource), Address: memberC.String(), Height: 5}}, exported.JoinRecords)
	suite.Require().NoError(exported.Validate())

	fresh, freshCtx := keepertest.DaoKeeper(suite.T(), keepertest.DaoCodec(), nil, nil, testutil.NewMockBankKeeper(gomock.NewController(suite.T())), nil, nil)
	dao.InitGenesis(freshCtx, &fresh, *exported)
	suite.Require().Equal(exported, dao.ExportGenesis(freshCtx, &fresh))
}
