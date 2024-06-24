package keeper_test

import (
	"fmt"
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/tradebin/keeper"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"testing"
	"time"
)

const (
	denomBze   = "ubze"
	denomStake = "stake"
)

var market = types.Market{
	Base:    denomStake,
	Quote:   denomBze,
	Creator: "bze1m33n82r5x3eyjmjtwjkl82zzdlrnv8pevd8u9r",
}

func getMarketId() string {
	return fmt.Sprintf("%s/%s", market.Base, market.Quote)
}

// TODO write test for multiple orders in the same aggregated price and check the history resulting upon order fill is correct
type IntegrationTestSuite struct {
	suite.Suite

	app       *simapp.SimApp
	ctx       sdk.Context
	k         *keeper.Keeper
	msgServer types.MsgServer
}

func (suite *IntegrationTestSuite) SetupTest() {
	app := simapp.Setup(false, true)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	suite.app = app
	suite.ctx = ctx

	suite.k = &app.TradebinKeeper
	suite.msgServer = keeper.NewMsgServerImpl(app.TradebinKeeper)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
