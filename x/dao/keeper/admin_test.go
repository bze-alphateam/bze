package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

var (
	victimAddr   = sdk.AccAddress("dao_victim__________")
	attackerAddr = sdk.AccAddress("dao_attacker________")
)

func ubze(amount int64) sdk.Coins {
	return sdk.NewCoins(sdk.NewInt64Coin("ubze", amount))
}

// createAdminDao creates DAO 1 with adminAddr as its external admin and a
// treasury of 100ubze.
func (suite *IntegrationTestSuite) createAdminDao() types.Dao {
	suite.expectCreation(1)
	msg := validCreateDaoMsg()
	msg.Admin = adminAddr.String()
	_, err := suite.msgServer.CreateDao(suite.ctx, msg)
	suite.Require().NoError(err)

	suite.ledger.fund(daoAddress(1), ubze(100))
	dao, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().True(found)

	return dao
}

func (suite *IntegrationTestSuite) adminExecute(admin sdk.AccAddress, daoId uint64, msgs ...sdk.Msg) error {
	msg, err := types.NewMsgAdminExecute(admin.String(), daoId, msgs)
	suite.Require().NoError(err)

	_, err = suite.msgServer.AdminExecute(suite.ctx, msg)

	return err
}

func treasurySend(to sdk.AccAddress, amount int64) *banktypes.MsgSend {
	return banktypes.NewMsgSend(daoAddress(1), to, ubze(amount))
}

func (suite *IntegrationTestSuite) hasEvent(eventType string) bool {
	for _, event := range suite.ctx.EventManager().Events() {
		if event.Type == eventType {
			return true
		}
	}

	return false
}

// D10: the external admin moves treasury funds; the handler's events and the
// module's event are emitted.
func (suite *IntegrationTestSuite) TestAdminExecute_AdminSpendsTreasury() {
	suite.createAdminDao()

	suite.Require().NoError(suite.adminExecute(adminAddr, 1, treasurySend(memberA, 40), treasurySend(memberB, 10)))

	suite.Require().Equal(ubze(50), suite.ledger.balance(daoAddress(1)))
	suite.Require().Equal(ubze(40), suite.ledger.balance(memberA))
	suite.Require().Equal(ubze(10), suite.ledger.balance(memberB))
	suite.Require().True(suite.hasEvent(fakeSendEvent))
	suite.Require().True(suite.hasEvent("bze.dao.EventAdminExecuted"))
}

// D10: only the admin can execute; members and the creator cannot.
func (suite *IntegrationTestSuite) TestAdminExecute_NonAdminRejected() {
	suite.createAdminDao()

	for _, signer := range []sdk.AccAddress{memberA, creatorAddr, attackerAddr} {
		err := suite.adminExecute(signer, 1, treasurySend(signer, 40))
		suite.Require().ErrorIs(err, types.ErrNotDaoAdmin, signer.String())
	}
	suite.Require().Equal(ubze(100), suite.ledger.balance(daoAddress(1)))
}

// D1: the admin cannot spend an address that is not the DAO.
func (suite *IntegrationTestSuite) TestAdminExecute_VictimSendRejected() {
	suite.createAdminDao()
	suite.ledger.fund(victimAddr, ubze(100))

	err := suite.adminExecute(adminAddr, 1, banktypes.NewMsgSend(victimAddr, adminAddr, ubze(100)))

	suite.Require().ErrorIs(err, types.ErrUnauthorizedSigner)
	suite.Require().Equal(ubze(100), suite.ledger.balance(victimAddr))
}

// D24: every signer counts, not only the first; the DAO plus an attacker is refused.
func (suite *IntegrationTestSuite) TestAdminExecute_SecondSignerRejected() {
	suite.createAdminDao()

	multiSend := &banktypes.MsgMultiSend{
		Inputs: []banktypes.Input{
			banktypes.NewInput(daoAddress(1), ubze(10)),
			banktypes.NewInput(attackerAddr, ubze(10)),
		},
		Outputs: []banktypes.Output{banktypes.NewOutput(attackerAddr, ubze(20))},
	}
	err := suite.adminExecute(adminAddr, 1, multiSend)

	suite.Require().ErrorIs(err, types.ErrUnauthorizedSigner)
	suite.Require().ErrorContains(err, attackerAddr.String())
}

// D26: a self-owned DAO has no external admin; proposals execute for it.
func (suite *IntegrationTestSuite) TestAdminExecute_SelfOwnedDaoRejected() {
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)
	suite.ledger.fund(daoAddress(1), ubze(100))

	for _, signer := range []sdk.AccAddress{daoAddress(1), creatorAddr, memberA} {
		err := suite.adminExecute(signer, 1, treasurySend(signer, 40))
		suite.Require().ErrorIs(err, types.ErrNotDaoAdmin, signer.String())
		suite.Require().ErrorContains(err, "self-owned")
	}
	suite.Require().Equal(ubze(100), suite.ledger.balance(daoAddress(1)))
}

// D26: a MsgAdminExecute nested in the msgs cannot pass. Signed by the admin it
// fails the DAO-signer check; signed by the DAO it is not the admin.
func (suite *IntegrationTestSuite) TestAdminExecute_NestedRejected() {
	suite.createAdminDao()

	for _, tc := range []struct {
		desc    string
		signer  sdk.AccAddress
		wantErr error
	}{
		{desc: "inner signed by the admin", signer: adminAddr, wantErr: types.ErrUnauthorizedSigner},
		{desc: "inner signed by the dao", signer: daoAddress(1), wantErr: types.ErrNotDaoAdmin},
	} {
		inner, err := types.NewMsgAdminExecute(tc.signer.String(), 1, []sdk.Msg{treasurySend(attackerAddr, 40)})
		suite.Require().NoError(err)

		err = suite.adminExecute(adminAddr, 1, inner)
		suite.Require().ErrorIs(err, tc.wantErr, tc.desc)
	}
	suite.Require().Equal(ubze(100), suite.ledger.balance(daoAddress(1)))
}

func (suite *IntegrationTestSuite) TestAdminExecute_UnknownDao() {
	suite.createAdminDao()

	err := suite.adminExecute(adminAddr, 2, treasurySend(adminAddr, 1))
	suite.Require().ErrorIs(err, types.ErrDaoNotFound)
}

func (suite *IntegrationTestSuite) TestAdminExecute_NoMsgsRejected() {
	suite.createAdminDao()

	err := suite.adminExecute(adminAddr, 1)
	suite.Require().ErrorIs(err, types.ErrInvalidMsgs)
}

func (suite *IntegrationTestSuite) TestAdminExecute_MsgCountCap() {
	suite.createAdminDao()
	params := suite.k.GetParams(suite.ctx)
	params.MaxMsgsPerProposal = 2
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	err := suite.adminExecute(adminAddr, 1, treasurySend(memberA, 1), treasurySend(memberA, 1), treasurySend(memberA, 1))
	suite.Require().ErrorIs(err, types.ErrInvalidMsgs)
	suite.Require().Equal(ubze(100), suite.ledger.balance(daoAddress(1)))

	suite.Require().NoError(suite.adminExecute(adminAddr, 1, treasurySend(memberA, 1), treasurySend(memberA, 1)))
	suite.Require().Equal(ubze(2), suite.ledger.balance(memberA))
}

// The byte cap is the total packed size of the msgs.
func (suite *IntegrationTestSuite) TestAdminExecute_MsgBytesCap() {
	suite.createAdminDao()
	probe, err := types.NewMsgAdminExecute(adminAddr.String(), 1, []sdk.Msg{treasurySend(memberA, 1), treasurySend(memberA, 1)})
	suite.Require().NoError(err)
	size := uint64(probe.Msgs[0].Size() + probe.Msgs[1].Size())

	params := suite.k.GetParams(suite.ctx)
	params.MaxProposalBytes = size - 1
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	err = suite.adminExecute(adminAddr, 1, treasurySend(memberA, 1), treasurySend(memberA, 1))
	suite.Require().ErrorIs(err, types.ErrInvalidMsgs)
	suite.Require().Equal(ubze(100), suite.ledger.balance(daoAddress(1)))

	params.MaxProposalBytes = size
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))
	suite.Require().NoError(suite.adminExecute(adminAddr, 1, treasurySend(memberA, 1), treasurySend(memberA, 1)))
}

// A message without a route is refused.
func (suite *IntegrationTestSuite) TestAdminExecute_UnroutableMsgRejected() {
	suite.createAdminDao()

	err := suite.adminExecute(adminAddr, 1, distrtypes.NewMsgFundCommunityPool(ubze(1), daoAddress(1).String()))
	suite.Require().ErrorIs(err, types.ErrInvalidMsgs)
	suite.Require().ErrorContains(err, "no handler")
}

// A failing handler's error is returned as is, so the transaction reverts.
func (suite *IntegrationTestSuite) TestAdminExecute_HandlerErrorReturned() {
	suite.createAdminDao()

	err := suite.adminExecute(adminAddr, 1, treasurySend(memberA, 101))
	suite.Require().ErrorIs(err, sdkerrors.ErrInsufficientFunds)
	suite.Require().False(suite.hasEvent("bze.dao.EventAdminExecuted"))
}

var successorAddr = sdk.AccAddress("dao_successor_______")

func (suite *IntegrationTestSuite) nominate(authority sdk.AccAddress, daoId uint64, nominee string) error {
	_, err := suite.msgServer.NominateAdmin(suite.ctx, types.NewMsgNominateAdmin(authority.String(), daoId, nominee))

	return err
}

// requireAdminInvariant checks I7 on DAO 1: the admin is set and indexed alone,
// and a pending admin is neither the admin nor the DAO itself.
func (suite *IntegrationTestSuite) requireAdminInvariant(wantAdmin sdk.AccAddress, wantPending string) {
	dao, found := suite.k.GetDao(suite.ctx, 1)
	suite.Require().True(found)

	suite.Require().Equal(wantAdmin.String(), dao.Admin)
	suite.Require().Equal(wantPending, dao.PendingAdmin)
	if dao.PendingAdmin != "" {
		suite.Require().NotEqual(dao.Admin, dao.PendingAdmin)
		suite.Require().NotEqual(dao.Address, dao.PendingAdmin)
	}

	suite.Require().True(suite.k.HasDaoAdminIndex(suite.ctx, wantAdmin, 1))
	for _, other := range []sdk.AccAddress{adminAddr, daoAddress(1), successorAddr, creatorAddr} {
		if !other.Equals(wantAdmin) {
			suite.Require().False(suite.k.HasDaoAdminIndex(suite.ctx, other, 1), other.String())
		}
	}
}

// The external admin nominates a successor: it is stored as pending, the admin
// stays in charge until the successor accepts.
func (suite *IntegrationTestSuite) TestNominateAdmin_StoresPending() {
	suite.createAdminDao()

	suite.Require().NoError(suite.nominate(adminAddr, 1, successorAddr.String()))

	suite.requireAdminInvariant(adminAddr, successorAddr.String())
	suite.Require().True(suite.hasEvent("bze.dao.EventAdminNominated"))
	suite.Require().False(suite.hasEvent("bze.dao.EventAdminChanged"))
}

// D27: a second nomination waits for the first to be settled, also a hand-over
// to the voters.
func (suite *IntegrationTestSuite) TestNominateAdmin_PendingRejected() {
	suite.createAdminDao()
	suite.Require().NoError(suite.nominate(adminAddr, 1, successorAddr.String()))

	for _, nominee := range []string{attackerAddr.String(), ""} {
		err := suite.nominate(adminAddr, 1, nominee)
		suite.Require().ErrorIs(err, types.ErrNominationPending, nominee)
	}
	suite.requireAdminInvariant(adminAddr, successorAddr.String())
}

// D27: the nominee is neither the current admin nor the DAO itself.
func (suite *IntegrationTestSuite) TestNominateAdmin_InvalidNomineeRejected() {
	suite.createAdminDao()

	for _, tc := range []struct {
		desc    string
		nominee sdk.AccAddress
		wantMsg string
	}{
		{desc: "current admin", nominee: adminAddr, wantMsg: "already the admin"},
		{desc: "the dao itself", nominee: daoAddress(1), wantMsg: "nominate nobody"},
	} {
		err := suite.nominate(adminAddr, 1, tc.nominee.String())
		suite.Require().ErrorIs(err, types.ErrInvalidNominee, tc.desc)
		suite.Require().ErrorContains(err, tc.wantMsg, tc.desc)
	}
	suite.requireAdminInvariant(adminAddr, "")
}

// D27: only the current admin nominates; members, the creator, the DAO account
// and others are refused.
func (suite *IntegrationTestSuite) TestNominateAdmin_NonAdminRejected() {
	suite.createAdminDao()

	for _, signer := range []sdk.AccAddress{memberA, creatorAddr, daoAddress(1), attackerAddr} {
		err := suite.nominate(signer, 1, signer.String())
		suite.Require().ErrorIs(err, types.ErrNotDaoAdmin, signer.String())
	}
	suite.requireAdminInvariant(adminAddr, "")
}

// The admin cannot nominate in the DAO's name through admin-execute: the DAO
// is not its own admin while an external one exists.
func (suite *IntegrationTestSuite) TestNominateAdmin_AsDaoThroughAdminExecuteRejected() {
	suite.createAdminDao()

	err := suite.adminExecute(adminAddr, 1, types.NewMsgNominateAdmin(daoAddress(1).String(), 1, attackerAddr.String()))

	suite.Require().ErrorIs(err, types.ErrNotDaoAdmin)
	suite.requireAdminInvariant(adminAddr, "")
}

func (suite *IntegrationTestSuite) TestNominateAdmin_UnknownDao() {
	suite.createAdminDao()

	err := suite.nominate(adminAddr, 2, successorAddr.String())
	suite.Require().ErrorIs(err, types.ErrDaoNotFound)
}

// Nominating nobody hands a static-list DAO to its members at once.
func (suite *IntegrationTestSuite) TestNominateAdmin_NoneMakesDaoSelfOwned() {
	suite.createAdminDao()

	suite.Require().NoError(suite.nominate(adminAddr, 1, ""))

	suite.requireAdminInvariant(daoAddress(1), "")
	suite.Require().True(suite.hasEvent("bze.dao.EventAdminChanged"))
	// the former admin has lost its superuser power
	err := suite.adminExecute(adminAddr, 1, treasurySend(adminAddr, 1))
	suite.Require().ErrorIs(err, types.ErrNotDaoAdmin)
}

// D11: a stake DAO is handed to its voters only when they can vote: at least
// one source and some stake.
func (suite *IntegrationTestSuite) TestNominateAdmin_NoneNeedsLiveElectorate() {
	for _, tc := range []struct {
		desc    string
		srIds   []string
		live    bool
		wantErr error
	}{
		{desc: "no sources", wantErr: types.ErrElectorateNotLive},
		{desc: "nothing staked", srIds: []string{"2"}, wantErr: types.ErrElectorateNotLive},
		{desc: "stake on a source", srIds: []string{"1", "2"}, live: true},
	} {
		suite.SetupTest()
		suite.stubRewards(defaultRewardsState())
		suite.expectCreation(1)
		msg := validStakeDaoMsg(tc.srIds, "")
		msg.Admin = adminAddr.String()
		_, err := suite.msgServer.CreateDao(suite.ctx, msg)
		suite.Require().NoError(err, tc.desc)

		err = suite.nominate(adminAddr, 1, "")
		if !tc.live {
			suite.Require().ErrorIs(err, tc.wantErr, tc.desc)
			suite.requireAdminInvariant(adminAddr, "")
			continue
		}

		suite.Require().NoError(err, tc.desc)
		suite.requireAdminInvariant(daoAddress(1), "")
	}
}

// A self-owned DAO has nothing to hand over to its voters.
func (suite *IntegrationTestSuite) TestNominateAdmin_NoneOnSelfOwnedDaoRejected() {
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)

	err = suite.nominate(daoAddress(1), 1, "")

	suite.Require().ErrorIs(err, types.ErrInvalidNominee)
	suite.Require().ErrorContains(err, "already self-owned")
	suite.requireAdminInvariant(daoAddress(1), "")
}

// A self-owned DAO (signing through a passed proposal, called directly here)
// nominates an external admin.
func (suite *IntegrationTestSuite) TestNominateAdmin_SelfOwnedDaoNominates() {
	suite.expectCreation(1)
	_, err := suite.msgServer.CreateDao(suite.ctx, validCreateDaoMsg())
	suite.Require().NoError(err)

	suite.Require().NoError(suite.nominate(daoAddress(1), 1, adminAddr.String()))

	suite.requireAdminInvariant(daoAddress(1), adminAddr.String())
}
