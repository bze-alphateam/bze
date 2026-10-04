package keeper_test

import (
	"fmt"

	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"
)

func haltAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

// countTypedEvents counts the typed events of tev's kind emitted so far on the suite context.
func (suite *IntegrationTestSuite) countTypedEvents(tev proto.Message) int {
	evType := proto.MessageName(tev)
	n := 0
	for _, ev := range suite.ctx.EventManager().Events() {
		if ev.Type == evType {
			n++
		}
	}

	return n
}

func (suite *IntegrationTestSuite) TestMsgHaltDenoms_HaltsEveryListedDenom() {
	msg := types.NewMsgHaltDenoms(haltAuthority(), []string{denomHalted, haltedIbcDenom})

	resp, err := suite.msgServer.HaltDenoms(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	suite.Require().True(suite.k.IsDenomHalted(suite.ctx, denomHalted))
	suite.Require().True(suite.k.IsDenomHalted(suite.ctx, haltedIbcDenom))
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomStake), "unlisted denoms are untouched")
	suite.Require().Equal(2, suite.countTypedEvents(&types.DenomHaltedEvent{}), "one event per denom halted")
}

func (suite *IntegrationTestSuite) TestMsgHaltDenoms_InvalidAuthorityRejected() {
	msg := types.NewMsgHaltDenoms(getTestAddress(), []string{denomHalted})

	_, err := suite.msgServer.HaltDenoms(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidSigner)
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomHalted))
	suite.Require().Zero(suite.countTypedEvents(&types.DenomHaltedEvent{}))
}

// The native denom is refused before anything is written: the other denoms of the same message stay
// un-halted too, so a rejected proposal changes nothing.
func (suite *IntegrationTestSuite) TestMsgHaltDenoms_NativeDenomRejectedNothingWritten() {
	msg := types.NewMsgHaltDenoms(haltAuthority(), []string{denomHalted, denomBze})

	_, err := suite.msgServer.HaltDenoms(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidDenom)
	suite.Require().Contains(err.Error(), "cannot be halted")
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomBze))
	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomHalted))
	suite.Require().Empty(suite.k.GetAllHaltedDenoms(suite.ctx))
}

func (suite *IntegrationTestSuite) TestMsgHaltDenoms_ValidationRejected() {
	tooMany := make([]string, types.MaxDenomsPerHaltMsg+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("udenom%d", i)
	}

	cases := []struct {
		name   string
		denoms []string
	}{
		{"empty list", nil},
		{"duplicate", []string{denomHalted, denomHalted}},
		{"malformed denom", []string{"1bad"}},
		{"empty denom", []string{""}},
		{"too many", tooMany},
	}
	for _, c := range cases {
		_, err := suite.msgServer.HaltDenoms(suite.ctx, types.NewMsgHaltDenoms(haltAuthority(), c.denoms))
		suite.Require().ErrorIs(err, types.ErrInvalidDenom, c.name)
		suite.Require().Empty(suite.k.GetAllHaltedDenoms(suite.ctx), c.name)
	}
}

// Halting an already halted denom is a no-op, so a batch proposal never fails because one of its
// entries was halted by an earlier proposal while this one was being voted.
func (suite *IntegrationTestSuite) TestMsgHaltDenoms_Idempotent() {
	_, err := suite.msgServer.HaltDenoms(suite.ctx, types.NewMsgHaltDenoms(haltAuthority(), []string{denomHalted}))
	suite.Require().NoError(err)

	_, err = suite.msgServer.HaltDenoms(suite.ctx, types.NewMsgHaltDenoms(haltAuthority(), []string{denomHalted, haltedIbcDenom}))
	suite.Require().NoError(err)

	suite.Require().Equal([]string{haltedIbcDenom, denomHalted}, suite.k.GetAllHaltedDenoms(suite.ctx))
	suite.Require().Equal(2, suite.countTypedEvents(&types.DenomHaltedEvent{}), "the re-halted denom emits nothing")
}

func (suite *IntegrationTestSuite) TestMsgUnhaltDenoms_LiftsOnlyListedDenoms() {
	suite.setHaltedDenoms(denomHalted, haltedIbcDenom)

	resp, err := suite.msgServer.UnhaltDenoms(suite.ctx, types.NewMsgUnhaltDenoms(haltAuthority(), []string{denomHalted}))
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	suite.Require().False(suite.k.IsDenomHalted(suite.ctx, denomHalted))
	suite.Require().True(suite.k.IsDenomHalted(suite.ctx, haltedIbcDenom), "the other halted denom stays halted")
	suite.Require().Equal(1, suite.countTypedEvents(&types.DenomUnhaltedEvent{}))
}

// Un-halting a denom that is not halted is a no-op with no event.
func (suite *IntegrationTestSuite) TestMsgUnhaltDenoms_NotHaltedIsNoop() {
	suite.setHaltedDenoms(haltedIbcDenom)

	_, err := suite.msgServer.UnhaltDenoms(suite.ctx, types.NewMsgUnhaltDenoms(haltAuthority(), []string{denomHalted, haltedIbcDenom}))
	suite.Require().NoError(err)

	suite.Require().Empty(suite.k.GetAllHaltedDenoms(suite.ctx))
	suite.Require().Equal(1, suite.countTypedEvents(&types.DenomUnhaltedEvent{}), "only the denom that was halted emits")
}

func (suite *IntegrationTestSuite) TestMsgUnhaltDenoms_InvalidAuthorityRejected() {
	suite.setHaltedDenoms(denomHalted)

	_, err := suite.msgServer.UnhaltDenoms(suite.ctx, types.NewMsgUnhaltDenoms(sdk.AccAddress("not-the-gov-account_").String(), []string{denomHalted}))
	suite.Require().ErrorIs(err, types.ErrInvalidSigner)
	suite.Require().True(suite.k.IsDenomHalted(suite.ctx, denomHalted), "a rejected message changes nothing")
}

func (suite *IntegrationTestSuite) TestMsgUnhaltDenoms_ValidationRejected() {
	suite.setHaltedDenoms(denomHalted)

	for name, denoms := range map[string][]string{"empty list": nil, "duplicate": {denomHalted, denomHalted}, "malformed": {"1bad"}} {
		_, err := suite.msgServer.UnhaltDenoms(suite.ctx, types.NewMsgUnhaltDenoms(haltAuthority(), denoms))
		suite.Require().ErrorIs(err, types.ErrInvalidDenom, name)
		suite.Require().True(suite.k.IsDenomHalted(suite.ctx, denomHalted), name)
	}
}

// Halt through the msg server, then a market on the denom is refused; un-halt through the msg server,
// then the same message gets past the guard again — the two governance messages flip every guard.
func (suite *IntegrationTestSuite) TestMsgHaltUnhalt_FlipsTheGuard() {
	msg := &types.MsgCreateMarket{Creator: getTestAddress(), Base: denomHalted, Quote: denomStake}

	_, err := suite.msgServer.HaltDenoms(suite.ctx, types.NewMsgHaltDenoms(haltAuthority(), []string{denomHalted}))
	suite.Require().NoError(err)

	// refused by the halt guard before the supply check or any fee capture: no bank expectation
	_, err = suite.msgServer.CreateMarket(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrDenomHalted)

	_, err = suite.msgServer.UnhaltDenoms(suite.ctx, types.NewMsgUnhaltDenoms(haltAuthority(), []string{denomHalted}))
	suite.Require().NoError(err)

	// past the guard: the message reaches the supply check, the step right after it
	suite.bankMock.EXPECT().HasSupply(gomock.Any(), denomHalted).Return(false).Times(1)
	_, err = suite.msgServer.CreateMarket(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrDenomHasNoSupply)
}
