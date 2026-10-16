package keeper_test

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

func govAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

// adminChangeEvents returns the DenomAdminChangeEvents emitted on the suite context so far.
func (suite *IntegrationTestSuite) adminChangeEvents() []*types.DenomAdminChangeEvent {
	var found []*types.DenomAdminChangeEvent
	for _, ev := range suite.ctx.EventManager().Events() {
		if ev.Type != "bze.tokenfactory.DenomAdminChangeEvent" {
			continue
		}
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		suite.Require().NoError(err)
		typed, ok := msg.(*types.DenomAdminChangeEvent)
		suite.Require().True(ok)
		found = append(found, typed)
	}

	return found
}

func (suite *IntegrationTestSuite) resetEvents() {
	suite.ctx = suite.ctx.WithEventManager(sdk.NewEventManager())
}

func (suite *IntegrationTestSuite) requireAdmin(denom, admin string) {
	dAuth, err := suite.k.GetDenomAuthority(suite.ctx, denom)
	suite.Require().NoError(err)
	suite.Require().Equal(admin, dAuth.Admin)
}

// mintAs sends a MsgMint for 1000 coins of denom signed by creator. When expectMint is true the bank
// calls of a successful mint are expected.
func (suite *IntegrationTestSuite) mintAs(creator, denom string, expectMint bool) error {
	suite.bank.EXPECT().GetDenomMetaData(suite.ctx, denom).Return(banktypes.Metadata{Base: denom}, true).Times(1)
	if expectMint {
		coins := sdk.NewCoins(sdk.NewInt64Coin(denom, 1000))
		suite.bank.EXPECT().MintCoins(suite.ctx, types.ModuleName, coins).Return(nil).Times(1)
		suite.bank.EXPECT().SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(creator), coins).Return(nil).Times(1)
	}

	_, err := suite.msgServer.Mint(suite.ctx, &types.MsgMint{Creator: creator, Coins: "1000" + denom})

	return err
}

// A renounced denom is locked until governance assigns it an admin, who can then mint.
func (suite *IntegrationTestSuite) TestSetDenomAdmin_RenouncedDenom() {
	creator := sdk.AccAddress("creator").String()
	newAdmin := sdk.AccAddress("dao-account").String()
	denom := "factory/" + creator + "/testtoken"
	suite.renounceAdmin(creator, denom)

	suite.Require().ErrorIs(suite.mintAs(newAdmin, denom, false), types.ErrUnauthorized, "still locked before the proposal")

	suite.resetEvents()
	res, err := suite.msgServer.SetDenomAdmin(suite.ctx, types.NewMsgSetDenomAdmin(govAuthority(), denom, newAdmin))
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	suite.requireAdmin(denom, newAdmin)
	events := suite.adminChangeEvents()
	suite.Require().Len(events, 1)
	suite.Require().Equal("", events[0].Admin, "the event carries the previous (empty) admin")
	suite.Require().Equal(newAdmin, events[0].NewAdmin)
	suite.Require().Equal(denom, events[0].Denom)

	suite.Require().NoError(suite.mintAs(newAdmin, denom, true))
	suite.Require().ErrorIs(suite.mintAs(creator, denom, false), types.ErrUnauthorized, "the renounced creator stays out")
}

// A live admin is overwritten without its consent: the old admin loses every right, the new one gets them.
func (suite *IntegrationTestSuite) TestSetDenomAdmin_OverwritesLiveAdmin() {
	oldAdmin := sdk.AccAddress("old-admin").String()
	newAdmin := sdk.AccAddress("new-admin").String()
	denom := "factory/" + oldAdmin + "/testtoken"
	suite.Require().NoError(suite.k.SetDenomAuthority(suite.ctx, denom, types.DenomAuthority{Admin: oldAdmin}))

	suite.resetEvents()
	_, err := suite.msgServer.SetDenomAdmin(suite.ctx, types.NewMsgSetDenomAdmin(govAuthority(), denom, newAdmin))
	suite.Require().NoError(err)
	suite.requireAdmin(denom, newAdmin)

	events := suite.adminChangeEvents()
	suite.Require().Len(events, 1)
	suite.Require().Equal(oldAdmin, events[0].Admin)
	suite.Require().Equal(newAdmin, events[0].NewAdmin)

	suite.Require().ErrorIs(suite.mintAs(oldAdmin, denom, false), types.ErrUnauthorized)
	_, err = suite.msgServer.ChangeAdmin(suite.ctx, types.NewMsgChangeAdmin(oldAdmin, denom, sdk.AccAddress("thief").String()))
	suite.Require().ErrorIs(err, types.ErrUnauthorized)

	suite.Require().NoError(suite.mintAs(newAdmin, denom, true))
	_, err = suite.msgServer.ChangeAdmin(suite.ctx, types.NewMsgChangeAdmin(newAdmin, denom, oldAdmin))
	suite.Require().NoError(err, "the new admin can hand the denom on like any creator")
	suite.requireAdmin(denom, oldAdmin)
}

// Only the gov module account may send the message; anybody else changes nothing.
func (suite *IntegrationTestSuite) TestSetDenomAdmin_NonAuthority() {
	creator := sdk.AccAddress("creator").String()
	denom := "factory/" + creator + "/testtoken"
	suite.renounceAdmin(creator, denom)

	for _, signer := range []string{creator, sdk.AccAddress("stranger").String(), ""} {
		suite.resetEvents()
		_, err := suite.msgServer.SetDenomAdmin(suite.ctx, types.NewMsgSetDenomAdmin(signer, denom, creator))
		suite.Require().ErrorIs(err, types.ErrInvalidSigner, "signer %q", signer)
		suite.requireAdmin(denom, "")
		suite.Require().Empty(suite.adminChangeEvents())
	}
}

// A denom without a tokenfactory authority record (never created, ubze, ibc) cannot be adopted.
func (suite *IntegrationTestSuite) TestSetDenomAdmin_UnknownDenom() {
	newAdmin := sdk.AccAddress("new-admin").String()
	for _, denom := range []string{"factory/" + newAdmin + "/never", "ubze", "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"} {
		suite.resetEvents()
		_, err := suite.msgServer.SetDenomAdmin(suite.ctx, types.NewMsgSetDenomAdmin(govAuthority(), denom, newAdmin))
		suite.Require().ErrorIs(err, types.ErrDenomDoesNotExist, denom)
		suite.Require().Empty(suite.adminChangeEvents())

		_, err = suite.k.GetDenomAuthority(suite.ctx, denom)
		suite.Require().Error(err, "nothing written for %s", denom)
	}
}

// Governance cannot lock a denom or assign a malformed address: the handler runs ValidateBasic itself.
func (suite *IntegrationTestSuite) TestSetDenomAdmin_InvalidNewAdmin() {
	admin := sdk.AccAddress("admin").String()
	denom := "factory/" + admin + "/testtoken"
	suite.Require().NoError(suite.k.SetDenomAuthority(suite.ctx, denom, types.DenomAuthority{Admin: admin}))

	for _, newAdmin := range []string{"", "not-an-address"} {
		suite.resetEvents()
		_, err := suite.msgServer.SetDenomAdmin(suite.ctx, types.NewMsgSetDenomAdmin(govAuthority(), denom, newAdmin))
		suite.Require().ErrorIs(err, sdkerrors.ErrInvalidAddress, "new admin %q", newAdmin)
		suite.requireAdmin(denom, admin)
		suite.Require().Empty(suite.adminChangeEvents())
	}
}

// Assigning the current admin again is accepted and still emits the event.
func (suite *IntegrationTestSuite) TestSetDenomAdmin_SameAdmin() {
	admin := sdk.AccAddress("admin").String()
	denom := "factory/" + admin + "/testtoken"
	suite.Require().NoError(suite.k.SetDenomAuthority(suite.ctx, denom, types.DenomAuthority{Admin: admin}))

	suite.resetEvents()
	_, err := suite.msgServer.SetDenomAdmin(suite.ctx, types.NewMsgSetDenomAdmin(govAuthority(), denom, admin))
	suite.Require().NoError(err)
	suite.requireAdmin(denom, admin)

	events := suite.adminChangeEvents()
	suite.Require().Len(events, 1)
	suite.Require().Equal(admin, events[0].Admin)
	suite.Require().Equal(admin, events[0].NewAdmin)
}
