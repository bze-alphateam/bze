package keeper_test

import (
	"errors"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"

	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

const (
	denomTokenA = "factory/bze1creator/tokena"
	denomTokenB = "factory/bze1creator/tokenb"
	denomIbc    = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
)

var (
	cpModuleAddr = sdk.AccAddress("txfeecollector_cp___")
	tokenACp     = sdk.AccAddress("token_a_cp__________")
)

// paidEvents returns the CommunityPoolFeePaidEvents emitted on ctx.
func paidEvents(ctx sdk.Context) []types.CommunityPoolFeePaidEvent {
	var out []types.CommunityPoolFeePaidEvent
	for _, e := range ctx.EventManager().Events() {
		if e.Type != proto.MessageName(&types.CommunityPoolFeePaidEvent{}) {
			continue
		}
		msg, err := sdk.ParseTypedEvent(abci.Event(e))
		if err != nil {
			panic(err)
		}
		out = append(out, *msg.(*types.CommunityPoolFeePaidEvent))
	}

	return out
}

func (suite *IntegrationTestSuite) expectCpBalances(balances sdk.Coins) {
	suite.accountMock.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpModuleAddr).Times(1)
	suite.bankMock.EXPECT().GetAllBalances(gomock.Any(), cpModuleAddr).Return(balances).Times(1)
}

func (suite *IntegrationTestSuite) TestRouteCommunityPoolFees_NilProviderDoesNothing() {
	// no expectations: any call on a mock would fail the test
	suite.Require().NoError(suite.k.RouteCommunityPoolFeesToDenomCommunityPools(suite.ctx))
	suite.Require().Empty(paidEvents(suite.ctx))
}

func (suite *IntegrationTestSuite) TestRouteCommunityPoolFees_NoBalance() {
	suite.useDenomCpProvider()
	suite.expectCpBalances(sdk.NewCoins())

	suite.Require().NoError(suite.k.RouteCommunityPoolFeesToDenomCommunityPools(suite.ctx))
	suite.Require().Empty(paidEvents(suite.ctx))
}

func (suite *IntegrationTestSuite) TestRouteCommunityPoolFees_CoinWithCpIsSent() {
	suite.useDenomCpProvider()
	coin := sdk.NewInt64Coin(denomTokenA, 1234)
	suite.expectCpBalances(sdk.NewCoins(coin))

	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denomTokenA).Return(tokenACp, true).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenACp, sdk.NewCoins(coin)).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.RouteCommunityPoolFeesToDenomCommunityPools(suite.ctx))

	suite.Require().Equal([]types.CommunityPoolFeePaidEvent{{
		Denom:     denomTokenA,
		Recipient: tokenACp.String(),
		Amount:    "1234",
	}}, paidEvents(suite.ctx))
}

// Only the coins whose denom has a community pool are forwarded; native, IBC and factory coins without one are left
// to the conversion, which then sees only them.
func (suite *IntegrationTestSuite) TestRouteCommunityPoolFees_MixedDenoms() {
	suite.useDenomCpProvider()
	tokenA := sdk.NewInt64Coin(denomTokenA, 500)
	tokenB := sdk.NewInt64Coin(denomTokenB, 600)
	ibc := sdk.NewInt64Coin(denomIbc, 700)
	bze := sdk.NewInt64Coin(denomBze, 800)
	suite.expectCpBalances(sdk.NewCoins(tokenA, tokenB, ibc, bze))

	suite.cpMock.EXPECT().
		GetDenomCommunityPool(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, denom string) (sdk.AccAddress, bool) {
			if denom == denomTokenA {
				return tokenACp, true
			}

			return nil, false
		}).
		Times(4)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenACp, sdk.NewCoins(tokenA)).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.RouteCommunityPoolFeesToDenomCommunityPools(suite.ctx))
	suite.Require().Len(paidEvents(suite.ctx), 1)

	// the conversion that follows in the same EndBlock handles what is left as before
	suite.expectCpBalances(sdk.NewCoins(tokenB, ibc, bze))
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, denom string) bool { return denom == denomBze }).
		Times(3)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibc).Return(true).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), tokenB).Return(false).Times(1)
	suite.tradeMock.EXPECT().HasDeepLiquidityWithNativeDenom(gomock.Any(), denomTokenB).Return(false).Times(1)
	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(ibc)).
		Return(sdk.NewInt64Coin(denomBze, 70), nil).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.CpFeeCollector, burnermoduletypes.ModuleName, sdk.NewCoins(tokenB)).
		Return(nil).
		Times(1)
	suite.accountMock.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpModuleAddr).Times(1)
	suite.distrMock.EXPECT().
		FundCommunityPool(gomock.Any(), sdk.NewCoins(sdk.NewInt64Coin(denomBze, 870)), cpModuleAddr).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ConvertCommunityPoolFeesToNativeDenom(suite.ctx))
}

// A failed send returns the error and emits nothing; the coin is still in the collector (EndBlock discards the
// routing step), so the conversion takes it as before.
func (suite *IntegrationTestSuite) TestRouteCommunityPoolFees_FailedSendReturnsError() {
	suite.useDenomCpProvider()
	coin := sdk.NewInt64Coin(denomTokenA, 1000)
	suite.expectCpBalances(sdk.NewCoins(coin))

	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denomTokenA).Return(tokenACp, true).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenACp, sdk.NewCoins(coin)).
		Return(errors.New("send failed")).
		Times(1)

	err := suite.k.RouteCommunityPoolFeesToDenomCommunityPools(suite.ctx)
	suite.Require().ErrorContains(err, "send failed")
	suite.Require().Empty(paidEvents(suite.ctx))

	// today's path takes the coin: swapped for the chain's community pool
	suite.expectCpBalances(sdk.NewCoins(coin))
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomTokenA).Return(false).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), coin).Return(true).Times(1)
	suite.tradeMock.EXPECT().
		ModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(coin)).
		Return(sdk.NewInt64Coin(denomBze, 90), nil).
		Times(1)
	suite.accountMock.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpModuleAddr).Times(1)
	suite.distrMock.EXPECT().
		FundCommunityPool(gomock.Any(), sdk.NewCoins(sdk.NewInt64Coin(denomBze, 90)), cpModuleAddr).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ConvertCommunityPoolFeesToNativeDenom(suite.ctx))
}

// The first failed send stops the routing: the coins after it are not looked at.
func (suite *IntegrationTestSuite) TestRouteCommunityPoolFees_FailedSendStopsRouting() {
	suite.useDenomCpProvider()
	tokenA := sdk.NewInt64Coin(denomTokenA, 10)
	tokenB := sdk.NewInt64Coin(denomTokenB, 20)
	suite.expectCpBalances(sdk.NewCoins(tokenA, tokenB))

	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denomTokenA).Return(tokenACp, true).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenACp, sdk.NewCoins(tokenA)).
		Return(errors.New("send failed")).
		Times(1)

	// no expectation for token B: a call would fail the test
	suite.Require().Error(suite.k.RouteCommunityPoolFeesToDenomCommunityPools(suite.ctx))
	suite.Require().Empty(paidEvents(suite.ctx))
}

// The routing looks at no more than MaxBalanceIterations coins per block; a coin over the bound is routed in a later
// block, once the coins before it have left the collector.
func (suite *IntegrationTestSuite) TestRouteCommunityPoolFees_IterationBound() {
	suite.useDenomCpProvider()
	params := suite.k.GetParams(suite.ctx)
	params.MaxBalanceIterations = 2
	suite.Require().NoError(suite.k.SetParams(suite.ctx, params))

	aaa := sdk.NewInt64Coin("aaa", 1)
	aab := sdk.NewInt64Coin("aab", 2)
	tokenA := sdk.NewInt64Coin(denomTokenA, 3)
	bze := sdk.NewInt64Coin(denomBze, 4)

	// block 1: sorted aaa, aab, token A, ubze; token A is past the bound and not looked at
	suite.expectCpBalances(sdk.NewCoins(aaa, aab, tokenA, bze))
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), "aaa").Return(nil, false).Times(1)
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), "aab").Return(nil, false).Times(1)
	suite.Require().NoError(suite.k.RouteCommunityPoolFeesToDenomCommunityPools(suite.ctx))
	suite.Require().Empty(paidEvents(suite.ctx))

	// block 2: aaa and aab were converted, token A is now within the bound
	suite.expectCpBalances(sdk.NewCoins(tokenA, bze))
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denomTokenA).Return(tokenACp, true).Times(1)
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denomBze).Return(nil, false).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenACp, sdk.NewCoins(tokenA)).
		Return(nil).
		Times(1)
	suite.Require().NoError(suite.k.RouteCommunityPoolFeesToDenomCommunityPools(suite.ctx))
	suite.Require().Len(paidEvents(suite.ctx), 1)
}
