package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"strconv"
	"strings"
	"time"
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
		Denom:   "aau",
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

func (suite *IntegrationTestSuite) TestJoinRaffle_Stress() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	initialBalanceAmount := sdk.NewInt(1_000_000_000_000_000)
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", initialBalanceAmount.Int64()))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))

	potInt := sdk.NewInt(150_000_000_000)
	ticketCost := sdk.NewInt(10_000_000)
	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         potInt.String(),
		Duration:    "1",
		Chances:     "100",
		Ratio:       "0.1",
		TicketPrice: ticketCost.String(),
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().NoError(err)

	addrBalance := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
	suite.Require().EqualValues(addrBalance.Amount, initialBalanceAmount.Sub(potInt))

	moduleAddress := suite.app.AccountKeeper.GetModuleAddress(types.RaffleModuleName)
	moduleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
	suite.Require().EqualValues(moduleBalance.Amount, potInt)

	ratio, err := sdk.NewDecFromStr(msg.Ratio)
	suite.Require().NoError(err)

	winCount := 0
	ticketsToPlace := time.Now().Unix()
	totalPotWon := sdk.ZeroInt()

	for i := ticketsToPlace; i <= ticketsToPlace+100000; i++ {
		appHash := fmt.Sprintf("%x", i)
		blockHash := []byte("block_id" + appHash)
		suite.ctx = suite.ctx.WithBlockHeader(tmproto.Header{
			LastBlockId: tmproto.BlockID{
				Hash: tmhash.Sum(blockHash),
			},
			AppHash: tmhash.Sum([]byte(appHash)),
			Height:  int64(i),
		}).WithEventManager(sdk.NewEventManager())
		goCtx = sdk.WrapSDKContext(suite.ctx)
		addrBalance = suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
		moduleBalance = suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
		winners := suite.k.GetRaffleWinners(suite.ctx, "aau")
		raffle, ok := suite.k.GetRaffle(suite.ctx, "aau")
		suite.Require().True(ok)
		raffleTotalWon, ok := sdk.NewIntFromString(raffle.TotalWon)
		suite.Require().True(ok)

		join := types.MsgJoinRaffle{
			Creator: addr1.String(),
			Denom:   "aau",
		}

		_, err := suite.msgServer.JoinRaffle(goCtx, &join)
		suite.Require().NoError(err)

		suite.k.WithdrawLuckyRaffleParticipants(suite.ctx, suite.ctx.BlockHeight())

		for _, e := range suite.ctx.EventManager().Events() {
			if strings.Contains(e.Type, "RaffleWinnerEvent") {
				winCount++
				//wonAmount := currentPot.Amount.Sub(ticketPriceInt).ToDec().Mul(winRatio).TruncateInt()
				prize := moduleBalance.Amount.ToDec().Mul(ratio).TruncateInt()
				if !prize.IsPositive() {
					prize = moduleBalance.SubAmount(ticketCost).Amount
				}
				totalPotWon = totalPotWon.Add(prize)
				newBalance := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
				suite.Require().EqualValuesf(newBalance.Amount, addrBalance.Amount.Add(prize).Sub(ticketCost), fmt.Sprintf("expected balance: %s - actual balance: %s", newBalance.String(), addrBalance.Amount.Add(prize).Sub(ticketCost).String()))

				newModuleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
				suite.Require().EqualValues(newModuleBalance.Amount, moduleBalance.Amount.Sub(prize).Add(ticketCost))

				newWinners := suite.k.GetRaffleWinners(suite.ctx, "aau")
				suite.Require().True(len(newWinners) == len(winners)+1 || len(newWinners) == 100)

				//check totalWon
				r, ok := suite.k.GetRaffle(suite.ctx, "aau")
				suite.Require().True(ok)
				rTotalWon, ok := sdk.NewIntFromString(r.TotalWon)
				suite.Require().True(ok)
				suite.Require().EqualValues(rTotalWon, raffleTotalWon.Add(prize))
			} else if strings.Contains(e.Type, "RaffleLostEvent") {

				newBalance := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
				suite.Require().EqualValues(newBalance.Amount, addrBalance.Amount.Sub(ticketCost))

				newModuleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
				suite.Require().EqualValues(newModuleBalance.Amount, moduleBalance.Amount.Add(ticketCost))

				newWinners := suite.k.GetRaffleWinners(suite.ctx, "aau")
				suite.Require().Len(newWinners, len(winners))
			}
		}
	}

	newModuleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
	suite.ctx.Logger().Info(fmt.Sprintf("test finished"))
	suite.ctx.Logger().Info(fmt.Sprintf("%d users won. total participants %d", winCount, 100000))
	suite.ctx.Logger().Info(fmt.Sprintf("total pot won: %s", totalPotWon.QuoRaw(1_000_000).String()))
	suite.ctx.Logger().Info(fmt.Sprintf("module balance is: %s", newModuleBalance.Amount.QuoRaw(1_000_000).String()))
}

func (suite *IntegrationTestSuite) TestJoinRaffle_Simulation() {
	goCtx := sdk.WrapSDKContext(suite.ctx)

	addr1 := sdk.AccAddress("addr1_______________")
	addr2 := sdk.AccAddress("addr2_______________")
	initialBalanceAmount := sdk.NewInt(1_000_000_000_000_000)
	balances := sdk.NewCoins(sdk.NewInt64Coin("aau", initialBalanceAmount.Int64()))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr1, balances))
	suite.Require().NoError(simapp.FundAccount(suite.app.BankKeeper, suite.ctx, addr2, balances))

	potInt := sdk.NewInt(50_000_000_000)
	ticketCost := sdk.NewInt(10_000_000)
	msg := types.MsgStartRaffle{
		Creator:     addr1.String(),
		Pot:         potInt.String(),
		Duration:    "1",
		Chances:     "100",
		Ratio:       "0.15",
		TicketPrice: ticketCost.String(),
		Denom:       "aau",
	}
	_, err := suite.msgServer.StartRaffle(goCtx, &msg)
	suite.Require().NoError(err)

	addrBalance := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
	suite.Require().EqualValues(addrBalance.Amount, initialBalanceAmount.Sub(potInt))

	addr2Balance := suite.app.BankKeeper.GetBalance(suite.ctx, addr2, "aau")
	suite.Require().EqualValues(addr2Balance.Amount, initialBalanceAmount)

	moduleAddress := suite.app.AccountKeeper.GetModuleAddress(types.RaffleModuleName)
	moduleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
	suite.Require().EqualValues(moduleBalance.Amount, potInt)

	ratio, err := sdk.NewDecFromStr(msg.Ratio)
	suite.Require().NoError(err)

	winCount := 0
	ticketsStart := time.Now().Unix()
	ticketsToPlace := int64(10000)
	totalPotWon := sdk.ZeroInt()

	for i := ticketsStart; i < ticketsStart+ticketsToPlace; i++ {
		appHash := fmt.Sprintf("%x", i)
		blockHash := []byte("block_id" + appHash)
		suite.ctx = suite.ctx.WithBlockHeader(tmproto.Header{
			LastBlockId: tmproto.BlockID{
				Hash: tmhash.Sum(blockHash),
			},
			AppHash: tmhash.Sum([]byte(appHash)),
			Height:  i,
		}).WithEventManager(sdk.NewEventManager())

		goCtx = sdk.WrapSDKContext(suite.ctx)
		addrBalance = suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "aau")
		addr2Balance = suite.app.BankKeeper.GetBalance(suite.ctx, addr2, "aau")
		moduleBalance = suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
		winners := suite.k.GetRaffleWinners(suite.ctx, "aau")

		join := types.MsgJoinRaffle{
			Creator: addr1.String(),
			Denom:   "aau",
		}
		join2 := types.MsgJoinRaffle{
			Creator: addr2.String(),
			Denom:   "aau",
		}

		_, err = suite.msgServer.JoinRaffle(goCtx, &join)
		suite.Require().NoError(err)
		_, err = suite.msgServer.JoinRaffle(goCtx, &join2)
		suite.Require().NoError(err)

		suite.k.WithdrawLuckyRaffleParticipants(suite.ctx, suite.ctx.BlockHeight())

		for _, e := range suite.ctx.EventManager().Events() {
			if strings.Contains(e.Type, "RaffleWinnerEvent") {
				winCount++
				//wonAmount := currentPot.Amount.Sub(ticketPriceInt).ToDec().Mul(winRatio).TruncateInt()
				prize := moduleBalance.Amount.ToDec().Mul(ratio).TruncateInt()
				if !prize.IsPositive() {
					prize = moduleBalance.SubAmount(ticketCost).Amount
				}
				totalPotWon = totalPotWon.Add(prize)

				wantedAddress := addr1
				compareAddress := addrBalance
				for _, attr := range e.Attributes {
					if strings.Contains(attr.String(), addr2.String()) {
						wantedAddress = addr2
						compareAddress = addr2Balance
					}
				}
				newBalance := suite.app.BankKeeper.GetBalance(suite.ctx, wantedAddress, "aau")
				suite.Require().True(newBalance.Amount.GT(compareAddress.Amount))

				newWinners := suite.k.GetRaffleWinners(suite.ctx, "aau")
				suite.Require().True(len(newWinners) > len(winners) || len(newWinners) == 100)
			}
		}
	}

	newModuleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddress, "aau")
	suite.ctx.Logger().Info(fmt.Sprintf("test finished"))
	suite.ctx.Logger().Info(fmt.Sprintf("%d users won. total participants %d", winCount, ticketsToPlace))
	suite.ctx.Logger().Info(fmt.Sprintf("total pot won: %s", totalPotWon.QuoRaw(1_000_000).String()))
	suite.ctx.Logger().Info(fmt.Sprintf("module balance is: %s", newModuleBalance.Amount.QuoRaw(1_000_000).String()))
}
