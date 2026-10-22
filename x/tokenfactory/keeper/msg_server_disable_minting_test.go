package keeper_test

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// mintingDisabledEvents returns the DenomMintingDisabledEvents emitted on the suite context so far.
func (suite *IntegrationTestSuite) mintingDisabledEvents() []*types.DenomMintingDisabledEvent {
	var found []*types.DenomMintingDisabledEvent
	for _, ev := range suite.ctx.EventManager().Events() {
		if ev.Type != proto.MessageName(&types.DenomMintingDisabledEvent{}) {
			continue
		}
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		suite.Require().NoError(err)
		typed, ok := msg.(*types.DenomMintingDisabledEvent)
		suite.Require().True(ok)
		found = append(found, typed)
	}

	return found
}

// setupMintableDenom stores a mintable denom administered by admin, like CreateDenom does.
func (suite *IntegrationTestSuite) setupMintableDenom(admin string) string {
	denom := "factory/" + admin + "/testtoken"
	suite.Require().NoError(suite.k.SetDenomAuthority(suite.ctx, denom, types.DenomAuthority{Admin: admin, Mintable: true}))

	return denom
}

func (suite *IntegrationTestSuite) disableMinting(admin, denom string) error {
	_, err := suite.msgServer.DisableMinting(suite.ctx, types.NewMsgDisableMinting(admin, denom))

	return err
}

// The admin turns minting off: the flag is stored, the admin is kept and the event is emitted.
func (suite *IntegrationTestSuite) TestDisableMinting_ValidRequest() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupMintableDenom(admin)

	suite.resetEvents()
	res, err := suite.msgServer.DisableMinting(suite.ctx, types.NewMsgDisableMinting(admin, denom))
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	suite.requireMintable(denom, false)
	suite.requireAdmin(denom, admin)

	events := suite.mintingDisabledEvents()
	suite.Require().Len(events, 1)
	suite.Require().Equal(denom, events[0].Denom)
	suite.Require().Equal(admin, events[0].Admin)
	suite.Require().Empty(suite.adminChangeEvents())
}

// Anybody other than the admin is refused and changes nothing.
func (suite *IntegrationTestSuite) TestDisableMinting_NonAdmin() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupMintableDenom(admin)

	suite.resetEvents()
	err := suite.disableMinting(sdk.AccAddress("stranger").String(), denom)
	suite.Require().ErrorIs(err, types.ErrUnauthorized)
	suite.requireMintable(denom, true)
	suite.Require().Empty(suite.mintingDisabledEvents())
}

// A denom without an authority record cannot be touched.
func (suite *IntegrationTestSuite) TestDisableMinting_UnknownDenom() {
	admin := sdk.AccAddress("admin").String()
	err := suite.disableMinting(admin, "factory/"+admin+"/never")
	suite.Require().Error(err)
}

// Disabling twice is refused with a clear error instead of a silent no-op.
func (suite *IntegrationTestSuite) TestDisableMinting_AlreadyDisabled() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupMintableDenom(admin)
	suite.Require().NoError(suite.disableMinting(admin, denom))

	suite.resetEvents()
	err := suite.disableMinting(admin, denom)
	suite.Require().ErrorIs(err, types.ErrMintingDisabled)
	suite.requireMintable(denom, false)
	suite.Require().Empty(suite.mintingDisabledEvents())
}

// After disabling, mint is refused while burn, metadata, branding and admin change still work.
func (suite *IntegrationTestSuite) TestDisableMinting_AdminKeepsOtherRights() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupMintableDenom(admin)
	suite.Require().NoError(suite.disableMinting(admin, denom))

	suite.Require().ErrorIs(suite.mintAs(admin, denom, false), types.ErrMintingDisabled)

	// burn
	adminAddr := sdk.MustAccAddressFromBech32(admin)
	coins := sdk.NewCoins(sdk.NewInt64Coin(denom, 500))
	suite.bank.EXPECT().SendCoinsFromAccountToModule(suite.ctx, adminAddr, types.ModuleName, coins).Return(nil).Times(1)
	suite.bank.EXPECT().BurnCoins(suite.ctx, types.ModuleName, coins).Return(nil).Times(1)
	_, err := suite.msgServer.Burn(suite.ctx, &types.MsgBurn{Creator: admin, Coins: "500" + denom})
	suite.Require().NoError(err)

	// metadata
	metadata := banktypes.Metadata{
		Base:    denom,
		Display: "TESTTOKEN",
		Name:    "testtoken",
		Symbol:  "TESTTOKEN",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: denom, Exponent: 0},
			{Denom: "TESTTOKEN", Exponent: 6},
		},
	}
	suite.bank.EXPECT().SetDenomMetaData(suite.ctx, metadata).Times(1)
	_, err = suite.msgServer.SetDenomMetadata(suite.ctx, &types.MsgSetDenomMetadata{Creator: admin, Metadata: metadata})
	suite.Require().NoError(err)

	// branding
	branding := types.DenomBranding{
		Font:  "inter",
		Light: &types.BrandingColors{Background: "#FFFFFF", Text: "#111111", Primary: "#1a2b3c", Secondary: "#a1b2c3"},
		Dark:  &types.BrandingColors{Background: "#000000", Text: "#eeeeee", Primary: "#c0ffee", Secondary: "#abcdef"},
	}
	_, err = suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{Creator: admin, Denom: denom, Branding: &branding})
	suite.Require().NoError(err)

	// admin change keeps the flag off
	newAdmin := sdk.AccAddress("new-admin").String()
	_, err = suite.msgServer.ChangeAdmin(suite.ctx, types.NewMsgChangeAdmin(admin, denom, newAdmin))
	suite.Require().NoError(err)
	suite.requireAdmin(denom, newAdmin)
	suite.requireMintable(denom, false)
	suite.Require().ErrorIs(suite.mintAs(newAdmin, denom, false), types.ErrMintingDisabled)
}

// Handing the denom to another address keeps it mintable and emits no minting event.
func (suite *IntegrationTestSuite) TestChangeAdmin_KeepsMintable() {
	admin := sdk.AccAddress("admin").String()
	newAdmin := sdk.AccAddress("new-admin").String()
	denom := suite.setupMintableDenom(admin)

	suite.resetEvents()
	_, err := suite.msgServer.ChangeAdmin(suite.ctx, types.NewMsgChangeAdmin(admin, denom, newAdmin))
	suite.Require().NoError(err)
	suite.requireMintable(denom, true)
	suite.Require().Empty(suite.mintingDisabledEvents())
	suite.Require().NoError(suite.mintAs(newAdmin, denom, true))
}

// Renouncing the admin turns minting off in the same write and emits the event.
func (suite *IntegrationTestSuite) TestChangeAdmin_RenounceDisablesMinting() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupMintableDenom(admin)

	suite.resetEvents()
	_, err := suite.msgServer.ChangeAdmin(suite.ctx, types.NewMsgChangeAdmin(admin, denom, ""))
	suite.Require().NoError(err)
	suite.requireAdmin(denom, "")
	suite.requireMintable(denom, false)

	events := suite.mintingDisabledEvents()
	suite.Require().Len(events, 1)
	suite.Require().Equal(denom, events[0].Denom)
	suite.Require().Equal(admin, events[0].Admin)
	suite.Require().Len(suite.adminChangeEvents(), 1)
}

// Renouncing a denom whose minting is already off emits no second minting event.
func (suite *IntegrationTestSuite) TestChangeAdmin_RenounceNonMintableEmitsNoMintingEvent() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupMintableDenom(admin)
	suite.Require().NoError(suite.disableMinting(admin, denom))

	suite.resetEvents()
	_, err := suite.msgServer.ChangeAdmin(suite.ctx, types.NewMsgChangeAdmin(admin, denom, ""))
	suite.Require().NoError(err)
	suite.requireMintable(denom, false)
	suite.Require().Empty(suite.mintingDisabledEvents())
	suite.Require().Len(suite.adminChangeEvents(), 1)
}

// Governance assigning an admin to a non-mintable denom leaves it non-mintable.
func (suite *IntegrationTestSuite) TestSetDenomAdmin_NonMintableStaysNonMintable() {
	admin := sdk.AccAddress("admin").String()
	govAdmin := sdk.AccAddress("gov-admin").String()
	denom := suite.setupMintableDenom(admin)
	suite.Require().NoError(suite.disableMinting(admin, denom))

	_, err := suite.msgServer.SetDenomAdmin(suite.ctx, types.NewMsgSetDenomAdmin(govAuthority(), denom, govAdmin))
	suite.Require().NoError(err)
	suite.requireAdmin(denom, govAdmin)
	suite.requireMintable(denom, false)
	suite.Require().ErrorIs(suite.mintAs(govAdmin, denom, false), types.ErrMintingDisabled)
	suite.Require().ErrorIs(suite.disableMinting(govAdmin, denom), types.ErrMintingDisabled)
}
