package keeper_test

import (
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
)

// denomConfigEvents returns the DenomConfigChangeEvents emitted on the suite context so far.
func (suite *IntegrationTestSuite) denomConfigEvents() []*types.DenomConfigChangeEvent {
	var found []*types.DenomConfigChangeEvent
	for _, ev := range suite.ctx.EventManager().Events() {
		if ev.Type != proto.MessageName(&types.DenomConfigChangeEvent{}) {
			continue
		}
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		suite.Require().NoError(err)
		typed, ok := msg.(*types.DenomConfigChangeEvent)
		suite.Require().True(ok)
		found = append(found, typed)
	}

	return found
}

// setDenomConfig sends a MsgSetDenomConfig. A non-empty community pool is checked against the
// bank's blocked addresses, which the mock reports as not blocked.
func (suite *IntegrationTestSuite) setDenomConfig(creator, denom, cp, transferFee, gasSurcharge string) error {
	if cp != "" {
		suite.bank.EXPECT().BlockedAddr(sdk.MustAccAddressFromBech32(cp)).Return(false).MaxTimes(1)
	}
	msg := types.NewMsgSetDenomConfig(creator, denom, cp, math.LegacyMustNewDecFromStr(transferFee), math.LegacyMustNewDecFromStr(gasSurcharge))
	_, err := suite.msgServer.SetDenomConfig(suite.ctx, msg)

	return err
}

// requireConfig checks the stored config through the query and both getters.
func (suite *IntegrationTestSuite) requireConfig(denom, cp, transferFee, gasSurcharge string) {
	res, err := suite.k.DenomConfig(suite.ctx, &types.QueryDenomConfigRequest{Denom: denom})
	suite.Require().NoError(err)
	suite.Require().Equal(cp, res.Config.CommunityPool)
	suite.Require().False(res.Config.TransferFee.IsNil())
	suite.Require().False(res.Config.GasSurcharge.IsNil())
	suite.Require().True(math.LegacyMustNewDecFromStr(transferFee).Equal(res.Config.TransferFee), res.Config.TransferFee.String())
	suite.Require().True(math.LegacyMustNewDecFromStr(gasSurcharge).Equal(res.Config.GasSurcharge), res.Config.GasSurcharge.String())

	tf, gs := suite.k.GetDenomFeeRates(suite.ctx, denom)
	suite.Require().True(tf.Equal(res.Config.TransferFee))
	suite.Require().True(gs.Equal(res.Config.GasSurcharge))

	addr, ok := suite.k.GetDenomCommunityPool(suite.ctx, denom)
	suite.Require().Equal(cp != "", ok)
	if cp != "" {
		suite.Require().Equal(cp, addr.String())
	} else {
		suite.Require().Nil(addr)
	}
}

// A denom without a stored config and a non-factory denom both read as the defaults.
func (suite *IntegrationTestSuite) TestDenomConfig_DefaultsWhenMissing() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupMintableDenom(admin)

	for _, d := range []string{denom, "ubze", "ibc/6490A7EAB61059BFC1CDDEB05917DD70BDF3A611654162A1A47DB930D40D8AF4"} {
		suite.requireConfig(d, "", "0", "0")
		suite.Require().Equal(types.DefaultDenomConfig(), suite.k.GetDenomConfig(suite.ctx, d))
	}
}

func (suite *IntegrationTestSuite) TestDenomConfig_QueryEmptyDenom() {
	_, err := suite.k.DenomConfig(suite.ctx, &types.QueryDenomConfigRequest{})
	suite.Require().Error(err)
	_, err = suite.k.DenomConfig(suite.ctx, nil)
	suite.Require().Error(err)
}

// The admin sets a full config: query, getters and event all carry it.
func (suite *IntegrationTestSuite) TestSetDenomConfig_ValidRequest() {
	admin := sdk.AccAddress("admin").String()
	cp := sdk.AccAddress("community-pool").String()
	denom := suite.setupMintableDenom(admin)

	suite.resetEvents()
	suite.Require().NoError(suite.setDenomConfig(admin, denom, cp, "0.001", "0.5"))
	suite.requireConfig(denom, cp, "0.001", "0.5")

	events := suite.denomConfigEvents()
	suite.Require().Len(events, 1)
	suite.Require().Equal(denom, events[0].Denom)
	suite.Require().Equal(cp, events[0].CommunityPool)
	suite.Require().Equal("0.001000000000000000", events[0].TransferFee.String())
	suite.Require().Equal("0.500000000000000000", events[0].GasSurcharge.String())
}

// Each message replaces the whole config: rates go up and down, the community pool changes.
func (suite *IntegrationTestSuite) TestSetDenomConfig_Replaces() {
	admin := sdk.AccAddress("admin").String()
	cp := sdk.AccAddress("community-pool").String()
	otherCP := sdk.AccAddress("other-pool").String()
	denom := suite.setupMintableDenom(admin)

	suite.Require().NoError(suite.setDenomConfig(admin, denom, cp, "0.01", "1"))
	suite.Require().NoError(suite.setDenomConfig(admin, denom, otherCP, "0.05", "0.1"))
	suite.requireConfig(denom, otherCP, "0.05", "0.1")

	suite.Require().NoError(suite.setDenomConfig(admin, denom, "", "0.002", "2"))
	suite.requireConfig(denom, "", "0.002", "2")
}

// Sending the defaults deletes the entry: export holds nothing and the query still returns the defaults.
func (suite *IntegrationTestSuite) TestSetDenomConfig_DefaultsDeleteTheEntry() {
	admin := sdk.AccAddress("admin").String()
	cp := sdk.AccAddress("community-pool").String()
	denom := suite.setupMintableDenom(admin)

	suite.Require().NoError(suite.setDenomConfig(admin, denom, cp, "0.01", "1"))
	suite.Require().NotNil(suite.k.GetDenomPrefixStore(suite.ctx, denom).Get([]byte(types.DenomConfigKey)))

	suite.resetEvents()
	suite.Require().NoError(suite.setDenomConfig(admin, denom, "", "0", "0"))
	suite.Require().Nil(suite.k.GetDenomPrefixStore(suite.ctx, denom).Get([]byte(types.DenomConfigKey)))
	suite.requireConfig(denom, "", "0", "0")
	suite.Require().Len(suite.denomConfigEvents(), 1)
}

func (suite *IntegrationTestSuite) TestSetDenomConfig_NonAdmin() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupMintableDenom(admin)

	suite.resetEvents()
	err := suite.setDenomConfig(sdk.AccAddress("stranger").String(), denom, "", "0.01", "0")
	suite.Require().ErrorIs(err, types.ErrUnauthorized)
	suite.requireConfig(denom, "", "0", "0")
	suite.Require().Empty(suite.denomConfigEvents())
}

func (suite *IntegrationTestSuite) TestSetDenomConfig_UnknownDenom() {
	admin := sdk.AccAddress("admin").String()
	denom := "factory/" + admin + "/never"
	err := suite.setDenomConfig(admin, denom, "", "0.01", "0")
	suite.Require().ErrorContains(err, "denom authority not found for denom "+denom)
	suite.requireConfig(denom, "", "0", "0")
	suite.Require().Empty(suite.denomConfigEvents())
}

// After a renounce nobody can change the config, and the renounce leaves it as it was.
func (suite *IntegrationTestSuite) TestSetDenomConfig_RenouncedDenomFrozen() {
	admin := sdk.AccAddress("admin").String()
	cp := sdk.AccAddress("community-pool").String()
	denom := suite.setupMintableDenom(admin)
	suite.Require().NoError(suite.setDenomConfig(admin, denom, cp, "0.01", "1"))

	_, err := suite.msgServer.ChangeAdmin(suite.ctx, types.NewMsgChangeAdmin(admin, denom, ""))
	suite.Require().NoError(err)
	suite.requireMintable(denom, false)
	suite.requireConfig(denom, cp, "0.01", "1")

	suite.Require().ErrorIs(suite.setDenomConfig(admin, denom, "", "0", "0"), types.ErrUnauthorized)
	suite.requireConfig(denom, cp, "0.01", "1")
}

// A blocked community pool address (a module account) is refused and nothing is stored.
func (suite *IntegrationTestSuite) TestSetDenomConfig_BlockedCommunityPool() {
	admin := sdk.AccAddress("admin").String()
	blocked := sdk.AccAddress("module-account")
	denom := suite.setupMintableDenom(admin)

	suite.bank.EXPECT().BlockedAddr(blocked).Return(true).Times(1)
	suite.resetEvents()
	msg := types.NewMsgSetDenomConfig(admin, denom, blocked.String(), math.LegacyZeroDec(), math.LegacyZeroDec())
	_, err := suite.msgServer.SetDenomConfig(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidDenomConfig)
	suite.requireConfig(denom, "", "0", "0")
	suite.Require().Empty(suite.denomConfigEvents())
}

// The msg server refuses invalid values even when ValidateBasic was skipped.
func (suite *IntegrationTestSuite) TestSetDenomConfig_InvalidValues() {
	admin := sdk.AccAddress("admin").String()
	denom := suite.setupMintableDenom(admin)

	tests := []struct {
		name string
		msg  *types.MsgSetDenomConfig
	}{
		{name: "missing transfer fee", msg: types.NewMsgSetDenomConfig(admin, denom, "", math.LegacyDec{}, math.LegacyZeroDec())},
		{name: "missing gas surcharge", msg: types.NewMsgSetDenomConfig(admin, denom, "", math.LegacyZeroDec(), math.LegacyDec{})},
		{name: "transfer fee above the cap", msg: types.NewMsgSetDenomConfig(admin, denom, "", math.LegacyMustNewDecFromStr("0.051"), math.LegacyZeroDec())},
		{name: "gas surcharge above the cap", msg: types.NewMsgSetDenomConfig(admin, denom, "", math.LegacyZeroDec(), math.LegacyMustNewDecFromStr("2.01"))},
		{name: "negative transfer fee", msg: types.NewMsgSetDenomConfig(admin, denom, "", math.LegacyMustNewDecFromStr("-0.001"), math.LegacyZeroDec())},
		{name: "invalid community pool", msg: types.NewMsgSetDenomConfig(admin, denom, "bze1notanaddress", math.LegacyZeroDec(), math.LegacyZeroDec())},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			_, err := suite.msgServer.SetDenomConfig(suite.ctx, tt.msg)
			suite.Require().ErrorIs(err, types.ErrInvalidDenomConfig)
			suite.requireConfig(denom, "", "0", "0")
		})
	}
}
