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
