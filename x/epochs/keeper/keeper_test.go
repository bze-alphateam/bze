package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/epochs/keeper"
	"github.com/bze-alphateam/bze/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"testing"
	"time"
)

type IntegrationTestSuite struct {
	suite.Suite

	app       *simapp.SimApp
	ctx       sdk.Context
	keeper    *keeper.Keeper
	msgServer types.MsgServer
}

func (suite *IntegrationTestSuite) SetupTest() {
	app := simapp.Setup(false, false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	suite.app = app
	suite.ctx = ctx

	suite.keeper = &app.EpochsKeeper
	suite.msgServer = keeper.NewMsgServerImpl(app.EpochsKeeper)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
