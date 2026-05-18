package keeper_test

import (
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/mock/gomock"

	burnertypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/daodao/types"
)

// TestCreateDao_HappyPath checks the basic create flow with no fee, no
// parent, default admin (= creator), default STATIC voting (creator-only).
func (suite *IntegrationTestSuite) TestCreateDao_HappyPath() {
	creator := freshAddr()
	suite.expectAccountCreated(1)

	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("alpha"),
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1), resp.DaoId)
	suite.Require().Equal(types.DaoAccountAddress(1).String(), resp.AccountAddress)

	dao, ok := suite.k.GetDao(suite.ctx, resp.DaoId)
	suite.Require().True(ok)
	suite.Require().Equal(creator, dao.Creator)
	suite.Require().Equal(creator, dao.Admin)
	suite.Require().Empty(dao.PendingAdmin)
	suite.Require().Equal(uint64(0), dao.ParentDaoId)
	suite.Require().Equal("alpha", dao.Metadata.Name)
	suite.Require().Equal(types.VotingBackendType_VOTING_BACKEND_STATIC, dao.VotingBackend)
	suite.Require().Empty(dao.RewardId)
	suite.Require().Equal(resp.AccountAddress, dao.AccountAddress)

	suite.Require().Equal(uint64(2), suite.k.GetDaoIDCounter(suite.ctx))
}

// TestCreateDao_ExplicitAdmin ensures msg.Admin is honored when non-empty.
func (suite *IntegrationTestSuite) TestCreateDao_ExplicitAdmin() {
	creator := freshAddr()
	admin := freshAddr()
	suite.expectAccountCreated(1)

	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("beta"),
		Admin:        admin,
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)
	dao, _ := suite.k.GetDao(suite.ctx, resp.DaoId)
	suite.Require().Equal(admin, dao.Admin)
	suite.Require().Equal(creator, dao.Creator)
}

// TestCreateDao_FeeBurner verifies fee deduction and burner routing.
func (suite *IntegrationTestSuite) TestCreateDao_FeeBurner() {
	creator := suite.mustAcc(freshAddr())
	feeAmt := math.NewInt(1_000_000)
	feeCoin := sdk.NewCoin("ubze", feeAmt)
	feeCoins := sdk.NewCoins(feeCoin)

	params := types.DefaultParams()
	params.DaoCreationFee = feeCoin
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	suite.bank.EXPECT().
		SpendableCoins(gomock.Any(), creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", feeAmt.MulRaw(2)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), creator, burnertypes.ModuleName, feeCoins).
		Return(nil).
		Times(1)
	suite.expectAccountCreated(1)

	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator.String(),
		Metadata:     sampleMetadata("gamma"),
		VotingConfig: staticConfig(creator.String()),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)
}

// TestCreateDao_FeeCommunityPool verifies distr routing when destination is
// configured to community_pool.
func (suite *IntegrationTestSuite) TestCreateDao_FeeCommunityPool() {
	creator := suite.mustAcc(freshAddr())
	feeAmt := math.NewInt(1_000_000)
	feeCoin := sdk.NewCoin("ubze", feeAmt)
	feeCoins := sdk.NewCoins(feeCoin)

	params := types.DefaultParams()
	params.DaoCreationFee = feeCoin
	params.DaoCreationFeeDestination = types.FeeDestinationCommunityPool
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	suite.bank.EXPECT().
		SpendableCoins(gomock.Any(), creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", feeAmt.MulRaw(2)))).
		Times(1)
	suite.distr.EXPECT().
		FundCommunityPool(gomock.Any(), feeCoins, creator).
		Return(nil).
		Times(1)
	suite.expectAccountCreated(1)

	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator.String(),
		Metadata:     sampleMetadata("delta"),
		VotingConfig: staticConfig(creator.String()),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)
}

// TestCreateDao_FeeInsufficient rejects when creator can't cover the fee.
func (suite *IntegrationTestSuite) TestCreateDao_FeeInsufficient() {
	creator := suite.mustAcc(freshAddr())
	feeAmt := math.NewInt(1_000_000)

	params := types.DefaultParams()
	params.DaoCreationFee = sdk.NewCoin("ubze", feeAmt)
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	suite.bank.EXPECT().
		SpendableCoins(gomock.Any(), creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10)))).
		Times(1)

	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator.String(),
		Metadata:     sampleMetadata("epsilon"),
		VotingConfig: staticConfig(creator.String()),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().Error(err)
	suite.Require().True(strings.Contains(err.Error(), "insufficient"), "got %s", err.Error())

	suite.Require().Equal(uint64(1), suite.k.GetDaoIDCounter(suite.ctx))
	_, ok := suite.k.GetDao(suite.ctx, 1)
	suite.Require().False(ok)
}

// TestCreateDao_InvalidMetadata rejects empty name.
func (suite *IntegrationTestSuite) TestCreateDao_InvalidMetadata() {
	creator := freshAddr()
	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     types.DaoMetadata{Name: ""},
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().Error(err)
}

// TestCreateDao_DaoByAddress_RoundTrip ensures the address index is wired.
func (suite *IntegrationTestSuite) TestCreateDao_DaoByAddress_RoundTrip() {
	creator := freshAddr()
	suite.expectAccountCreated(1)
	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("eta"),
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)

	daoAddr := suite.mustAcc(resp.AccountAddress)
	got, ok := suite.k.GetDaoByAddress(suite.ctx, daoAddr)
	suite.Require().True(ok)
	suite.Require().Equal(resp.DaoId, got.Id)
}

// TestCreateDao_PreFundedAddressReused: someone pre-funded the next DAO's
// derived address. The keeper reuses the existing BaseAccount rather than
// fail — locking in liveness against DoS-via-pre-fund.
func (suite *IntegrationTestSuite) TestCreateDao_PreFundedAddressReused() {
	creator := freshAddr()
	daoAddr := types.DaoAccountAddress(1)
	suite.acc.EXPECT().HasAccount(gomock.Any(), daoAddr).Return(true).Times(1)

	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator,
		Metadata:     sampleMetadata("preexisting"),
		VotingConfig: staticConfig(creator),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1), resp.DaoId)
	suite.Require().Equal(daoAddr.String(), resp.AccountAddress)
}

// TestCreateDao_BurnerSendFailure: bank reject after spendable check passes
// must propagate; no DAO created, counter unchanged.
func (suite *IntegrationTestSuite) TestCreateDao_BurnerSendFailure() {
	creator := suite.mustAcc(freshAddr())
	feeAmt := math.NewInt(1_000_000)
	feeCoin := sdk.NewCoin("ubze", feeAmt)
	feeCoins := sdk.NewCoins(feeCoin)

	params := types.DefaultParams()
	params.DaoCreationFee = feeCoin
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	suite.bank.EXPECT().
		SpendableCoins(gomock.Any(), creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", feeAmt.MulRaw(2)))).
		Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(gomock.Any(), creator, burnertypes.ModuleName, feeCoins).
		Return(errSimulatedBankFailure).
		Times(1)

	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator.String(),
		Metadata:     sampleMetadata("fail-burner"),
		VotingConfig: staticConfig(creator.String()),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().Error(err)

	suite.Require().Equal(uint64(1), suite.k.GetDaoIDCounter(suite.ctx))
	_, ok := suite.k.GetDao(suite.ctx, 1)
	suite.Require().False(ok)
}

// TestCreateDao_FundCommunityPoolFailure: distr reject must propagate.
func (suite *IntegrationTestSuite) TestCreateDao_FundCommunityPoolFailure() {
	creator := suite.mustAcc(freshAddr())
	feeAmt := math.NewInt(1_000_000)
	feeCoin := sdk.NewCoin("ubze", feeAmt)
	feeCoins := sdk.NewCoins(feeCoin)

	params := types.DefaultParams()
	params.DaoCreationFee = feeCoin
	params.DaoCreationFeeDestination = types.FeeDestinationCommunityPool
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	suite.bank.EXPECT().
		SpendableCoins(gomock.Any(), creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", feeAmt.MulRaw(2)))).
		Times(1)
	suite.distr.EXPECT().
		FundCommunityPool(gomock.Any(), feeCoins, creator).
		Return(errSimulatedDistrFailure).
		Times(1)

	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:      creator.String(),
		Metadata:     sampleMetadata("fail-distr"),
		VotingConfig: staticConfig(creator.String()),
		Governance:   validGovernance(),
		Deposit:      validDeposit(),
	})
	suite.Require().Error(err)

	suite.Require().Equal(uint64(1), suite.k.GetDaoIDCounter(suite.ctx))
	_, ok := suite.k.GetDao(suite.ctx, 1)
	suite.Require().False(ok)
}

// TestCreateDao_MissingVotingConfig rejects MsgCreateDao without the oneof.
func (suite *IntegrationTestSuite) TestCreateDao_MissingVotingConfig() {
	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:  freshAddr(),
		Metadata: sampleMetadata("nocfg"),
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "voting_config")
}

// TestCreateDao_RewardStakedRejected: Epic 2 explicitly rejects the
// REWARD_STAKED variant at creation. Epic 5's MsgUpdateVotingBackend is
// the supported path.
func (suite *IntegrationTestSuite) TestCreateDao_RewardStakedRejected() {
	creator := freshAddr()
	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:  creator,
		Metadata: sampleMetadata("rs"),
		VotingConfig: &types.MsgCreateDao_RewardStaked{
			RewardStaked: &types.RewardStakedVotingConfig{RewardId: "00000000-0000-0000-0000-000000000001"},
		},
	})
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "REWARD_STAKED")
}

// TestCreateDao_StaticMultiMember: more than one initial member.
func (suite *IntegrationTestSuite) TestCreateDao_StaticMultiMember() {
	creator := freshAddr()
	other := freshAddr()
	suite.expectAccountCreated(1)

	resp, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:  creator,
		Metadata: sampleMetadata("multi"),
		VotingConfig: staticConfigWithMembers([]types.StaticMember{
			{Address: creator, Weight: 3},
			{Address: other, Weight: 2},
		}),
		Governance: validGovernance(),
		Deposit:    validDeposit(),
	})
	suite.Require().NoError(err)

	dao, _ := suite.k.GetDao(suite.ctx, resp.DaoId)
	suite.Require().Equal(types.VotingBackendType_VOTING_BACKEND_STATIC, dao.VotingBackend)

	// Total power = 3 + 2.
	total, err := suite.k.TotalVotingPower(suite.ctx, &types.QueryTotalVotingPowerRequest{DaoId: resp.DaoId})
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(5), total.Total)

	// Per-address power.
	for addr, want := range map[string]uint64{creator: 3, other: 2} {
		got, err := suite.k.VotingPower(suite.ctx, &types.QueryVotingPowerRequest{DaoId: resp.DaoId, Address: addr})
		suite.Require().NoError(err)
		suite.Require().Equal(want, got.Power, "addr=%s", addr)
		suite.Require().Equal(uint64(5), got.Total)
	}
}

// TestCreateDao_StaticEmptyMembers rejected by ValidateBasic.
func (suite *IntegrationTestSuite) TestCreateDao_StaticEmptyMembers() {
	_, err := suite.msgServer.CreateDao(suite.ctx, &types.MsgCreateDao{
		Creator:  freshAddr(),
		Metadata: sampleMetadata("empty"),
		VotingConfig: staticConfigWithMembers(nil),
	})
	suite.Require().Error(err)
}
