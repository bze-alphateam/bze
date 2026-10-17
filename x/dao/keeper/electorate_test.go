package keeper_test

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// The static-list electorate rules: a non-empty list of unique valid addresses,
// capped by max_static_members, and never mixed with a stake electorate (D29).
func (suite *IntegrationTestSuite) TestCreateDao_StaticElectorate() {
	for _, tc := range []struct {
		desc    string
		mutate  func(m *types.MsgCreateDao)
		wantErr error
	}{
		{desc: "one member", mutate: func(m *types.MsgCreateDao) { m.Members = []string{memberA.String()} }},
		{desc: "members at the cap", mutate: func(m *types.MsgCreateDao) {
			m.Members = []string{memberA.String(), memberB.String(), memberC.String()}
		}},
		{desc: "members above the cap", mutate: func(m *types.MsgCreateDao) {
			m.Members = []string{memberA.String(), memberB.String(), memberC.String(), adminAddr.String()}
		}, wantErr: types.ErrInvalidMembers},
		{desc: "no members", mutate: func(m *types.MsgCreateDao) { m.Members = nil }, wantErr: types.ErrInvalidMembers},
		{desc: "duplicate member", mutate: func(m *types.MsgCreateDao) {
			m.Members = []string{memberA.String(), memberB.String(), memberA.String()}
		}, wantErr: types.ErrInvalidMembers},
		{desc: "duplicate member in another case", mutate: func(m *types.MsgCreateDao) {
			m.Members = []string{memberA.String(), strings.ToUpper(memberA.String())}
		}, wantErr: types.ErrInvalidMembers},
		{desc: "invalid member address", mutate: func(m *types.MsgCreateDao) { m.Members = []string{memberA.String(), "bze1nope"} }, wantErr: types.ErrInvalidMembers},
		{desc: "empty member address", mutate: func(m *types.MsgCreateDao) { m.Members = []string{""} }, wantErr: types.ErrInvalidMembers},
		{desc: "D29: members and staking rewards", mutate: func(m *types.MsgCreateDao) {
			m.Electorate = &types.StakeElectorate{SrIds: []string{"1"}}
		}, wantErr: types.ErrInvalidElectorate},
		{desc: "D29: members and a denom reward", mutate: func(m *types.MsgCreateDao) {
			m.Electorate = &types.StakeElectorate{DrDenom: "ubze"}
		}, wantErr: types.ErrInvalidElectorate},
		{desc: "D29: stake electorate without members", mutate: func(m *types.MsgCreateDao) {
			m.Members = nil
			m.Electorate = &types.StakeElectorate{SrIds: []string{"1"}}
		}, wantErr: types.ErrInvalidElectorate},
		{desc: "an empty electorate object is no electorate", mutate: func(m *types.MsgCreateDao) {
			m.Electorate = &types.StakeElectorate{}
		}},
	} {
		suite.Run(tc.desc, func() {
			suite.SetupTest()
			params := types.DefaultParams()
			params.MaxStaticMembers = 3
			suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

			msg := validCreateDaoMsg()
			tc.mutate(msg)
			if tc.wantErr == nil {
				suite.expectCreation(1)
			}

			res, err := suite.msgServer.CreateDao(suite.ctx, msg)
			if tc.wantErr != nil {
				suite.Require().ErrorIs(err, tc.wantErr)
				suite.Require().Zero(suite.k.GetDaoCounter(suite.ctx))
				return
			}

			suite.Require().NoError(err)
			dao, _ := suite.k.GetDao(suite.ctx, res.Id)
			suite.Require().Equal(msg.Members, dao.Members)
			suite.Require().Nil(dao.Electorate)
		})
	}
}

// The default cap is 128 members.
func (suite *IntegrationTestSuite) TestCreateDao_DefaultMemberCap() {
	members := make([]string, 129)
	for i := range members {
		members[i] = memberAddress(i).String()
	}

	msg := validCreateDaoMsg()
	msg.Members = members
	_, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidMembers)

	msg.Members = members[:128]
	suite.expectCreation(1)
	_, err = suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)
}

// Members written in upper case are stored in canonical (lower case) form.
func (suite *IntegrationTestSuite) TestCreateDao_MembersStoredCanonical() {
	suite.expectCreation(1)
	msg := validCreateDaoMsg()
	msg.Members = []string{strings.ToUpper(memberA.String()), memberB.String()}

	res, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)

	dao, _ := suite.k.GetDao(suite.ctx, res.Id)
	suite.Require().Equal([]string{memberA.String(), memberB.String()}, dao.Members)
}

func memberAddress(i int) sdk.AccAddress {
	addr := make([]byte, 20)
	addr[0], addr[1] = byte(i>>8), byte(i)
	return addr
}
