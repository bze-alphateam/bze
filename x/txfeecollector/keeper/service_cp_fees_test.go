package keeper_test

import (
	"errors"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"

	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/txfeecollector/types"
)

const (
	denomTokenA = "factory/bze1creator/tokena"
	denomTokenB = "factory/bze1creator/tokenb"
	denomIbc    = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	denomLpHash = "0A1B2C3D4E5F60718293A4B5C6D7E8F90A1B2C3D4E5F60718293A4B5C6D7E8F9"
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

// expectCpBalances expects the CP pass to read balances as its first page, from the first denom.
func (suite *IntegrationTestSuite) expectCpBalances(balances sdk.Coins) {
	suite.accountMock.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpModuleAddr).AnyTimes()
	suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), cpModuleAddr, "", 100).Return(balances, "").Times(1)
}

func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_NoBalance() {
	suite.useDenomCpProvider()
	suite.expectCpBalances(sdk.NewCoins())

	// no other expectation: any call on a mock would fail the test
	suite.Require().NoError(suite.k.ConvertCommunityPoolFees(suite.ctx))
	suite.Require().Empty(paidEvents(suite.ctx))
}

func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_CoinWithCpIsSent() {
	suite.useDenomCpProvider()
	coin := sdk.NewInt64Coin(denomTokenA, 1234)
	suite.expectCpBalances(sdk.NewCoins(coin))

	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomTokenA).Return(false).Times(1)
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denomTokenA).Return(tokenACp, true).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenACp, sdk.NewCoins(coin)).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ConvertCommunityPoolFees(suite.ctx))

	suite.Require().Equal([]types.CommunityPoolFeePaidEvent{{
		Denom:     denomTokenA,
		Recipient: tokenACp.String(),
		Amount:    "1234",
	}}, paidEvents(suite.ctx))
}

// One pass decides per coin: a CP denom goes to its CP, ubze and the swapped coins fund the chain's community pool,
// an LP share goes to the burner, and a coin that can't be swapped (too small, no pool) waits in the collector.
func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_EveryRule() {
	suite.useDenomCpProvider()
	tokenA := sdk.NewInt64Coin(denomTokenA, 500)       // has a CP
	tokenB := sdk.NewInt64Coin(denomTokenB, 600)       // no CP, can't be swapped: kept
	ibc := sdk.NewInt64Coin(denomIbc, 700)             // no CP, swappable
	lp := sdk.NewInt64Coin("ulp/"+denomLpHash, 50)     // LP share
	legacyLp := sdk.NewInt64Coin("ulp_ubze_uother", 5) // legacy LP share
	bze := sdk.NewInt64Coin(denomBze, 800)
	suite.expectCpBalances(sdk.NewCoins(tokenA, tokenB, ibc, lp, legacyLp, bze))

	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, denom string) bool { return denom == denomBze }).
		Times(6)
	suite.cpMock.EXPECT().
		GetDenomCommunityPool(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, denom string) (sdk.AccAddress, bool) {
			if denom == denomTokenA {
				return tokenACp, true
			}

			return nil, false
		}).
		Times(5)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenACp, sdk.NewCoins(tokenA)).
		Return(nil).
		Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), ibc).Return(true).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), tokenB).Return(false).Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(ibc)).
		Return(sdk.NewInt64Coin(denomBze, 70), nil).
		Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.CpFeeCollector, burnermoduletypes.ModuleName, sdk.NewCoins(lp, legacyLp)).
		Return(nil).
		Times(1)
	suite.distrMock.EXPECT().
		FundCommunityPool(gomock.Any(), sdk.NewCoins(sdk.NewInt64Coin(denomBze, 870)), cpModuleAddr).
		Return(nil).
		Times(1)

	// no send for token B: a call would fail the test
	suite.Require().NoError(suite.k.ConvertCommunityPoolFees(suite.ctx))
	suite.Require().Len(paidEvents(suite.ctx), 1)
}

// A coin that can't be swapped is never burned: with nothing else in the page, the pass writes nothing.
func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_UnswappableIsKept() {
	coin := sdk.NewInt64Coin(denomStake, 10)
	suite.expectCpBalances(sdk.NewCoins(coin))

	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomStake).Return(false).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), coin).Return(false).Times(1)

	// no swap, burner send or funding expectation: a call would fail the test
	suite.Require().NoError(suite.k.ConvertCommunityPoolFees(suite.ctx))
}

// Without a denom community pool provider, no coin has a CP and the pass still converts the rest.
func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_NilProvider() {
	coin := sdk.NewInt64Coin(denomTokenA, 1000)
	suite.expectCpBalances(sdk.NewCoins(coin))

	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomTokenA).Return(false).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), coin).Return(true).Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(coin)).
		Return(sdk.NewInt64Coin(denomBze, 90), nil).
		Times(1)
	suite.distrMock.EXPECT().
		FundCommunityPool(gomock.Any(), sdk.NewCoins(sdk.NewInt64Coin(denomBze, 90)), cpModuleAddr).
		Return(nil).
		Times(1)

	suite.Require().NoError(suite.k.ConvertCommunityPoolFees(suite.ctx))
}

// A failed send returns the error, stops the pass (the coins after it are not looked at) and emits nothing.
func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_FailedSendStopsPass() {
	suite.useDenomCpProvider()
	tokenA := sdk.NewInt64Coin(denomTokenA, 10)
	tokenB := sdk.NewInt64Coin(denomTokenB, 20)
	suite.expectCpBalances(sdk.NewCoins(tokenA, tokenB))

	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomTokenA).Return(false).Times(1)
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denomTokenA).Return(tokenACp, true).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenACp, sdk.NewCoins(tokenA)).
		Return(errors.New("send failed")).
		Times(1)

	// no expectation for token B: a call would fail the test
	suite.Require().ErrorContains(suite.k.ConvertCommunityPoolFees(suite.ctx), "send failed")
	suite.Require().Empty(paidEvents(suite.ctx))
}

func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_SwapError() {
	coin := sdk.NewInt64Coin(denomStake, 1000)
	suite.expectCpBalances(sdk.NewCoins(coin))

	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomStake).Return(false).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), coin).Return(true).Times(1)
	suite.tradeMock.EXPECT().
		SafeModuleSwapForNativeDenom(gomock.Any(), types.CpFeeCollector, sdk.NewCoins(coin)).
		Return(sdk.Coin{}, sdkerrors.ErrInvalidCoins).
		Times(1)

	// the coin stays in the collector and the pass succeeds: no funding expectation, a call would fail the test
	suite.Require().NoError(suite.k.ConvertCommunityPoolFees(suite.ctx))
}

func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_BurnerSendError() {
	lp := sdk.NewInt64Coin("ulp/"+denomLpHash, 50)
	suite.expectCpBalances(sdk.NewCoins(lp))

	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), lp.Denom).Return(false).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.CpFeeCollector, burnermoduletypes.ModuleName, sdk.NewCoins(lp)).
		Return(sdkerrors.ErrInsufficientFunds).
		Times(1)

	suite.Require().ErrorIs(suite.k.ConvertCommunityPoolFees(suite.ctx), sdkerrors.ErrInsufficientFunds)
}

func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_FundCommunityPoolError() {
	bze := sdk.NewInt64Coin(denomBze, 100)
	suite.expectCpBalances(sdk.NewCoins(bze))

	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), denomBze).Return(true).Times(1)
	suite.distrMock.EXPECT().
		FundCommunityPool(gomock.Any(), sdk.NewCoins(bze), cpModuleAddr).
		Return(sdkerrors.ErrInsufficientFunds).
		Times(1)

	suite.Require().ErrorIs(suite.k.ConvertCommunityPoolFees(suite.ctx), sdkerrors.ErrInsufficientFunds)
}

// A coin whose denom has a community pool and sits past the per-block bound is never burned: the next pass starts
// after the cursor and sends it to its community pool.
func (suite *IntegrationTestSuite) TestConvertCommunityPoolFees_CoinPastTheBoundReachesItsCp() {
	suite.useDenomCpProvider()
	suite.setMaxBalanceIterations(2)
	suite.accountMock.EXPECT().GetModuleAddress(types.CpFeeCollector).Return(cpModuleAddr).AnyTimes()
	keptA, keptB := sdk.NewInt64Coin("aaa", 1), sdk.NewInt64Coin("aab", 2)
	tokenA := sdk.NewInt64Coin(denomTokenA, 30)

	gomock.InOrder(
		suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), cpModuleAddr, "", 2).Return(sdk.NewCoins(keptA, keptB), "aab").Times(1),
		suite.bankMock.EXPECT().GetBalancesPage(gomock.Any(), cpModuleAddr, "aab", 2).Return(sdk.NewCoins(tokenA), denomTokenA).Times(1),
	)
	suite.tradeMock.EXPECT().IsNativeDenom(gomock.Any(), gomock.Any()).Return(false).Times(3)
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), "aaa").Return(nil, false).Times(1)
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), "aab").Return(nil, false).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), keptA).Return(false).Times(1)
	suite.tradeMock.EXPECT().CanSwapForNativeDenom(gomock.Any(), keptB).Return(false).Times(1)
	suite.cpMock.EXPECT().GetDenomCommunityPool(gomock.Any(), denomTokenA).Return(tokenACp, true).Times(1)
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.CpFeeCollector, tokenACp, sdk.NewCoins(tokenA)).
		Return(nil).
		Times(1)

	// no burner expectation: a call would fail the test
	suite.Require().NoError(suite.k.ConvertCommunityPoolFees(suite.ctx))
	suite.Require().Empty(paidEvents(suite.ctx))
	suite.Require().NoError(suite.k.ConvertCommunityPoolFees(suite.ctx))
	suite.Require().Len(paidEvents(suite.ctx), 1)
}
