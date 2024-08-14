package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

func (suite *IntegrationTestSuite) TestFundBurner_InvalidAmount() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	msg := types.MsgFundBurner{
		Creator: "",
		Amount:  "-1.23",
	}
	_, err := suite.msgServer.FundBurner(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "coin")
}

func (suite *IntegrationTestSuite) TestFundBurner_InvalidCreator() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	msg := types.MsgFundBurner{
		Creator: "a",
		Amount:  "123ubze",
	}
	_, err := suite.msgServer.FundBurner(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "bech32")
}

func (suite *IntegrationTestSuite) TestFundBurner_NoBalance() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")

	//balances := sdk.NewCoins(newStakeCoin(10000), newBzeCoin(50000))
	//suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgFundBurner{
		Creator: addr1.String(),
		Amount:  "123ubze",
	}
	_, err := suite.msgServer.FundBurner(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "insufficient funds")
}

func (suite *IntegrationTestSuite) TestFundBurner_NotEnoughFunds() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")

	balances := sdk.NewCoins(sdk.NewInt64Coin("ubze", 122))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))

	msg := types.MsgFundBurner{
		Creator: addr1.String(),
		Amount:  "123ubze",
	}
	_, err := suite.msgServer.FundBurner(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "insufficient funds")

	//check module was not funded
	moduleAddress := suite.app.AccountKeeper.GetModuleAddress(types.ModuleName)
	moduleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "ubze")
	suite.Require().Equal(moduleBalance.Amount.String(), "0")
}

func (suite *IntegrationTestSuite) TestFundBurner_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")

	balances := sdk.NewCoins(sdk.NewInt64Coin("ubze", 123))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))

	msg := types.MsgFundBurner{
		Creator: addr1.String(),
		Amount:  "123ubze",
	}
	_, err := suite.msgServer.FundBurner(goCtx, &msg)
	suite.Require().NoError(err)

	accBalance := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "ubze")
	suite.Require().Equal(accBalance.Amount.String(), "0")

	moduleAddress := suite.app.AccountKeeper.GetModuleAddress(types.ModuleName)
	moduleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "ubze")
	suite.Require().Equal(moduleBalance.Amount.String(), "123")
}

func (suite *IntegrationTestSuite) TestStartRaffle_InvalidDenom() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	msg := types.MsgStartRaffle{
		Creator:     "",
		Pot:         "",
		Duration:    "",
		Chances:     "",
		Ratio:       "",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "denom aau does not exist")
}

func (suite *IntegrationTestSuite) TestStartRaffle_RaffleAlreadyExists() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	raffle := types.Raffle{
		Pot:         "",
		Duration:    0,
		Chances:     0,
		Ratio:       "",
		EndAt:       0,
		Winners:     0,
		TicketPrice: "",
		Denom:       "aau",
	}
	suite.k.SetRaffle(suite.ctx, raffle)

	msg := types.MsgStartRaffle{
		Creator:     "",
		Pot:         "",
		Duration:    "",
		Chances:     "",
		Ratio:       "",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "raffle already running for this coin")
}

func (suite *IntegrationTestSuite) TestStartRaffle_InvalidCreator() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     "a",
		Pot:         "",
		Duration:    "",
		Chances:     "",
		Ratio:       "",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "bech32")
}

func (suite *IntegrationTestSuite) TestStartRaffle_InvalidPot() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "wwqdsaca",
		Duration:    "",
		Chances:     "",
		Ratio:       "",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "invalid pot")
}

func (suite *IntegrationTestSuite) TestStartRaffle_NotPositivePot() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "0",
		Duration:    "",
		Chances:     "",
		Ratio:       "",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "provided pot is not positive")
}

func (suite *IntegrationTestSuite) TestStartRaffle_InvalidDuration() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "poweiqj",
		Chances:     "",
		Ratio:       "",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "invalid duration")
}

func (suite *IntegrationTestSuite) TestStartRaffle_NotPositiveDuration() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "0",
		Chances:     "",
		Ratio:       "",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "duration should be positive")

	msg2 := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "-3",
		Chances:     "",
		Ratio:       "",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err2 := suite.msgServer.StartRaffle(goCtx, &msg2)
	suite.Require().Error(err2)
	suite.Require().ErrorContains(err2, "duration should be positive")
}

func (suite *IntegrationTestSuite) TestStartRaffle_OutOfBoundDuration() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "220",
		Chances:     "",
		Ratio:       "",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "duration have a value between")
}

func (suite *IntegrationTestSuite) TestStartRaffle_InvalidRatio() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "",
		Ratio:       "nskadh",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "invalid ratio")
}

func (suite *IntegrationTestSuite) TestStartRaffle_NotPositiveRatio() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "",
		Ratio:       "0",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "ratio is not positive")

	msg = types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "",
		Ratio:       "-0.05",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err = suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "ratio is not positive")
}

func (suite *IntegrationTestSuite) TestStartRaffle_RatioBoundaries() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "",
		Ratio:       "0.001",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "ratio must have a value between")

	msg = types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "",
		Ratio:       "1.0001",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err = suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "ratio must have a value between")
}

func (suite *IntegrationTestSuite) TestStartRaffle_InvalidChances() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "asdfgh",
		Ratio:       "0.1",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "invalid chances provided")
}

func (suite *IntegrationTestSuite) TestStartRaffle_ChancesBoundaries() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "0",
		Ratio:       "0.1",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "chances should have a value between")

	msg = types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "1000001",
		Ratio:       "0.1",
		TicketPrice: "",
		Denom:       "aau",
	}
	_, err = suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "chances should have a value between")
}

func (suite *IntegrationTestSuite) TestStartRaffle_InvalidTicketPrice() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "20",
		Ratio:       "0.1",
		TicketPrice: "sdadsa",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "invalid ticket price provided")
}

func (suite *IntegrationTestSuite) TestStartRaffle_NegativeTicketPrice() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "20",
		Ratio:       "0.1",
		TicketPrice: "-10000002310",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "provided ticket price is not positive")
}

func (suite *IntegrationTestSuite) TestStartRaffle_IbcDenomFailure() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("ibc/aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "109999990",
		Duration:    "15",
		Chances:     "20",
		Ratio:       "0.1",
		TicketPrice: "1000000310",
		Denom:       "ibc/aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "coin not allowed in raffles")
}

func (suite *IntegrationTestSuite) TestStartRaffle_NotEnoughBalance() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundModuleAccount(suite.app.BankKeeper, suite.ctx, types.ModuleName, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "20",
		Ratio:       "0.1",
		TicketPrice: "150000000",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "balance")
}

func (suite *IntegrationTestSuite) TestStartRaffle_Success() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "100",
		Duration:    "15",
		Chances:     "20",
		Ratio:       "0.1",
		TicketPrice: "150000000",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().NoError(err)

	addrBalance := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
	suite.Require().Equal(addrBalance.Amount.String(), "900")

	moduleAddress := suite.app.AccountKeeper.GetModuleAddress(types.RaffleModuleName)
	moduleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
	suite.Require().Equal(moduleBalance.Amount.String(), "100")

	endAt := suite.app.EpochsKeeper.GetEpochCountByIdentifier(suite.ctx, "hour") + (15 * 24)
	storageRaffle, ok := suite.k.GetRaffle(suite.ctx, "aau")
	suite.Require().True(ok)
	suite.Require().EqualValues(msg.Pot, storageRaffle.Pot)
	suite.Require().EqualValues(msg.Duration, strconv.Itoa(int(storageRaffle.Duration)))
	suite.Require().EqualValues(msg.Chances, strconv.Itoa(int(storageRaffle.Chances)))
	suite.Require().EqualValues(msg.Ratio, storageRaffle.Ratio)
	suite.Require().EqualValues(msg.Denom, storageRaffle.Denom)
	suite.Require().EqualValues(msg.TicketPrice, storageRaffle.TicketPrice)
	suite.Require().EqualValues(uint64(endAt), storageRaffle.EndAt)

	deleteHook := suite.k.GetRaffleDeleteHookByEndAtPrefix(suite.ctx, uint64(endAt))
	suite.Require().NotEmpty(deleteHook)
	suite.Require().Equal(deleteHook[0].Denom, "aau")
	suite.Require().EqualValues(deleteHook[0].EndAt, endAt)
}

func (suite *IntegrationTestSuite) TestJoinRaffle_InvalidDenom() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	msg := types.MsgJoinRaffle{
		Creator: "",
		Denom:   "dsa",
	}
	_, err := suite.msgServer.JoinRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "denom")
}

func (suite *IntegrationTestSuite) TestJoinRaffle_RaffleNotFound() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))

	msg := types.MsgJoinRaffle{
		Creator: "",
		Denom:   "dsa",
	}

	_, err := suite.msgServer.JoinRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "raffle")
}

func (suite *IntegrationTestSuite) TestJoinRaffle_InvalidCreator() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	raffle := types.Raffle{
		Pot:         "",
		Duration:    0,
		Chances:     0,
		Ratio:       "",
		EndAt:       0,
		Winners:     0,
		TicketPrice: "",
		Denom:       "aau",
	}

	suite.k.SetRaffle(suite.ctx, raffle)

	msg := types.MsgJoinRaffle{
		Creator: "aa",
		Denom:   "aau",
	}

	_, err := suite.msgServer.JoinRaffle(goCtx, &msg)
	suite.Require().Error(err)
	suite.Require().ErrorContains(err, "bech32")
}

// TODO: increment blocks for this test to actually work
func (suite *IntegrationTestSuite) TestJoinRaffle_Stress() {
	goCtx := sdk.WrapSDKContext(suite.ctx)
	addr1 := sdk.AccAddress("addr1_______________")
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", 1_000_000_000_000))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))

	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         "150_000_000",
		Duration:    "1",
		Chances:     "10000",
		Ratio:       "0.1",
		TicketPrice: "1000000",
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().NoError(err)

	addrBalance := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
	suite.Require().EqualValues(addrBalance.Amount.Int64(), 1_000_000_000_000-150_000_000)

	moduleAddress := suite.app.AccountKeeper.GetModuleAddress(types.RaffleModuleName)
	moduleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
	suite.Require().EqualValues(moduleBalance.Amount.String(), "150000000")

	ratio, err := sdk.NewDecFromStr("0.1")
	suite.Require().NoError(err)
	ticketCost := sdk.NewInt(1000000)

	for i := 1; i <= 100000; i++ {
		addrBalance = suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
		moduleBalance = suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
		winners := suite.k.GetRaffleWinners(suite.ctx, "aau")

		join := types.MsgJoinRaffle{
			Creator: addr1.String(),
			Denom:   "aau",
		}

		response, err := suite.msgServer.JoinRaffle(goCtx, &join)
		suite.Require().NoError(err)

		if response.Winner {
			prize := moduleBalance.Amount.ToDec().Mul(ratio).TruncateInt()
			newBalance := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
			suite.Require().EqualValues(newBalance.Amount, addrBalance.Amount.Add(prize).Sub(ticketCost))

			newModuleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
			suite.Require().EqualValues(newModuleBalance.Amount, moduleBalance.Amount.Sub(prize).Add(ticketCost))

			newWinners := suite.k.GetRaffleWinners(suite.ctx, "aau")
			suite.Require().Len(newWinners, len(winners)+1)
		} else {
			newBalance := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
			suite.Require().EqualValues(newBalance.Amount, addrBalance.Amount.Sub(ticketCost))

			newModuleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
			suite.Require().EqualValues(newModuleBalance.Amount, moduleBalance.Amount.Add(ticketCost))

			newWinners := suite.k.GetRaffleWinners(suite.ctx, "aau")
			suite.Require().Len(newWinners, len(winners))
		}
	}
}
