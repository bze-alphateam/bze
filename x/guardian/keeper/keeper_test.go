package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	keepertest "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/guardian/keeper"
	"github.com/bze-alphateam/bze/x/guardian/testutil"
	"github.com/bze-alphateam/bze/x/guardian/types"
)

const (
	testDenom   = "ubze"
	testChannel = "channel-0"
)

var (
	guardianAddr = sdk.AccAddress("guardian_address____").String()
	strangerAddr = sdk.AccAddress("stranger_address_____").String()
	authority    = authtypes.NewModuleAddress(govtypes.ModuleName).String()
)

type env struct {
	ctx       sdk.Context
	k         *keeper.Keeper
	msgServer types.MsgServer
	rl        *testutil.MockRateLimitKeeper
	bank      *testutil.MockBankKeeper
	circuit   *testutil.MockCircuitKeeper
	router    *testutil.MockMsgRouter
}

// newEnv is a real keeper on an in-memory store with a mocked rate-limiting
// keeper, bank keeper, circuit keeper and msg router, and the guardian param
// set.
func newEnv(t *testing.T) *env {
	t.Helper()

	ctrl := gomock.NewController(t)
	bank := testutil.NewMockBankKeeper(ctrl)
	k, ctx := keepertest.GuardianKeeper(t, bank)
	rl := testutil.NewMockRateLimitKeeper(ctrl)
	k.SetRateLimitKeeper(rl)
	circuit := testutil.NewMockCircuitKeeper(ctrl)
	router := testutil.NewMockMsgRouter(ctrl)
	k.SetCircuitKeeper(circuit)
	k.SetMsgRouter(router)
	require.NoError(t, k.SetParams(ctx, types.NewParams(guardianAddr)))

	return &env{ctx: ctx, k: &k, msgServer: keeper.NewMsgServerImpl(&k), rl: rl, bank: bank, circuit: circuit, router: router}
}

// storedLimit is a rate limit with a flow already counted in its window.
func storedLimit(send, recv int64) ratelimittypes.RateLimit {
	return ratelimittypes.RateLimit{
		Path: &ratelimittypes.Path{Denom: testDenom, ChannelOrClientId: testChannel},
		Quota: &ratelimittypes.Quota{
			MaxPercentSend: math.NewInt(send),
			MaxPercentRecv: math.NewInt(recv),
			DurationHours:  24,
		},
		Flow: &ratelimittypes.Flow{
			Inflow:       math.NewInt(123),
			Outflow:      math.NewInt(456),
			ChannelValue: math.NewInt(1_000_000),
		},
	}
}

func tighten(signer string, send, recv int64) *types.MsgTightenRateLimit {
	return &types.MsgTightenRateLimit{
		Signer:         signer,
		Denom:          testDenom,
		ChannelId:      testChannel,
		MaxPercentSend: math.NewInt(send),
		MaxPercentRecv: math.NewInt(recv),
	}
}
