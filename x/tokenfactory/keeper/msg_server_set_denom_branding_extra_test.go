package keeper_test

import (
	"strings"

	"github.com/cosmos/gogoproto/proto"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Extra SetDenomBranding coverage: event-payload correctness (indexers read the attribute, not
// just the type) and robustness to a malformed denom that slips past ValidateBasic.

// brandingEventDenom returns the `denom` attribute of the last DenomBrandingChangeEvent, with the
// JSON quoting the event manager adds stripped. Empty if no such event was emitted.
func (suite *IntegrationTestSuite) brandingEventDenom() string {
	eventType := proto.MessageName(&types.DenomBrandingChangeEvent{})
	denom := ""
	for _, event := range suite.ctx.EventManager().Events() {
		if event.Type != eventType {
			continue
		}
		for _, attr := range event.Attributes {
			if attr.Key == "denom" {
				denom = strings.Trim(attr.Value, "\"")
			}
		}
	}

	return denom
}

// TestSetDenomBranding_EventCarriesDenom: the existing suite only counts branding events; this
// pins that the emitted DenomBrandingChangeEvent actually names the denom that changed, so an
// indexer refreshing branding for the right denom is covered.
func (suite *IntegrationTestSuite) TestSetDenomBranding_EventCarriesDenom() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupBrandedDenom(admin)
	branding := testBranding()

	_, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
		Creator:  admin,
		Denom:    denom,
		Branding: &branding,
	})
	suite.Require().NoError(err)

	suite.Require().Equal(denom, suite.brandingEventDenom())
}

// TestSetDenomBranding_MalformedDenom_NoPanic: MsgSetDenomBranding.ValidateBasic only rejects an
// empty denom — unlike MsgMint/MsgBurn it does not call DeconstructDenom — so a structurally
// invalid denom reaches the keeper. It must be stopped cleanly by the authority lookup (no
// panic, nothing stored, no event), not by a coin-construction panic. Documents the validation
// divergence flagged in the BZE-99 sweep.
func (suite *IntegrationTestSuite) TestSetDenomBranding_MalformedDenom_NoPanic() {
	admin := sdk.AccAddress("admin").String()
	branding := testBranding()

	for _, denom := range []string{"factory//", "not-a-denom", "factory/badbech32/x"} {
		suite.Require().NotPanics(func() {
			_, err := suite.msgServer.SetDenomBranding(suite.ctx, &types.MsgSetDenomBranding{
				Creator:  admin,
				Denom:    denom,
				Branding: &branding,
			})
			suite.Require().Error(err, "denom %q", denom)
		}, "denom %q panicked", denom)

		_, found := suite.k.GetDenomBranding(suite.ctx, denom)
		suite.Require().False(found, "denom %q must not be stored", denom)
	}
}
