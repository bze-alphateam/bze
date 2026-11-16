package keeper_test

import (
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"
)

// The message server checks the raffle fields itself, without relying on ValidateBasic.
func (suite *IntegrationTestSuite) TestMsgRaffle_StartRaffle_AmountFormat() {
	creator := sdk.AccAddress("creator")
	denom := "utoken"
	valid := func() *types.MsgStartRaffle {
		return &types.MsgStartRaffle{Creator: creator.String(), Pot: "100000", Duration: "7", Chances: "100", Ratio: "0.1", TicketPrice: "10", Denom: denom}
	}

	suite.bank.EXPECT().HasSupply(gomock.Any(), denom).Return(true).AnyTimes()
	suite.epoch.EXPECT().SafeGetEpochCountByIdentifier(gomock.Any(), gomock.Any()).Return(int64(100), nil).AnyTimes()
	suite.bank.EXPECT().SpendableCoins(gomock.Any(), creator).Return(sdk.NewCoins()).AnyTimes()
	suite.acc.EXPECT().GetModuleAccount(gomock.Any(), types.RaffleModuleName).Return(&authtypes.ModuleAccount{}).AnyTimes()

	cases := []struct {
		name string
		set  func(*types.MsgStartRaffle)
	}{
		{"pot 0400000", func(m *types.MsgStartRaffle) { m.Pot = "0400000" }},
		{"pot +100000", func(m *types.MsgStartRaffle) { m.Pot = "+100000" }},
		{"pot 0x186a0", func(m *types.MsgStartRaffle) { m.Pot = "0x186a0" }},
		{"pot 100_000", func(m *types.MsgStartRaffle) { m.Pot = "100_000" }},
		{"ticket_price 0100", func(m *types.MsgStartRaffle) { m.TicketPrice = "0100" }},
		{"ticket_price +10", func(m *types.MsgStartRaffle) { m.TicketPrice = "+10" }},
		{"ticket_price 0b10", func(m *types.MsgStartRaffle) { m.TicketPrice = "0b10" }},
		{"duration 010", func(m *types.MsgStartRaffle) { m.Duration = "010" }},
		{"duration 0x10", func(m *types.MsgStartRaffle) { m.Duration = "0x10" }},
		{"chances 0100", func(m *types.MsgStartRaffle) { m.Chances = "0100" }},
		{"chances 1_0", func(m *types.MsgStartRaffle) { m.Chances = "1_0" }},
	}

	for _, c := range cases {
		suite.Run(c.name, func() {
			msg := valid()
			c.set(msg)
			_, err := suite.msgServer.StartRaffle(suite.ctx, msg)
			suite.Require().ErrorContains(err, "canonical", "%s must be refused for its form", c.name)
			_, found := suite.k.GetRaffle(suite.ctx, denom)
			suite.Require().False(found)
		})
	}

	// a valid message goes on to the balance check (the creator has no coins here)
	_, err := suite.msgServer.StartRaffle(suite.ctx, valid())
	suite.Require().ErrorContains(err, "not enough balance")
}
