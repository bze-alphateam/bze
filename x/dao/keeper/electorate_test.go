package keeper_test

import (
	"encoding/json"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/dao/types"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
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

const (
	stakeDenom = "ubze"
	otherDenom = "uother"
)

// rewardsState is the x/rewards state seen by the stake electorate tests.
type rewardsState struct {
	srs map[string]rewardstypes.StakingReward
	drs map[string]rewardstypes.DenomReward
}

func stakingReward(id, denom string, lock uint32, staked string) rewardstypes.StakingReward {
	return rewardstypes.StakingReward{RewardId: id, StakingDenom: denom, Lock: lock, StakedAmount: staked}
}

func denomReward(denom string, lock uint32, staked int64) rewardstypes.DenomReward {
	return rewardstypes.DenomReward{StakingDenom: denom, Lock: lock, StakedAmount: math.NewInt(staked)}
}

// defaultRewardsState: SR 1, 5, 6 (ubze, lock 5, staked), SR 2 (ubze, nothing
// staked), SR 3 (another denom), SR 4 (lock 4); DR ubze (lock 5), DR uother,
// DR ulow (lock 4); SR 9 + DR uzero hold no stake.
func defaultRewardsState() rewardsState {
	return rewardsState{
		srs: map[string]rewardstypes.StakingReward{
			"1": stakingReward("1", stakeDenom, 5, "100"),
			"2": stakingReward("2", stakeDenom, 5, "0"),
			"3": stakingReward("3", otherDenom, 5, "10"),
			"4": stakingReward("4", stakeDenom, 4, "10"),
			"5": stakingReward("5", stakeDenom, 5, "10"),
			"6": stakingReward("6", stakeDenom, 30, "10"),
			"9": stakingReward("9", "uzero", 5, "0"),
		},
		drs: map[string]rewardstypes.DenomReward{
			stakeDenom: denomReward(stakeDenom, 5, 50),
			otherDenom: denomReward(otherDenom, 5, 1),
			"ulow":     denomReward("ulow", 4, 1),
			"uzero":    denomReward("uzero", 5, 0),
		},
	}
}

// stubRewards serves the given x/rewards state to the dao keeper.
func (suite *IntegrationTestSuite) stubRewards(state rewardsState) {
	suite.rewards.EXPECT().GetStakingReward(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ sdk.Context, id string) (rewardstypes.StakingReward, bool) {
			sr, ok := state.srs[id]
			return sr, ok
		}).AnyTimes()
	suite.rewards.EXPECT().GetDenomReward(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ sdk.Context, denom string) (rewardstypes.DenomReward, bool) {
			dr, ok := state.drs[denom]
			return dr, ok
		}).AnyTimes()
}

func validStakeDaoMsg(srIds []string, drDenom string) *types.MsgCreateDao {
	msg := validCreateDaoMsg()
	msg.Type = types.DaoType_DAO_TYPE_STAKE
	msg.Members = nil
	msg.Electorate = &types.StakeElectorate{SrIds: srIds, DrDenom: drDenom}

	return msg
}

// The stake electorate rules: up to max_sr_per_dao unique existing staking
// rewards and/or one existing Denom Reward, one staking denom (D29), a lock of
// at least min_electorate_lock_days (D28), no member list (D29), and a live
// electorate unless an external admin sets it up (D11).
func (suite *IntegrationTestSuite) TestCreateDao_StakeElectorate() {
	for _, tc := range []struct {
		desc    string
		msg     func() *types.MsgCreateDao
		wantErr string
	}{
		{desc: "staking rewards only", msg: func() *types.MsgCreateDao { return validStakeDaoMsg([]string{"1", "2"}, "") }},
		{desc: "denom reward only", msg: func() *types.MsgCreateDao { return validStakeDaoMsg(nil, stakeDenom) }},
		{desc: "staking rewards and a denom reward", msg: func() *types.MsgCreateDao { return validStakeDaoMsg([]string{"1", "5"}, stakeDenom) }},
		{desc: "staking rewards at the cap", msg: func() *types.MsgCreateDao { return validStakeDaoMsg([]string{"1", "2", "5"}, "") }},
		{desc: "staking rewards above the cap", msg: func() *types.MsgCreateDao {
			return validStakeDaoMsg([]string{"1", "2", "5", "6"}, "")
		}, wantErr: "at most 3 staking rewards"},
		{desc: "D14: duplicate staking reward", msg: func() *types.MsgCreateDao {
			return validStakeDaoMsg([]string{"1", "5", "1"}, "")
		}, wantErr: "duplicate staking reward 1"},
		{desc: "empty staking reward id", msg: func() *types.MsgCreateDao { return validStakeDaoMsg([]string{""}, "") }, wantErr: "empty staking reward id"},
		{desc: "unknown staking reward", msg: func() *types.MsgCreateDao { return validStakeDaoMsg([]string{"1", "99"}, "") }, wantErr: "staking reward 99 not found"},
		{desc: "unknown denom reward", msg: func() *types.MsgCreateDao { return validStakeDaoMsg([]string{"1"}, "unone") }, wantErr: "denom reward unone not found"},
		{desc: "invalid denom reward denom", msg: func() *types.MsgCreateDao { return validStakeDaoMsg(nil, "1x") }, wantErr: "invalid denom reward denom"},

		{desc: "D28: staking reward lock below the min", msg: func() *types.MsgCreateDao {
			return validStakeDaoMsg([]string{"1", "4"}, "")
		}, wantErr: "staking reward 4 has a 4 day lock, at least 5 days are required"},
		{desc: "D28: denom reward lock below the min", msg: func() *types.MsgCreateDao {
			return validStakeDaoMsg(nil, "ulow")
		}, wantErr: "denom reward ulow has a 4 day lock, at least 5 days are required"},
		{desc: "D28: lock exactly at the min", msg: func() *types.MsgCreateDao { return validStakeDaoMsg([]string{"5"}, stakeDenom) }},

		{desc: "D29: staking reward and denom reward with different denoms", msg: func() *types.MsgCreateDao {
			return validStakeDaoMsg([]string{"1"}, otherDenom)
		}, wantErr: "all sources must share one staking denom"},
		{desc: "D29: two staking rewards with different denoms", msg: func() *types.MsgCreateDao {
			return validStakeDaoMsg([]string{"1", "3"}, "")
		}, wantErr: "all sources must share one staking denom"},
		{desc: "D29: stake dao with members", msg: func() *types.MsgCreateDao {
			msg := validStakeDaoMsg([]string{"1"}, "")
			msg.Members = []string{memberA.String()}
			return msg
		}, wantErr: "cannot have a member list"},

		{desc: "D11: self-owned without an electorate", msg: func() *types.MsgCreateDao {
			msg := validStakeDaoMsg(nil, "")
			msg.Electorate = nil
			return msg
		}, wantErr: "needs a live electorate"},
		{desc: "D11: self-owned with an empty electorate", msg: func() *types.MsgCreateDao { return validStakeDaoMsg(nil, "") }, wantErr: "needs a live electorate"},
		{desc: "D11: self-owned with nothing staked", msg: func() *types.MsgCreateDao {
			return validStakeDaoMsg([]string{"9"}, "uzero")
		}, wantErr: "needs a live electorate"},
		{desc: "D11: admin given as the dao address is self-owned", msg: func() *types.MsgCreateDao {
			msg := validStakeDaoMsg([]string{"2"}, "")
			msg.Admin = daoAddress(1).String()
			return msg
		}, wantErr: "needs a live electorate"},
		{desc: "D11: external admin without an electorate", msg: func() *types.MsgCreateDao {
			msg := validStakeDaoMsg(nil, "")
			msg.Electorate = nil
			msg.Admin = adminAddr.String()
			return msg
		}},
		{desc: "D11: external admin with nothing staked", msg: func() *types.MsgCreateDao {
			msg := validStakeDaoMsg([]string{"9"}, "uzero")
			msg.Admin = adminAddr.String()
			return msg
		}},
	} {
		suite.Run(tc.desc, func() {
			suite.SetupTest()
			suite.stubRewards(defaultRewardsState())
			params := types.DefaultParams()
			params.MaxSrPerDao = 3
			suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

			msg := tc.msg()
			if tc.wantErr == "" {
				suite.expectCreation(1)
			}

			res, err := suite.msgServer.CreateDao(suite.ctx, msg)
			if tc.wantErr != "" {
				suite.Require().ErrorIs(err, types.ErrInvalidElectorate)
				suite.Require().ErrorContains(err, tc.wantErr)
				suite.Require().Zero(suite.k.GetDaoCounter(suite.ctx))
				for _, id := range []string{"1", "2", "4", "5"} {
					suite.Require().Zero(suite.k.GetSourcePins(suite.ctx, types.StakingRewardSource(id)))
				}
				return
			}

			suite.Require().NoError(err)
			dao, found := suite.k.GetDao(suite.ctx, res.Id)
			suite.Require().True(found)
			suite.Require().Equal(types.DaoType_DAO_TYPE_STAKE, dao.Type)
			suite.Require().Empty(dao.Members)
			if msg.Electorate.IsEmpty() {
				suite.Require().Nil(dao.Electorate)
				return
			}

			suite.Require().Equal(msg.Electorate, dao.Electorate)
			for _, id := range msg.Electorate.SrIds {
				suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, types.StakingRewardSource(id)))
			}
			if msg.Electorate.DrDenom != "" {
				suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, types.DenomRewardSource(msg.Electorate.DrDenom)))
			}
		})
	}
}

// The default cap is 5 staking rewards.
func (suite *IntegrationTestSuite) TestCreateDao_DefaultStakingRewardCap() {
	state := defaultRewardsState()
	state.srs["7"] = stakingReward("7", stakeDenom, 5, "1")
	suite.stubRewards(state)

	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1", "2", "5", "6", "7", "4"}, ""))
	suite.Require().ErrorContains(err, "at most 5 staking rewards")

	suite.expectCreation(1)
	_, err = suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1", "2", "5", "6", "7"}, ""))
	suite.Require().NoError(err)
}

// I4: every DAO using a source adds one pin to it.
func (suite *IntegrationTestSuite) TestCreateDao_PinsCountEveryDao() {
	suite.stubRewards(defaultRewardsState())

	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1", "2"}, stakeDenom))
	suite.Require().NoError(err)

	suite.expectCreation(2)
	_, err = suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"1"}, stakeDenom))
	suite.Require().NoError(err)

	suite.Require().Equal(uint64(2), suite.k.GetSourcePins(suite.ctx, types.StakingRewardSource("1")))
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, types.StakingRewardSource("2")))
	suite.Require().Equal(uint64(2), suite.k.GetSourcePins(suite.ctx, types.DenomRewardSource(stakeDenom)))
	suite.Require().Zero(suite.k.GetSourcePins(suite.ctx, types.StakingRewardSource("5")))
}

// D28: raising min_electorate_lock_days later leaves existing DAOs as they are;
// only new electorates must meet the new minimum.
func (suite *IntegrationTestSuite) TestCreateDao_RaisedMinLockKeepsExistingDaos() {
	suite.stubRewards(defaultRewardsState())

	suite.expectCreation(1)
	msg := validStakeDaoMsg([]string{"5"}, stakeDenom)
	_, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)
	before, _ := suite.k.GetDao(suite.ctx, 1)

	params := types.DefaultParams()
	params.MinElectorateLockDays = 10
	_, err = suite.msgServer.UpdateParams(suite.ctx, &types.MsgUpdateParams{Authority: suite.k.GetAuthority(), Params: params})
	suite.Require().NoError(err)

	after, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().True(found)
	suite.Require().Equal(before, after)
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, types.StakingRewardSource("5")))

	_, err = suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"5"}, stakeDenom))
	suite.Require().ErrorContains(err, "at least 10 days are required")

	suite.expectCreation(2)
	_, err = suite.msgServer.CreateDao(suite.ctx, validStakeDaoMsg([]string{"6"}, ""))
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) updateConfig(authority sdk.AccAddress, daoId uint64, config types.DaoConfig) error {
	_, err := suite.msgServer.UpdateDaoConfig(suite.ctx, types.NewMsgUpdateDaoConfig(authority.String(), daoId, config))
	return err
}

// requireConfig checks the stored config of DAO 1.
func (suite *IntegrationTestSuite) requireConfig(want types.DaoConfig) {
	dao, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().True(found)
	suite.Require().Equal(want, dao.Config)
}

// requireConfigEvent checks the fields of the last EventDaoConfigUpdated of DAO 1.
func (suite *IntegrationTestSuite) requireConfigEvent(fields ...string) {
	attrs := suite.lastEventAttrs(&types.EventDaoConfigUpdated{})
	suite.Require().NotNil(attrs)
	suite.Require().Equal(`"1"`, attrs["id"])

	got := []string{}
	if raw, ok := attrs["fields"]; ok {
		suite.Require().NoError(json.Unmarshal([]byte(raw), &got))
	}
	if fields == nil {
		fields = []string{}
	}
	suite.Require().Equal(fields, got)
}

// Business Logic §3: config messages accept only the DAO account. The admin
// signing directly and any other address are refused; nothing changes.
func (suite *IntegrationTestSuite) TestUpdateDaoConfig_DirectSignersRejected() {
	suite.createAdminDao()
	next := validConfig()
	next.Threshold = dec("0.6")

	for _, signer := range []sdk.AccAddress{adminAddr, memberA, attackerAddr, daoAddress(2)} {
		suite.Require().ErrorIs(suite.updateConfig(signer, 1, next), types.ErrUnauthorizedSigner, signer.String())
	}

	suite.requireConfig(validConfig())
	suite.Require().False(suite.hasEvent("bze.dao.EventDaoConfigUpdated"))
}

func (suite *IntegrationTestSuite) TestUpdateDaoConfig_UnknownDao() {
	suite.createStaticDao(noConfigChange)

	suite.Require().ErrorIs(suite.updateConfig(daoAddress(2), 2, validConfig()), types.ErrDaoNotFound)

	_, err := suite.msgServer.UpdateDaoConfig(suite.ctx, types.NewMsgUpdateDaoConfig(daoAddress(1).String(), 0, validConfig()))
	suite.Require().ErrorIs(err, types.ErrDaoNotFound)
}

func (suite *IntegrationTestSuite) TestUpdateDaoConfig_InvalidAuthority() {
	suite.createStaticDao(noConfigChange)

	_, err := suite.msgServer.UpdateDaoConfig(suite.ctx, types.NewMsgUpdateDaoConfig("bad", 1, validConfig()))
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidAddress)
}

// Every config bound of the create matrix holds here too: a rejected update
// leaves the stored config as it was, an accepted one replaces it whole.
func (suite *IntegrationTestSuite) TestUpdateDaoConfig_ValidationMatrix() {
	for _, tc := range configCases() {
		suite.Run(tc.desc, func() {
			suite.SetupTest()
			suite.createStaticDao(noConfigChange)
			next := validConfig()
			tc.mutate(&next)

			err := suite.updateConfig(daoAddress(1), 1, next)
			if tc.wantErr != nil {
				suite.Require().ErrorIs(err, tc.wantErr)
				suite.requireConfig(validConfig())
				return
			}

			suite.Require().NoError(err)
			suite.requireConfig(next)
		})
	}
}

// The period bounds are the current params, as at creation.
func (suite *IntegrationTestSuite) TestUpdateDaoConfig_PeriodBoundsFollowParams() {
	suite.createStaticDao(noConfigChange)
	params := types.DefaultParams()
	params.MinVotingPeriod = 48 * time.Hour
	params.MaxVotingPeriod = 96 * time.Hour
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	// the stored config predates the raised min: only a new one is checked
	err := suite.updateConfig(daoAddress(1), 1, validConfig())
	suite.Require().ErrorIs(err, types.ErrInvalidConfig)
	suite.Require().ErrorContains(err, "expedited_period")

	next := validConfig()
	next.ExpeditedPeriod = 48 * time.Hour
	next.VotingPeriod = 96*time.Hour + time.Nanosecond
	err = suite.updateConfig(daoAddress(1), 1, next)
	suite.Require().ErrorIs(err, types.ErrInvalidConfig)
	suite.Require().ErrorContains(err, "voting_period")

	next.VotingPeriod = 96 * time.Hour
	suite.Require().NoError(suite.updateConfig(daoAddress(1), 1, next))
	suite.requireConfig(next)
}

// The event lists exactly the changed fields, by proto name, in proto order.
func (suite *IntegrationTestSuite) TestUpdateDaoConfig_EventListsChangedFields() {
	for _, tc := range []struct {
		desc   string
		mutate func(c *types.DaoConfig)
		fields []string
	}{
		{desc: "nothing changed", mutate: noConfigChange},
		{desc: "threshold only", mutate: func(c *types.DaoConfig) { c.Threshold = dec("0.6") }, fields: []string{"threshold"}},
		{desc: "same value written differently", mutate: func(c *types.DaoConfig) { c.Threshold = dec("0.500000000000000000") }},
		{desc: "several, set out of order", mutate: func(c *types.DaoConfig) {
			c.KeptDepositDestination = types.FundsDestination_TREASURY
			c.OpenSubmission = true
			c.VotingPeriod = 96 * time.Hour
			c.MinQuorum = dec("0.3")
			c.ProposalDeposit = coinPtr(10)
		}, fields: []string{"min_quorum", "voting_period", "open_submission", "proposal_deposit", "kept_deposit_destination"}},
		{desc: "every field", mutate: func(c *types.DaoConfig) {
			*c = types.DaoConfig{
				MinQuorum:              dec("0.5"),
				Threshold:              dec("0.6"),
				VotingPeriod:           48 * time.Hour,
				AllowExpedite:          false,
				ExpeditedPeriod:        12 * time.Hour,
				ExpediteQuorum:         dec("0.7"),
				ExpediteThreshold:      dec("0.8"),
				OpenSubmission:         true,
				ProposalCost:           coinPtr(1),
				ProposalDeposit:        coinPtr(2),
				DepositRefundPolicy:    types.DepositRefundPolicy_NEVER,
				CostDestination:        types.FundsDestination_TREASURY,
				KeptDepositDestination: types.FundsDestination_TREASURY,
			}
		}, fields: []string{
			"min_quorum", "threshold", "voting_period", "allow_expedite", "expedited_period", "expedite_quorum", "expedite_threshold",
			"open_submission", "proposal_cost", "proposal_deposit", "deposit_refund_policy", "cost_destination", "kept_deposit_destination",
		}},
		{desc: "zero cost set where none was", mutate: func(c *types.DaoConfig) { c.ProposalCost = coinPtr(0) }, fields: []string{"proposal_cost"}},
	} {
		suite.Run(tc.desc, func() {
			suite.SetupTest()
			suite.createStaticDao(noConfigChange)
			next := validConfig()
			tc.mutate(&next)

			suite.Require().NoError(suite.updateConfig(daoAddress(1), 1, next))
			suite.requireConfigEvent(tc.fields...)
		})
	}
}

func (suite *IntegrationTestSuite) updateElectorate(authority sdk.AccAddress, daoId uint64, srIds []string, drDenom string) error {
	msg := types.NewMsgUpdateElectorate(authority.String(), daoId, &types.StakeElectorate{SrIds: srIds, DrDenom: drDenom})
	_, err := suite.msgServer.UpdateElectorate(suite.ctx, msg)
	return err
}

// createAdminStakeDao creates stake DAO 1 with adminAddr as its external admin,
// voted by the given sources.
func (suite *IntegrationTestSuite) createAdminStakeDao(srIds []string, drDenom string) {
	suite.expectCreation(1)
	msg := validStakeDaoMsg(srIds, drDenom)
	msg.Admin = adminAddr.String()
	_, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)
}

// requireElectorate checks DAO 1's stored sources; no sources is a nil electorate.
func (suite *IntegrationTestSuite) requireElectorate(srIds []string, drDenom string) {
	dao, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().True(found)
	if len(srIds) == 0 && drDenom == "" {
		suite.Require().Nil(dao.Electorate)
		return
	}

	suite.Require().Equal(&types.StakeElectorate{SrIds: srIds, DrDenom: drDenom}, dao.Electorate)
}

// requirePinInvariant checks I4 for every source of defaultRewardsState: its
// pin count is the number of the given DAOs whose electorate names it plus the
// number of the given proposals still open that pinned it.
func (suite *IntegrationTestSuite) requirePinInvariant(daoIds []uint64, proposalIds []uint64) {
	want := map[string]uint64{}
	for _, daoId := range daoIds {
		dao, found := suite.k.GetDao(suite.ctx, daoId)
		suite.Require().True(found)
		for _, source := range electorateSourceKeys(dao.Electorate) {
			want[source]++
		}
	}

	for _, proposalId := range proposalIds {
		proposal, found := suite.k.GetProposal(suite.ctx, proposalId)
		suite.Require().True(found)
		if !suite.k.IsOpenProposal(suite.ctx, proposal.DaoId, proposalId) {
			continue
		}
		for _, source := range electorateSourceKeys(proposal.PinnedElectorate) {
			want[source]++
		}
	}

	state := defaultRewardsState()
	for id := range state.srs {
		source := types.StakingRewardSource(id)
		suite.Require().Equal(want[string(source)], suite.k.GetSourcePins(suite.ctx, source), "I4: staking reward %s", id)
	}
	for denom := range state.drs {
		source := types.DenomRewardSource(denom)
		suite.Require().Equal(want[string(source)], suite.k.GetSourcePins(suite.ctx, source), "I4: denom reward %s", denom)
	}
}

func electorateSourceKeys(electorate *types.StakeElectorate) []string {
	if electorate == nil {
		return nil
	}

	keys := make([]string, 0, len(electorate.SrIds)+1)
	for _, id := range electorate.SrIds {
		keys = append(keys, string(types.StakingRewardSource(id)))
	}
	if electorate.DrDenom != "" {
		keys = append(keys, string(types.DenomRewardSource(electorate.DrDenom)))
	}

	return keys
}

// Business Logic §3: electorate changes accept only the DAO account. The admin
// signing directly and any other address are refused; nothing changes.
func (suite *IntegrationTestSuite) TestUpdateElectorate_DirectSignersRejected() {
	suite.stubRewards(defaultRewardsState())
	suite.createAdminStakeDao([]string{"1"}, stakeDenom)

	for _, signer := range []sdk.AccAddress{adminAddr, memberA, attackerAddr, daoAddress(2)} {
		suite.Require().ErrorIs(suite.updateElectorate(signer, 1, []string{"5"}, ""), types.ErrUnauthorizedSigner, signer.String())
	}

	suite.requireElectorate([]string{"1"}, stakeDenom)
	suite.requirePinInvariant([]uint64{1}, nil)
	suite.Require().False(suite.hasEvent("bze.dao.EventDaoElectorateUpdated"))
}

func (suite *IntegrationTestSuite) TestUpdateElectorate_UnknownDao() {
	suite.stubRewards(defaultRewardsState())
	suite.createAdminStakeDao([]string{"1"}, stakeDenom)

	suite.Require().ErrorIs(suite.updateElectorate(daoAddress(2), 2, []string{"5"}, ""), types.ErrDaoNotFound)
	suite.Require().ErrorIs(suite.updateElectorate(daoAddress(1), 0, []string{"5"}, ""), types.ErrDaoNotFound)
}

func (suite *IntegrationTestSuite) TestUpdateElectorate_InvalidAuthority() {
	suite.stubRewards(defaultRewardsState())
	suite.createAdminStakeDao([]string{"1"}, stakeDenom)

	_, err := suite.msgServer.UpdateElectorate(suite.ctx, types.NewMsgUpdateElectorate("bad", 1, &types.StakeElectorate{SrIds: []string{"5"}}))
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidAddress)
}

// D29: a static-list DAO has no stake sources to change, even through its admin.
func (suite *IntegrationTestSuite) TestUpdateElectorate_D29_StaticDaoRejected() {
	suite.stubRewards(defaultRewardsState())
	suite.createAdminDao()

	err := suite.adminExecute(adminAddr, 1, types.NewMsgUpdateElectorate(daoAddress(1).String(), 1, &types.StakeElectorate{SrIds: []string{"1"}}))
	suite.Require().ErrorIs(err, types.ErrInvalidDaoType)

	dao, _ := suite.k.GetDao(suite.ctx, 1)
	suite.Require().Nil(dao.Electorate)
	suite.Require().Equal([]string{memberA.String(), memberB.String()}, dao.Members)
	suite.requirePinInvariant([]uint64{1}, nil)
}

// The source rules on DAO 1 (external admin, staking reward 1 and the ubze
// Denom Reward): up to max_sr_per_dao unique existing staking rewards (D14),
// an existing Denom Reward, one staking denom (D29) and the min lock for every
// added source (D28). With an external admin the result may be empty or hold
// no stake (D11). A refused update changes nothing; I4 holds after each case.
func (suite *IntegrationTestSuite) TestUpdateElectorate_Rules() {
	for _, tc := range []struct {
		desc    string
		srIds   []string
		drDenom string
		wantErr string
	}{
		{desc: "replace every source", srIds: []string{"5", "6"}, drDenom: ""},
		{desc: "keep one, add one", srIds: []string{"1", "5"}, drDenom: stakeDenom},
		{desc: "denom reward only", srIds: nil, drDenom: stakeDenom},
		{desc: "staking rewards at the cap", srIds: []string{"1", "2", "5"}, drDenom: stakeDenom},
		{desc: "staking rewards above the cap", srIds: []string{"1", "2", "5", "6"}, wantErr: "at most 3 staking rewards"},
		{desc: "D14: duplicate staking reward", srIds: []string{"1", "5", "1"}, wantErr: "duplicate staking reward 1"},
		{desc: "unknown staking reward", srIds: []string{"1", "99"}, wantErr: "staking reward 99 not found"},
		{desc: "unknown denom reward", srIds: []string{"1"}, drDenom: "unone", wantErr: "denom reward unone not found"},
		{desc: "D28: added staking reward below the min lock", srIds: []string{"1", "4"}, wantErr: "staking reward 4 has a 4 day lock"},
		{desc: "D28: added denom reward below the min lock", drDenom: "ulow", wantErr: "denom reward ulow has a 4 day lock"},
		{desc: "D29: staking reward and denom reward with different denoms", srIds: []string{"1"}, drDenom: otherDenom, wantErr: "all sources must share one staking denom"},
		{desc: "D29: two staking rewards with different denoms", srIds: []string{"1", "3"}, wantErr: "all sources must share one staking denom"},
		{desc: "D29: switch to another denom entirely", srIds: []string{"3"}, drDenom: otherDenom},
		{desc: "D11: external admin empties the electorate", srIds: nil, drDenom: ""},
		{desc: "D11: external admin keeps sources with nothing staked", srIds: []string{"9"}, drDenom: "uzero"},
	} {
		suite.Run(tc.desc, func() {
			suite.SetupTest()
			suite.stubRewards(defaultRewardsState())
			params := types.DefaultParams()
			params.MaxSrPerDao = 3
			suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
			suite.createAdminStakeDao([]string{"1"}, stakeDenom)

			err := suite.updateElectorate(daoAddress(1), 1, tc.srIds, tc.drDenom)
			if tc.wantErr != "" {
				suite.Require().ErrorIs(err, types.ErrInvalidElectorate)
				suite.Require().ErrorContains(err, tc.wantErr)
				suite.requireElectorate([]string{"1"}, stakeDenom)
				suite.requirePinInvariant([]uint64{1}, nil)
				suite.Require().False(suite.hasEvent("bze.dao.EventDaoElectorateUpdated"))
				return
			}

			suite.Require().NoError(err)
			suite.requireElectorate(tc.srIds, tc.drDenom)
			suite.requirePinInvariant([]uint64{1}, nil)
			suite.Require().True(suite.hasEvent("bze.dao.EventDaoElectorateUpdated"))
		})
	}
}

// D28: raising min_electorate_lock_days never forces a DAO to drop a source.
// Staking reward 5 and the ubze Denom Reward (5 day locks) stay valid under a
// 10 day minimum; adding staking reward 1 (5 days) is refused, adding staking
// reward 6 (30 days) is accepted.
func (suite *IntegrationTestSuite) TestUpdateElectorate_D28_RaisedMinLockKeepsOldSources() {
	suite.stubRewards(defaultRewardsState())
	suite.createAdminStakeDao([]string{"5"}, stakeDenom)
	params := types.DefaultParams()
	params.MinElectorateLockDays = 10
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	suite.Require().NoError(suite.updateElectorate(daoAddress(1), 1, []string{"5"}, stakeDenom))
	suite.requireElectorate([]string{"5"}, stakeDenom)

	err := suite.updateElectorate(daoAddress(1), 1, []string{"5", "1"}, stakeDenom)
	suite.Require().ErrorIs(err, types.ErrInvalidElectorate)
	suite.Require().ErrorContains(err, "staking reward 1 has a 5 day lock, at least 10 days are required")
	suite.requireElectorate([]string{"5"}, stakeDenom)
	suite.requirePinInvariant([]uint64{1}, nil)

	suite.Require().NoError(suite.updateElectorate(daoAddress(1), 1, []string{"5", "6"}, stakeDenom))
	suite.requireElectorate([]string{"5", "6"}, stakeDenom)
	suite.requirePinInvariant([]uint64{1}, nil)

	// a source the DAO dropped is new again when it comes back
	suite.Require().NoError(suite.updateElectorate(daoAddress(1), 1, []string{"6"}, ""))
	err = suite.updateElectorate(daoAddress(1), 1, []string{"6"}, stakeDenom)
	suite.Require().ErrorContains(err, "denom reward ubze has a 5 day lock, at least 10 days are required")
	suite.requirePinInvariant([]uint64{1}, nil)
}

// Pins count per DAO: DAO 2 dropping staking reward 1 leaves DAO 1's pin.
func (suite *IntegrationTestSuite) TestUpdateElectorate_PinsCountEveryDao() {
	suite.stubRewards(defaultRewardsState())
	suite.createAdminStakeDao([]string{"1"}, stakeDenom)
	suite.expectCreation(2)
	msg := validStakeDaoMsg([]string{"1", "2"}, stakeDenom)
	msg.Admin = adminAddr.String()
	_, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.Require().NoError(suite.updateElectorate(daoAddress(2), 2, []string{"2", "5"}, ""))

	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, srSource))
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, drSource))
	suite.requirePinInvariant([]uint64{1, 2}, nil)
}

// The external admin replaces the sources through MsgAdminExecute, signed as
// the DAO; the event lists the new sources.
func (suite *IntegrationTestSuite) TestUpdateElectorate_ThroughAdminExecute() {
	suite.stubRewards(defaultRewardsState())
	suite.createAdminStakeDao([]string{"1"}, stakeDenom)

	electorate := &types.StakeElectorate{SrIds: []string{"5", "6"}, DrDenom: stakeDenom}
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, types.NewMsgUpdateElectorate(daoAddress(1).String(), 1, electorate)))

	suite.requireElectorate([]string{"5", "6"}, stakeDenom)
	suite.requirePinInvariant([]uint64{1}, nil)
	attrs := suite.lastEventAttrs(&types.EventDaoElectorateUpdated{})
	suite.Require().NotNil(attrs)
	suite.Require().Equal(`"1"`, attrs["id"])
	suite.Require().Equal(`["5","6"]`, attrs["sr_ids"])
	suite.Require().Equal(`"ubze"`, attrs["dr_denom"])
	suite.Require().True(suite.hasEvent("bze.dao.EventAdminExecuted"))
}

// An emptied electorate emits an event with no sources.
func (suite *IntegrationTestSuite) TestUpdateElectorate_EmptyEvent() {
	suite.stubRewards(defaultRewardsState())
	suite.createAdminStakeDao([]string{"1"}, stakeDenom)

	suite.Require().NoError(suite.updateElectorate(daoAddress(1), 1, nil, ""))

	suite.requireElectorate(nil, "")
	attrs := suite.lastEventAttrs(&types.EventDaoElectorateUpdated{})
	suite.Require().NotNil(attrs)
	suite.Require().Equal(`"1"`, attrs["id"])
	suite.Require().Equal(`""`, attrs["dr_denom"])
	suite.requirePinInvariant([]uint64{1}, nil)
}

// passedStakeProposal submits msgs to the self-owned stake DAO 1 (staking
// reward 1 + ubze Denom Reward, 150 staked) from memberA, who holds 100 in
// staking reward 1 and votes yes, then ends the voting period.
func (suite *IntegrationTestSuite) passedStakeProposal(msgs ...sdk.Msg) uint64 {
	s := suite.createStakeDao()
	s.joinSr("1", memberA, 100)
	suite.atBlock(10)
	id, err := suite.submit(memberA, 1, false, msgs...)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.vote(memberA, id, types.VoteOption_YES))
	suite.endVoting(id, 0)

	return id
}

// D11: a self-owned DAO cannot leave itself without voters. Emptying the
// sources, or keeping only sources with nothing staked, is refused; through a
// passed proposal the same update ends EXECUTION_FAILED and changes nothing.
func (suite *IntegrationTestSuite) TestUpdateElectorate_D11_SelfOwnedNeedsLiveElectorate() {
	suite.Run("direct", func() {
		suite.SetupTest()
		suite.createStakeDao()

		suite.Require().ErrorIs(suite.updateElectorate(daoAddress(1), 1, nil, ""), types.ErrElectorateNotLive)
		suite.Require().ErrorIs(suite.updateElectorate(daoAddress(1), 1, []string{"9"}, "uzero"), types.ErrElectorateNotLive)
		suite.requireElectorate([]string{"1"}, stakeDenom)
		suite.requirePinInvariant([]uint64{1}, nil)
	})

	suite.Run("through a passed proposal", func() {
		suite.SetupTest()
		id := suite.passedStakeProposal(types.NewMsgUpdateElectorate(daoAddress(1).String(), 1, &types.StakeElectorate{}))

		status, err := suite.settle(id)
		suite.Require().NoError(err)
		suite.Require().Equal(types.ProposalStatus_EXECUTION_FAILED, status)
		suite.requireExecuted("message failed")
		suite.requireElectorate([]string{"1"}, stakeDenom)
		suite.requirePinInvariant([]uint64{1}, []uint64{id})
	})
}

// A self-owned DAO replaces its sources through a passed proposal; the
// proposal's own pins are released when it settles.
func (suite *IntegrationTestSuite) TestUpdateElectorate_ThroughProposal() {
	electorate := &types.StakeElectorate{SrIds: []string{"5"}, DrDenom: stakeDenom}
	id := suite.passedStakeProposal(types.NewMsgUpdateElectorate(daoAddress(1).String(), 1, electorate))

	status, err := suite.settle(id)
	suite.Require().NoError(err)
	suite.Require().Equal(types.ProposalStatus_EXECUTED, status)
	suite.requireExecuted("")
	suite.requireElectorate([]string{"5"}, stakeDenom)
	suite.Require().Zero(suite.k.GetSourcePins(suite.ctx, srSource))
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, types.StakingRewardSource("5")))
	suite.Require().Equal(uint64(1), suite.k.GetSourcePins(suite.ctx, drSource))
	suite.requirePinInvariant([]uint64{1}, []uint64{id})
}

func (suite *IntegrationTestSuite) updateMembers(authority sdk.AccAddress, daoId uint64, add, remove []sdk.AccAddress) error {
	_, err := suite.msgServer.UpdateMembers(suite.ctx, types.NewMsgUpdateMembers(authority.String(), daoId, bech32s(add), bech32s(remove)))
	return err
}

func bech32s(addrs []sdk.AccAddress) []string {
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		out = append(out, addr.String())
	}

	return out
}

// requireMembers checks DAO 1's stored member list, in order.
func (suite *IntegrationTestSuite) requireMembers(members ...sdk.AccAddress) {
	dao, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().True(found)
	suite.Require().Equal(bech32s(members), dao.Members)
}

// Business Logic §3: member changes accept only the DAO account. The admin
// signing directly, a member and any other address are refused.
func (suite *IntegrationTestSuite) TestUpdateMembers_DirectSignersRejected() {
	suite.createAdminDao()

	for _, signer := range []sdk.AccAddress{adminAddr, memberA, attackerAddr, daoAddress(2)} {
		err := suite.updateMembers(signer, 1, []sdk.AccAddress{memberC}, nil)
		suite.Require().ErrorIs(err, types.ErrUnauthorizedSigner, signer.String())
	}

	suite.requireMembers(memberA, memberB)
	suite.Require().False(suite.hasEvent("bze.dao.EventDaoMembersUpdated"))
}

func (suite *IntegrationTestSuite) TestUpdateMembers_UnknownDao() {
	suite.createAdminDao()

	suite.Require().ErrorIs(suite.updateMembers(daoAddress(2), 2, []sdk.AccAddress{memberC}, nil), types.ErrDaoNotFound)
	suite.Require().ErrorIs(suite.updateMembers(daoAddress(1), 0, []sdk.AccAddress{memberC}, nil), types.ErrDaoNotFound)
}

func (suite *IntegrationTestSuite) TestUpdateMembers_InvalidRequests() {
	suite.createAdminDao()

	_, err := suite.msgServer.UpdateMembers(suite.ctx, types.NewMsgUpdateMembers("bad", 1, []string{memberC.String()}, nil))
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidAddress)

	_, err = suite.msgServer.UpdateMembers(suite.ctx, types.NewMsgUpdateMembers(daoAddress(1).String(), 1, nil, nil))
	suite.Require().ErrorIs(err, types.ErrInvalidMembers)
	suite.requireMembers(memberA, memberB)
}

// D29: a stake DAO has no member list, even through its admin.
func (suite *IntegrationTestSuite) TestUpdateMembers_D29_StakeDaoRejected() {
	suite.stubRewards(defaultRewardsState())
	suite.createAdminStakeDao([]string{"1"}, stakeDenom)

	err := suite.adminExecute(adminAddr, 1, types.NewMsgUpdateMembers(daoAddress(1).String(), 1, []string{memberA.String()}, nil))
	suite.Require().ErrorIs(err, types.ErrInvalidDaoType)

	dao, _ := suite.k.GetDao(suite.ctx, 1)
	suite.Require().Empty(dao.Members)
	suite.requireElectorate([]string{"1"}, stakeDenom)
}

// The member rules on DAO 1 (external admin, members A and B, cap 3): every
// added address is new, every removed one a member, no address in both lists,
// and the result holds 1 to max_static_members members. Removed members leave
// the list in place, added ones are appended. A refused update changes nothing.
func (suite *IntegrationTestSuite) TestUpdateMembers_Rules() {
	for _, tc := range []struct {
		desc    string
		add     []string
		remove  []string
		want    []sdk.AccAddress
		wantErr string
	}{
		{desc: "add one", add: []string{memberC.String()}, want: []sdk.AccAddress{memberA, memberB, memberC}},
		{desc: "remove one", remove: []string{memberA.String()}, want: []sdk.AccAddress{memberB}},
		{desc: "add and remove", add: []string{memberC.String()}, remove: []string{memberA.String()}, want: []sdk.AccAddress{memberB, memberC}},
		{desc: "replace every member", add: []string{memberC.String()}, remove: []string{memberA.String(), memberB.String()}, want: []sdk.AccAddress{memberC}},
		{desc: "added address stored canonical", add: []string{strings.ToUpper(memberC.String())}, want: []sdk.AccAddress{memberA, memberB, memberC}},
		{desc: "removed address in another case", remove: []string{strings.ToUpper(memberB.String())}, want: []sdk.AccAddress{memberA}},
		{desc: "remove every member", remove: []string{memberA.String(), memberB.String()}, wantErr: "needs at least one member"},
		{desc: "above the cap", add: []string{memberC.String(), attackerAddr.String()}, wantErr: "at most 3 members"},
		{desc: "add an existing member", add: []string{memberC.String(), memberA.String()}, wantErr: "is already a member"},
		{desc: "add an existing member in another case", add: []string{strings.ToUpper(memberB.String())}, wantErr: "is already a member"},
		{desc: "remove a non-member", remove: []string{memberA.String(), memberC.String()}, wantErr: "is not a member"},
		{desc: "duplicate added address", add: []string{memberC.String(), memberC.String()}, wantErr: "duplicate add address"},
		{desc: "duplicate removed address", remove: []string{memberA.String(), memberA.String()}, wantErr: "duplicate remove address"},
		{desc: "address both added and removed", add: []string{memberC.String()}, remove: []string{memberC.String()}, wantErr: "is both added and removed"},
	} {
		suite.Run(tc.desc, func() {
			suite.SetupTest()
			params := types.DefaultParams()
			params.MaxStaticMembers = 3
			suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
			suite.createAdminDao()

			_, err := suite.msgServer.UpdateMembers(suite.ctx, types.NewMsgUpdateMembers(daoAddress(1).String(), 1, tc.add, tc.remove))
			if tc.wantErr != "" {
				suite.Require().ErrorIs(err, types.ErrInvalidMembers)
				suite.Require().ErrorContains(err, tc.wantErr)
				suite.requireMembers(memberA, memberB)
				suite.Require().False(suite.hasEvent("bze.dao.EventDaoMembersUpdated"))
				return
			}

			suite.Require().NoError(err)
			suite.requireMembers(tc.want...)
			suite.Require().True(suite.hasEvent("bze.dao.EventDaoMembersUpdated"))
		})
	}
}

// The default cap is 128 members.
func (suite *IntegrationTestSuite) TestUpdateMembers_DefaultMemberCap() {
	suite.createAdminDao()

	added := make([]sdk.AccAddress, 127)
	for i := range added {
		added[i] = memberAddress(i)
	}

	suite.Require().ErrorIs(suite.updateMembers(daoAddress(1), 1, added, nil), types.ErrInvalidMembers)
	suite.Require().NoError(suite.updateMembers(daoAddress(1), 1, added[:126], nil))

	dao, _ := suite.k.GetDao(suite.ctx, 1)
	suite.Require().Len(dao.Members, 128)
}

// D11: a self-owned DAO cannot remove its last member.
func (suite *IntegrationTestSuite) TestUpdateMembers_D11_SelfOwnedKeepsAMember() {
	suite.createStaticDao(noConfigChange)

	err := suite.updateMembers(daoAddress(1), 1, nil, []sdk.AccAddress{memberA, memberB})
	suite.Require().ErrorIs(err, types.ErrInvalidMembers)
	suite.requireMembers(memberA, memberB)

	suite.Require().NoError(suite.updateMembers(daoAddress(1), 1, nil, []sdk.AccAddress{memberA}))
	suite.requireMembers(memberB)
}

// The external admin changes the members through MsgAdminExecute, signed as
// the DAO; the event lists the changed addresses.
func (suite *IntegrationTestSuite) TestUpdateMembers_ThroughAdminExecute() {
	suite.createAdminDao()

	msg := types.NewMsgUpdateMembers(daoAddress(1).String(), 1, []string{memberC.String()}, []string{memberA.String()})
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, msg))

	suite.requireMembers(memberB, memberC)
	attrs := suite.lastEventAttrs(&types.EventDaoMembersUpdated{})
	suite.Require().NotNil(attrs)
	suite.Require().Equal(`"1"`, attrs["id"])
	suite.Require().Equal(`["`+memberC.String()+`"]`, attrs["added"])
	suite.Require().Equal(`["`+memberA.String()+`"]`, attrs["removed"])
	suite.Require().True(suite.hasEvent("bze.dao.EventAdminExecuted"))
}
