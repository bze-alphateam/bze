package keeper_test

import (
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

func (suite *IntegrationTestSuite) setupBrandedDenom(admin string) string {
	denom := "factory/" + sdk.AccAddress("creator").String() + "/token"
	err := suite.k.SetDenomAuthority(suite.ctx, denom, types.DenomAuthority{Admin: admin})
	suite.Require().NoError(err)

	return denom
}

func (suite *IntegrationTestSuite) countBrandingChangeEvents() int {
	eventType := proto.MessageName(&types.DenomBrandingChangeEvent{})
	count := 0
	for _, event := range suite.ctx.EventManager().Events() {
		if event.Type == eventType {
			count++
		}
	}

	return count
}

func (suite *IntegrationTestSuite) TestSetDenomBranding_ValidRequest() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupBrandedDenom(admin)
	branding := testBranding()

	res, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  admin,
		Denom:    denom,
		Branding: &branding,
	})

	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	stored, found := suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().True(found)
	suite.Require().True(branding.Equal(stored))
	suite.Require().Equal(1, suite.countBrandingChangeEvents())
}

func (suite *IntegrationTestSuite) TestSetDenomBranding_Overwrite() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupBrandedDenom(admin)

	first := testBranding()
	_, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  admin,
		Denom:    denom,
		Branding: &first,
	})
	suite.Require().NoError(err)

	second := testBranding()
	second.Font = "jetbrains-mono"
	_, err = suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  admin,
		Denom:    denom,
		Branding: &second,
	})
	suite.Require().NoError(err)

	stored, found := suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().True(found)
	suite.Require().True(second.Equal(stored))
	suite.Require().Equal(2, suite.countBrandingChangeEvents())
}

func (suite *IntegrationTestSuite) TestSetDenomBranding_Clear() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupBrandedDenom(admin)
	branding := testBranding()

	_, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  admin,
		Denom:    denom,
		Branding: &branding,
	})
	suite.Require().NoError(err)

	// clear with nil branding
	_, err = suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator: admin,
		Denom:   denom,
	})
	suite.Require().NoError(err)

	_, found := suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().False(found)
	suite.Require().Equal(2, suite.countBrandingChangeEvents())
}

func (suite *IntegrationTestSuite) TestSetDenomBranding_ClearWithEmptyBranding() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupBrandedDenom(admin)
	branding := testBranding()

	_, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  admin,
		Denom:    denom,
		Branding: &branding,
	})
	suite.Require().NoError(err)

	_, err = suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  admin,
		Denom:    denom,
		Branding: &types.DenomBranding{},
	})
	suite.Require().NoError(err)

	_, found := suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestSetDenomBranding_ClearIsIdempotent() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupBrandedDenom(admin)

	// nothing stored: clearing succeeds and still emits the event
	res, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator: admin,
		Denom:   denom,
	})

	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(1, suite.countBrandingChangeEvents())
}

func (suite *IntegrationTestSuite) TestSetDenomBranding_NonAdmin() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupBrandedDenom(admin)
	branding := testBranding()

	res, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  sdk.AccAddress("intruder").String(),
		Denom:    denom,
		Branding: &branding,
	})

	suite.Require().ErrorIs(err, types.ErrUnauthorized)
	suite.Require().Nil(res)
	_, found := suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().False(found)
	suite.Require().Equal(0, suite.countBrandingChangeEvents())
}

func (suite *IntegrationTestSuite) TestSetDenomBranding_RenouncedAdmin() {
	// a renounced admin (empty admin) freezes branding for everyone
	denom := suite.setupBrandedDenom("")
	branding := testBranding()

	res, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  sdk.AccAddress("creator").String(),
		Denom:    denom,
		Branding: &branding,
	})

	suite.Require().ErrorIs(err, types.ErrUnauthorized)
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestSetDenomBranding_AfterAdminTransfer() {
	oldAdmin := sdk.AccAddress("admin").String()
	newAdmin := sdk.AccAddress("newadmin").String()
	denom := suite.setupBrandedDenom(oldAdmin)
	branding := testBranding()

	// old admin sets branding, then transfers the denom
	_, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  oldAdmin,
		Denom:    denom,
		Branding: &branding,
	})
	suite.Require().NoError(err)

	_, err = suite.msgServer.ChangeAdmin(suite.ctx, &types.MsgChangeAdmin{
		Creator:  oldAdmin,
		Denom:    denom,
		NewAdmin: newAdmin,
	})
	suite.Require().NoError(err)

	// branding followed the denom
	stored, found := suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().True(found)
	suite.Require().True(branding.Equal(stored))

	// old admin can no longer change it
	updated := testBranding()
	updated.Font = "roboto"
	_, err = suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  oldAdmin,
		Denom:    denom,
		Branding: &updated,
	})
	suite.Require().ErrorIs(err, types.ErrUnauthorized)

	// new admin can
	_, err = suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  newAdmin,
		Denom:    denom,
		Branding: &updated,
	})
	suite.Require().NoError(err)

	stored, found = suite.k.GetDenomBranding(suite.ctx, denom)
	suite.Require().True(found)
	suite.Require().True(updated.Equal(stored))
}

func (suite *IntegrationTestSuite) TestSetDenomBranding_DenomWithoutAuthority() {
	branding := testBranding()

	res, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  sdk.AccAddress("admin").String(),
		Denom:    "ubze",
		Branding: &branding,
	})

	suite.Require().Error(err)
	suite.Require().Nil(res)
	suite.Require().Contains(err.Error(), "denom authority not found")
}

func (suite *IntegrationTestSuite) TestSetDenomBranding_InvalidPackage() {
	// defense in depth: the keeper validates even if ValidateBasic was bypassed
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupBrandedDenom(admin)

	branding := testBranding()
	branding.Light = nil

	res, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  admin,
		Denom:    denom,
		Branding: &branding,
	})

	suite.Require().ErrorIs(err, types.ErrInvalidBranding)
	suite.Require().Nil(res)
}
