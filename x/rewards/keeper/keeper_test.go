package keeper_test

import (
	"github.com/bze-alphateam/bze/testutil/simapp"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
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
	k         *keeper.Keeper
	msgServer types.MsgServer
}

func (suite *IntegrationTestSuite) SetupTest() {
	app := simapp.Setup(false, true)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	suite.app = app
	suite.ctx = ctx

	suite.k = &app.RewardsKeeper
	suite.msgServer = keeper.NewMsgServerImpl(app.RewardsKeeper)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
