package keeper_test

import (
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"go.uber.org/mock/gomock"

	"github.com/bze-alphateam/bze/x/dao/types"
)

const (
	burnerFeeCollector = "txfeecollector_burner"
	cpFeeCollector     = "txfeecollector_cp"
)

var (
	creatorAddr = sdk.AccAddress("dao_creator_________")
	memberA     = sdk.AccAddress("dao_member_a________")
	memberB     = sdk.AccAddress("dao_member_b________")
	memberC     = sdk.AccAddress("dao_member_c________")
	adminAddr   = sdk.AccAddress("dao_admin___________")
)

func validConfig() types.DaoConfig {
	return types.DaoConfig{
		MinQuorum:         dec("0.4"),
		Threshold:         dec("0.5"),
		VotingPeriod:      72 * time.Hour,
		AllowExpedite:     true,
		ExpeditedPeriod:   24 * time.Hour,
		ExpediteQuorum:    dec("0.6"),
		ExpediteThreshold: dec("0.66"),
	}
}

func validCreateDaoMsg() *types.MsgCreateDao {
	return &types.MsgCreateDao{
		Creator:     creatorAddr.String(),
		Type:        types.DaoType_DAO_TYPE_STATIC_LIST,
		Name:        "BZE builders",
		Description: "funds BZE tooling",
		Config:      validConfig(),
		Members:     []string{memberA.String(), memberB.String()},
	}
}

func dec(s string) math.LegacyDec {
	return math.LegacyMustNewDecFromStr(s)
}

func daoAddress(id uint64) sdk.AccAddress {
	return address.Module("dao", sdk.Uint64ToBigEndian(id))
}

// expectNewDaoAccount expects the creation of a fresh account at the DAO address.
func (suite *IntegrationTestSuite) expectNewDaoAccount(id uint64) {
	addr := daoAddress(id)
	suite.acc.EXPECT().GetAccount(gomock.Any(), addr).Return(nil)
	suite.acc.EXPECT().NewAccountWithAddress(gomock.Any(), addr).Return(authtypes.NewBaseAccountWithAddress(addr))
	suite.acc.EXPECT().SetAccount(gomock.Any(), authtypes.NewBaseAccountWithAddress(addr))
}

// expectFee expects the default creation fee to be paid to the given module.
func (suite *IntegrationTestSuite) expectFee(destModule string) {
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), creatorAddr, destModule, sdk.NewCoins(types.DefaultDaoCreationFee)).
		Return(nil)
}

// expectCreation expects a successful creation of the DAO with the given id.
func (suite *IntegrationTestSuite) expectCreation(id uint64) {
	suite.expectNewDaoAccount(id)
	suite.expectFee(burnerFeeCollector)
}

type configCase struct {
	desc    string
	mutate  func(c *types.DaoConfig)
	wantErr error
}

// configCases are every bound of a DaoConfig on both sides, under the default
// params, applied to validConfig. MsgCreateDao and MsgUpdateDaoConfig share them.
func configCases() []configCase {
	return []configCase{
		{desc: "zero min_quorum", mutate: func(c *types.DaoConfig) { c.MinQuorum = dec("0") }, wantErr: types.ErrInvalidConfig},
		{desc: "negative min_quorum", mutate: func(c *types.DaoConfig) { c.MinQuorum = dec("-0.1") }, wantErr: types.ErrInvalidConfig},
		{desc: "smallest min_quorum", mutate: func(c *types.DaoConfig) { c.MinQuorum = dec("0.000000000000000001") }},
		{desc: "min_quorum above 1", mutate: func(c *types.DaoConfig) { c.MinQuorum = dec("1.000000000000000001") }, wantErr: types.ErrInvalidConfig},
		{desc: "unset min_quorum", mutate: func(c *types.DaoConfig) { c.MinQuorum = math.LegacyDec{} }, wantErr: types.ErrInvalidConfig},
		{desc: "unset expedite_threshold", mutate: func(c *types.DaoConfig) { c.ExpediteThreshold = math.LegacyDec{} }, wantErr: types.ErrInvalidConfig},
		{desc: "zero threshold", mutate: func(c *types.DaoConfig) { c.Threshold = dec("0") }, wantErr: types.ErrInvalidConfig},
		{desc: "threshold above 1", mutate: func(c *types.DaoConfig) { c.Threshold = dec("1.1") }, wantErr: types.ErrInvalidConfig},
		{desc: "all ratios at 1", mutate: func(c *types.DaoConfig) {
			one := math.LegacyOneDec()
			c.MinQuorum, c.Threshold, c.ExpediteQuorum, c.ExpediteThreshold = one, one, one, one
		}},
		{desc: "zero expedite_quorum", mutate: func(c *types.DaoConfig) { c.ExpediteQuorum = dec("0") }, wantErr: types.ErrInvalidConfig},
		{desc: "expedite_quorum above 1", mutate: func(c *types.DaoConfig) { c.ExpediteQuorum = dec("1.5") }, wantErr: types.ErrInvalidConfig},
		{desc: "expedite_quorum below min_quorum", mutate: func(c *types.DaoConfig) { c.ExpediteQuorum = dec("0.39") }, wantErr: types.ErrInvalidConfig},
		{desc: "expedite_quorum equal to min_quorum", mutate: func(c *types.DaoConfig) { c.ExpediteQuorum = dec("0.4") }},
		{desc: "zero expedite_threshold", mutate: func(c *types.DaoConfig) { c.ExpediteThreshold = dec("0") }, wantErr: types.ErrInvalidConfig},
		{desc: "expedite_threshold above 1", mutate: func(c *types.DaoConfig) { c.ExpediteThreshold = dec("2") }, wantErr: types.ErrInvalidConfig},
		{desc: "expedite_threshold below threshold", mutate: func(c *types.DaoConfig) { c.ExpediteThreshold = dec("0.49") }, wantErr: types.ErrInvalidConfig},
		{desc: "expedite_threshold equal to threshold", mutate: func(c *types.DaoConfig) { c.ExpediteThreshold = dec("0.5") }},
		{desc: "invalid expedite fields checked when expedite is off", mutate: func(c *types.DaoConfig) {
			c.AllowExpedite = false
			c.ExpediteQuorum = dec("0")
		}, wantErr: types.ErrInvalidConfig},

		{desc: "voting period at min", mutate: func(c *types.DaoConfig) { c.VotingPeriod, c.ExpeditedPeriod = time.Hour, time.Hour }},
		{desc: "voting period below min", mutate: func(c *types.DaoConfig) {
			c.VotingPeriod, c.ExpeditedPeriod = time.Hour-time.Nanosecond, time.Hour-time.Nanosecond
		}, wantErr: types.ErrInvalidConfig},
		{desc: "voting period at max", mutate: func(c *types.DaoConfig) { c.VotingPeriod = 90 * 24 * time.Hour }},
		{desc: "voting period above max", mutate: func(c *types.DaoConfig) { c.VotingPeriod = 90*24*time.Hour + time.Nanosecond }, wantErr: types.ErrInvalidConfig},
		{desc: "expedited period below min", mutate: func(c *types.DaoConfig) { c.ExpeditedPeriod = time.Hour - time.Nanosecond }, wantErr: types.ErrInvalidConfig},
		{desc: "expedited period below min when expedite is off", mutate: func(c *types.DaoConfig) {
			c.AllowExpedite = false
			c.ExpeditedPeriod = 0
		}, wantErr: types.ErrInvalidConfig},
		{desc: "expedited period equal to voting period", mutate: func(c *types.DaoConfig) { c.ExpeditedPeriod = c.VotingPeriod }},
		{desc: "expedited period longer than voting period", mutate: func(c *types.DaoConfig) {
			c.ExpeditedPeriod = c.VotingPeriod + time.Nanosecond
		}, wantErr: types.ErrInvalidConfig},

		{desc: "zero proposal cost and deposit", mutate: func(c *types.DaoConfig) {
			zero := sdk.NewInt64Coin("ubze", 0)
			c.ProposalCost, c.ProposalDeposit = &zero, &zero
		}},
		{desc: "positive proposal cost and deposit", mutate: func(c *types.DaoConfig) {
			cost, deposit := sdk.NewInt64Coin("ubze", 1), sdk.NewInt64Coin("ubze", 100)
			c.ProposalCost, c.ProposalDeposit = &cost, &deposit
		}},
		{desc: "negative proposal cost", mutate: func(c *types.DaoConfig) {
			c.ProposalCost = &sdk.Coin{Denom: "ubze", Amount: math.NewInt(-1)}
		}, wantErr: types.ErrInvalidConfig},
		{desc: "invalid proposal cost denom", mutate: func(c *types.DaoConfig) {
			c.ProposalCost = &sdk.Coin{Denom: "1", Amount: math.NewInt(1)}
		}, wantErr: types.ErrInvalidConfig},
		{desc: "negative proposal deposit", mutate: func(c *types.DaoConfig) {
			c.ProposalDeposit = &sdk.Coin{Denom: "ubze", Amount: math.NewInt(-1)}
		}, wantErr: types.ErrInvalidConfig},
		{desc: "proposal deposit without amount", mutate: func(c *types.DaoConfig) {
			c.ProposalDeposit = &sdk.Coin{Denom: "ubze"}
		}, wantErr: types.ErrInvalidConfig},
		{desc: "unknown deposit refund policy", mutate: func(c *types.DaoConfig) { c.DepositRefundPolicy = 3 }, wantErr: types.ErrInvalidConfig},
		{desc: "unknown cost destination", mutate: func(c *types.DaoConfig) { c.CostDestination = 2 }, wantErr: types.ErrInvalidConfig},
		{desc: "unknown kept deposit destination", mutate: func(c *types.DaoConfig) { c.KeptDepositDestination = 2 }, wantErr: types.ErrInvalidConfig},
		{desc: "non-default policy and destinations", mutate: func(c *types.DaoConfig) {
			c.DepositRefundPolicy = types.DepositRefundPolicy_NEVER
			c.CostDestination = types.FundsDestination_TREASURY
			c.KeptDepositDestination = types.FundsDestination_TREASURY
		}},
	}
}

// Every bound of the config, metadata and addresses, on both sides. A rejected
// message touches neither the accounts, the bank (gomock fails on any
// unexpected call) nor the store.
func (suite *IntegrationTestSuite) TestCreateDao_ValidationMatrix() {
	type createCase struct {
		desc    string
		mutate  func(m *types.MsgCreateDao)
		wantErr error
	}
	cases := []createCase{
		{desc: "valid", mutate: func(m *types.MsgCreateDao) {}},
		{desc: "invalid creator", mutate: func(m *types.MsgCreateDao) { m.Creator = "bad" }, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "empty creator", mutate: func(m *types.MsgCreateDao) { m.Creator = "" }, wantErr: sdkerrors.ErrInvalidAddress},
		{desc: "invalid admin", mutate: func(m *types.MsgCreateDao) { m.Admin = "bad" }, wantErr: sdkerrors.ErrInvalidAddress},

		{desc: "empty name", mutate: func(m *types.MsgCreateDao) { m.Name = "" }, wantErr: types.ErrInvalidMetadata},
		{desc: "1 byte name", mutate: func(m *types.MsgCreateDao) { m.Name = "x" }},
		{desc: "64 byte name", mutate: func(m *types.MsgCreateDao) { m.Name = strings.Repeat("n", 64) }},
		{desc: "65 byte name", mutate: func(m *types.MsgCreateDao) { m.Name = strings.Repeat("n", 65) }, wantErr: types.ErrInvalidMetadata},
		{desc: "empty description", mutate: func(m *types.MsgCreateDao) { m.Description = "" }},
		{desc: "2000 byte description", mutate: func(m *types.MsgCreateDao) { m.Description = strings.Repeat("d", 2000) }},
		{desc: "2001 byte description", mutate: func(m *types.MsgCreateDao) { m.Description = strings.Repeat("d", 2001) }, wantErr: types.ErrInvalidMetadata},

		{desc: "unknown type", mutate: func(m *types.MsgCreateDao) { m.Type = 2 }, wantErr: types.ErrInvalidDaoType},
	}
	for _, cc := range configCases() {
		cases = append(cases, createCase{desc: cc.desc, mutate: func(m *types.MsgCreateDao) { cc.mutate(&m.Config) }, wantErr: cc.wantErr})
	}

	for _, tc := range cases {
		suite.Run(tc.desc, func() {
			suite.SetupTest()
			msg := validCreateDaoMsg()
			tc.mutate(msg)

			if tc.wantErr == nil {
				suite.expectCreation(1)
			}

			res, err := suite.msgServer.CreateDao(suite.ctx, msg)
			if tc.wantErr != nil {
				suite.Require().ErrorIs(err, tc.wantErr)
				suite.Require().Nil(res)
				suite.Require().Zero(suite.k.GetDaoCounter(suite.ctx))
				_, found := suite.k.GetDao(suite.ctx, 1)
				suite.Require().False(found)
				return
			}

			suite.Require().NoError(err)
			dao, found := suite.k.GetDao(suite.ctx, res.Id)
			suite.Require().True(found)
			suite.Require().Equal(msg.Config, dao.Config)
			suite.Require().Equal(msg.Name, dao.Name)
			suite.Require().Equal(msg.Description, dao.Description)
		})
	}
}

// The period bounds follow the params, not constants.
func (suite *IntegrationTestSuite) TestCreateDao_PeriodBoundsFollowParams() {
	params := types.DefaultParams()
	params.MinVotingPeriod = 48 * time.Hour
	params.MaxVotingPeriod = 96 * time.Hour
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	// default config: expedited period 24h < new min 48h
	_, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().ErrorIs(err, types.ErrInvalidConfig)
	suite.Require().ErrorContains(err, "expedited_period")

	msg := validCreateDaoMsg()
	msg.Config.ExpeditedPeriod = 48 * time.Hour
	msg.Config.VotingPeriod = 96*time.Hour + time.Nanosecond
	_, err = suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidConfig)
	suite.Require().ErrorContains(err, "voting_period")

	msg.Config.VotingPeriod = 96 * time.Hour
	suite.expectCreation(1)
	_, err = suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)
}

// The stored record, the response and the event of a successful creation.
func (suite *IntegrationTestSuite) TestCreateDao_StoresDaoAndEmitsEvent() {
	blockTime := time.Unix(1_800_000_000, 0)
	suite.ctx = suite.ctx.WithBlockTime(blockTime).WithEventManager(sdk.NewEventManager())
	suite.expectCreation(1)
	msg := validCreateDaoMsg()

	res, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)

	daoAddr := daoAddress(1)
	suite.Require().Equal(&types.MsgCreateDaoResponse{Id: 1, Address: daoAddr.String()}, res)

	dao, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().True(found)
	suite.Require().Equal(types.Dao{
		Id:          1,
		Address:     daoAddr.String(),
		Admin:       daoAddr.String(),
		Type:        types.DaoType_DAO_TYPE_STATIC_LIST,
		Name:        msg.Name,
		Description: msg.Description,
		Config:      msg.Config,
		Members:     msg.Members,
		CreatedAt:   blockTime.Unix(),
	}, dao)

	id, found := suite.k.GetDaoIdByAddress(suite.ctx, daoAddr)
	suite.Require().True(found)
	suite.Require().Equal(uint64(1), id)

	var event *sdk.Event
	for _, e := range suite.ctx.EventManager().Events() {
		if e.Type == "bze.dao.EventDaoCreated" {
			event = &e
		}
	}
	suite.Require().NotNil(event)
	attrs := map[string]string{}
	for _, a := range event.Attributes {
		attrs[a.Key] = a.Value
	}
	suite.Require().Equal(`"1"`, attrs["id"])
	suite.Require().Equal(`"`+daoAddr.String()+`"`, attrs["address"])
	suite.Require().Equal(`"`+daoAddr.String()+`"`, attrs["admin"])
	suite.Require().Equal(`"DAO_TYPE_STATIC_LIST"`, attrs["type"])
}

// The DAO address is address.Module("dao", uint64-big-endian(id)) and differs per id.
func (suite *IntegrationTestSuite) TestCreateDao_DerivedAddressFormula() {
	suite.expectCreation(1)
	suite.expectCreation(2)

	first, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)
	second, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)

	suite.Require().Equal(sdk.AccAddress(address.Module("dao", []byte{0, 0, 0, 0, 0, 0, 0, 1})).String(), first.Address)
	suite.Require().Equal(sdk.AccAddress(address.Module("dao", []byte{0, 0, 0, 0, 0, 0, 0, 2})).String(), second.Address)
	suite.Require().Equal(types.DaoAddress(1).String(), first.Address)
	suite.Require().NotEqual(first.Address, second.Address)
	suite.Require().NotEqual(authtypes.NewModuleAddress(types.ModuleName).String(), first.Address)
}

// An account with a public key at the derived address aborts the creation
// before any fee is charged.
func (suite *IntegrationTestSuite) TestCreateDao_AccountWithPubKeyAborts() {
	addr := daoAddress(1)
	acc := authtypes.NewBaseAccount(addr, secp256k1.GenPrivKey().PubKey(), 0, 0)
	suite.acc.EXPECT().GetAccount(gomock.Any(), addr).Return(acc)

	res, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().ErrorIs(err, types.ErrDaoAccountExists)
	suite.Require().Nil(res)
	suite.Require().Zero(suite.k.GetDaoCounter(suite.ctx))
	_, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().False(found)
}

// A module account at the derived address aborts. It can't appear at a
// dao-derived address; the check is a safety net.
func (suite *IntegrationTestSuite) TestCreateDao_ModuleAccountAborts() {
	addr := daoAddress(1)
	acc := authtypes.NewModuleAccount(authtypes.NewBaseAccountWithAddress(addr), "squatter")
	suite.acc.EXPECT().GetAccount(gomock.Any(), addr).Return(acc)

	_, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().ErrorIs(err, types.ErrDaoAccountExists)
	suite.Require().Zero(suite.k.GetDaoCounter(suite.ctx))
}

// Anyone can create a vesting account at the predictable address of the next
// DAO. It must not block the creation: the account is replaced with a plain
// BaseAccount keeping its number and sequence, and its coins stay at the
// address as the DAO's treasury.
func (suite *IntegrationTestSuite) TestCreateDao_VestingAccountReplaced() {
	addr := daoAddress(1)
	coins := sdk.NewCoins(sdk.NewInt64Coin("ubze", 1))
	base := authtypes.NewBaseAccount(addr, nil, 42, 3)
	continuous, err := vestingtypes.NewContinuousVestingAccount(base, coins, 0, 1<<40)
	suite.Require().NoError(err)
	locked, err := vestingtypes.NewPermanentLockedAccount(authtypes.NewBaseAccount(addr, nil, 42, 3), coins)
	suite.Require().NoError(err)

	for name, acc := range map[string]sdk.AccountI{"continuous": continuous, "permanent locked": locked} {
		suite.Run(name, func() {
			suite.SetupTest()
			suite.acc.EXPECT().GetAccount(gomock.Any(), addr).Return(acc)
			suite.acc.EXPECT().SetAccount(gomock.Any(), authtypes.NewBaseAccount(addr, nil, 42, 3))
			suite.expectFee(burnerFeeCollector)

			res, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
			suite.Require().NoError(err)
			suite.Require().Equal(addr.String(), res.Address)
			dao, found := suite.k.GetDao(suite.ctx, 1)
			suite.Require().True(found)
			suite.Require().Equal(addr.String(), dao.Address)
			suite.Require().Equal(uint64(1), suite.k.GetDaoCounter(suite.ctx))
		})
	}
}

// A vesting account with a public key still aborts.
func (suite *IntegrationTestSuite) TestCreateDao_VestingAccountWithPubKeyAborts() {
	addr := daoAddress(1)
	base := authtypes.NewBaseAccount(addr, secp256k1.GenPrivKey().PubKey(), 0, 0)
	acc := &vestingtypes.PermanentLockedAccount{BaseVestingAccount: &vestingtypes.BaseVestingAccount{BaseAccount: base}}
	suite.acc.EXPECT().GetAccount(gomock.Any(), addr).Return(acc)

	_, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().ErrorIs(err, types.ErrDaoAccountExists)
	suite.Require().Zero(suite.k.GetDaoCounter(suite.ctx))
}

// A BaseAccount without a public key (e.g. created by funds sent ahead of the
// DAO) is reused as is: no new account is created.
func (suite *IntegrationTestSuite) TestCreateDao_AccountWithoutPubKeyReused() {
	addr := daoAddress(1)
	suite.acc.EXPECT().GetAccount(gomock.Any(), addr).Return(authtypes.NewBaseAccountWithAddress(addr))
	suite.expectFee(burnerFeeCollector)

	res, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)
	suite.Require().Equal(addr.String(), res.Address)
	_, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().True(found)
}

func (suite *IntegrationTestSuite) TestCreateDao_FeeToBurner() {
	suite.expectNewDaoAccount(1)
	suite.expectFee(burnerFeeCollector)

	_, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)
}

func (suite *IntegrationTestSuite) TestCreateDao_FeeToCommunityPool() {
	params := types.DefaultParams()
	params.DaoCreationFeeDestination = types.FeeDestinationCommunityPool
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
	suite.expectNewDaoAccount(1)
	suite.expectFee(cpFeeCollector)

	_, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)
}

// A zero fee is not sent at all: the bank mock fails on any call.
func (suite *IntegrationTestSuite) TestCreateDao_ZeroFeeSkipsSend() {
	params := types.DefaultParams()
	params.DaoCreationFee = sdk.NewInt64Coin("ubze", 0)
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
	suite.expectNewDaoAccount(1)

	res, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1), res.Id)
}

// A creator who can't pay gets an error and nothing is stored or created.
func (suite *IntegrationTestSuite) TestCreateDao_InsufficientFunds() {
	addr := daoAddress(1)
	suite.acc.EXPECT().GetAccount(gomock.Any(), addr).Return(nil)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), creatorAddr, burnerFeeCollector, sdk.NewCoins(types.DefaultDaoCreationFee)).
		Return(sdkerrors.ErrInsufficientFunds)

	res, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
	suite.Require().Nil(res)
	suite.Require().Zero(suite.k.GetDaoCounter(suite.ctx))
	_, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().False(found)
	_, found = suite.k.GetDaoIdByAddress(suite.ctx, addr)
	suite.Require().False(found)
	suite.Require().False(suite.k.HasDaoAdminIndex(suite.ctx, addr, 1))
}

// I5: ids start at 1 and increase; a failed creation does not burn an id.
func (suite *IntegrationTestSuite) TestCreateDao_SequentialIds() {
	suite.expectCreation(1)
	first, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)

	bad := validCreateDaoMsg()
	bad.Name = ""
	_, err = suite.msgServer.CreateDao(suite.ctx, bad)
	suite.Require().Error(err)

	suite.expectCreation(2)
	second, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)

	suite.Require().Equal(uint64(1), first.Id)
	suite.Require().Equal(uint64(2), second.Id)
	suite.Require().Equal(uint64(2), suite.k.GetDaoCounter(suite.ctx))

	for _, res := range []*types.MsgCreateDaoResponse{first, second} {
		id, found := suite.k.GetDaoIdByAddress(suite.ctx, sdk.MustAccAddressFromBech32(res.Address))
		suite.Require().True(found)
		suite.Require().Equal(res.Id, id)
	}
}

// I7: without an admin the DAO owns itself and the admin index points at the DAO address.
func (suite *IntegrationTestSuite) TestCreateDao_DefaultAdminIsDaoAddress() {
	suite.expectCreation(1)

	res, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)

	dao, _ := suite.k.GetDao(suite.ctx, res.Id)
	suite.Require().Equal(res.Address, dao.Admin)
	suite.Require().Empty(dao.PendingAdmin)
	suite.Require().True(suite.k.HasDaoAdminIndex(suite.ctx, daoAddress(1), 1))
	suite.Require().False(suite.k.HasDaoAdminIndex(suite.ctx, creatorAddr, 1))
}

// I7: an explicit admin (a person or another DAO) is stored in canonical form and indexed.
func (suite *IntegrationTestSuite) TestCreateDao_ExplicitAdmin() {
	suite.expectCreation(1)
	suite.expectCreation(2)

	msg := validCreateDaoMsg()
	msg.Admin = strings.ToUpper(adminAddr.String())
	first, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)

	// a sub-DAO administered by the first DAO
	sub := validCreateDaoMsg()
	sub.Admin = first.Address
	second, err := suite.msgServer.CreateDao(suite.ctx, sub)
	suite.Require().NoError(err)

	dao, _ := suite.k.GetDao(suite.ctx, first.Id)
	suite.Require().Equal(adminAddr.String(), dao.Admin)
	suite.Require().True(suite.k.HasDaoAdminIndex(suite.ctx, adminAddr, first.Id))
	suite.Require().False(suite.k.HasDaoAdminIndex(suite.ctx, daoAddress(1), first.Id))

	subDao, _ := suite.k.GetDao(suite.ctx, second.Id)
	suite.Require().Equal(first.Address, subDao.Admin)
	suite.Require().True(suite.k.HasDaoAdminIndex(suite.ctx, daoAddress(1), second.Id))
	suite.Require().False(suite.k.HasDaoAdminIndex(suite.ctx, adminAddr, second.Id))
}

// A static-list DAO cannot list its own address as a member.
func (suite *IntegrationTestSuite) TestCreateDao_OwnAddressAsMemberRejected() {
	msg := validCreateDaoMsg()
	msg.Members = append(msg.Members, daoAddress(1).String())
	_, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidMembers)
	suite.Require().ErrorContains(err, "cannot be a member of itself")

	_, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().False(found)
}

