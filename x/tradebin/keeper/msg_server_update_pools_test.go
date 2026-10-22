package keeper_test

import (
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"

	burnermoduletypes "github.com/bze-alphateam/bze/x/burner/types"
	"github.com/bze-alphateam/bze/x/tradebin/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
)

const (
	updPoolId    = "stake_ubze"
	updPoolOther = "uhalt_ubze"
)

func updFeeDest(treasury, burner, providers string) types.FeeDestination {
	return types.FeeDestination{
		Treasury:  math.LegacyMustNewDecFromStr(treasury),
		Burner:    math.LegacyMustNewDecFromStr(burner),
		Providers: math.LegacyMustNewDecFromStr(providers),
	}
}

func (suite *IntegrationTestSuite) seedUpdatablePool(id, base, quote string) types.LiquidityPool {
	dest := updFeeDest("0.3", "0.3", "0.4")
	pool := types.LiquidityPool{
		Id:           id,
		Base:         base,
		Quote:        quote,
		LpDenom:      "ulp_" + id,
		Creator:      getTestAddress(),
		Fee:          math.LegacyMustNewDecFromStr("0.003"),
		FeeDest:      &dest,
		ReserveBase:  math.NewInt(1_000_000_000),
		ReserveQuote: math.NewInt(2_000_000_000),
		Stable:       false,
	}
	suite.k.SetLiquidityPool(suite.ctx, pool)

	return pool
}

func (suite *IntegrationTestSuite) poolFeesUpdatedEvents() []types.PoolFeesUpdatedEvent {
	var out []types.PoolFeesUpdatedEvent
	for _, ev := range suite.ctx.EventManager().Events() {
		parsed, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		if e, ok := parsed.(*types.PoolFeesUpdatedEvent); ok {
			out = append(out, *e)
		}
	}

	return out
}

func (suite *IntegrationTestSuite) TestMsgUpdatePools_UpdatesFeeAndSplitOnly() {
	before := suite.seedUpdatablePool(updPoolId, denomStake, denomBze)
	other := suite.seedUpdatablePool(updPoolOther, denomHalted, denomBze)

	newDest := updFeeDest("0.5", "0.25", "0.25")
	msg := types.NewMsgUpdatePools(haltAuthority(), []types.PoolUpdate{
		{PoolId: updPoolId, Fee: math.LegacyMustNewDecFromStr("0.01"), FeeDest: newDest},
	})

	resp, err := suite.msgServer.UpdatePools(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)

	after, found := suite.k.GetLiquidityPool(suite.ctx, updPoolId)
	suite.Require().True(found)
	suite.Require().True(after.Fee.Equal(math.LegacyMustNewDecFromStr("0.01")))
	suite.Require().True(after.FeeDest.Treasury.Equal(newDest.Treasury))
	suite.Require().True(after.FeeDest.Burner.Equal(newDest.Burner))
	suite.Require().True(after.FeeDest.Providers.Equal(newDest.Providers))

	// every other field is untouched
	expected := before
	expected.Fee = after.Fee
	expected.FeeDest = after.FeeDest
	suite.Require().Equal(expected, after)

	// a pool missing from the batch is untouched
	otherAfter, _ := suite.k.GetLiquidityPool(suite.ctx, updPoolOther)
	suite.Require().Equal(other, otherAfter)

	events := suite.poolFeesUpdatedEvents()
	suite.Require().Len(events, 1)
	suite.Require().Equal(updPoolId, events[0].PoolId)
	suite.Require().Equal(before.Fee.String(), events[0].OldFee)
	suite.Require().Equal(after.Fee.String(), events[0].NewFee)
	suite.Require().True(events[0].OldFeeDest.Treasury.Equal(before.FeeDest.Treasury))
	suite.Require().True(events[0].OldFeeDest.Burner.Equal(before.FeeDest.Burner))
	suite.Require().True(events[0].OldFeeDest.Providers.Equal(before.FeeDest.Providers))
	suite.Require().True(events[0].NewFeeDest.Treasury.Equal(newDest.Treasury))
	suite.Require().True(events[0].NewFeeDest.Burner.Equal(newDest.Burner))
	suite.Require().True(events[0].NewFeeDest.Providers.Equal(newDest.Providers))
}

func (suite *IntegrationTestSuite) TestMsgUpdatePools_InvalidAuthorityRejected() {
	before := suite.seedUpdatablePool(updPoolId, denomStake, denomBze)
	msg := types.NewMsgUpdatePools(getTestAddress(), []types.PoolUpdate{
		{PoolId: updPoolId, Fee: math.LegacyMustNewDecFromStr("0.01"), FeeDest: updFeeDest("0.5", "0.25", "0.25")},
	})

	_, err := suite.msgServer.UpdatePools(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidSigner)

	after, _ := suite.k.GetLiquidityPool(suite.ctx, updPoolId)
	suite.Require().Equal(before, after)
	suite.Require().Zero(suite.countTypedEvents(&types.PoolFeesUpdatedEvent{}))
}

func (suite *IntegrationTestSuite) TestMsgUpdatePools_ValidationRejected() {
	before := suite.seedUpdatablePool(updPoolId, denomStake, denomBze)
	valid := types.PoolUpdate{PoolId: updPoolId, Fee: math.LegacyMustNewDecFromStr("0.01"), FeeDest: updFeeDest("0.5", "0.25", "0.25")}
	tooMany := make([]types.PoolUpdate, types.MaxUpdatesPerUpdatePoolsMsg+1)
	for i := range tooMany {
		tooMany[i] = valid
		tooMany[i].PoolId = updPoolId + string(rune('a'+i%26)) + string(rune('a'+i/26))
	}

	cases := []struct {
		name    string
		updates []types.PoolUpdate
		wantErr error
	}{
		{"empty", nil, sdkerrors.ErrInvalidRequest},
		{"too many", tooMany, sdkerrors.ErrInvalidRequest},
		{"duplicate", []types.PoolUpdate{valid, valid}, sdkerrors.ErrInvalidRequest},
		{"fee below min", []types.PoolUpdate{{PoolId: updPoolId, Fee: math.LegacyMustNewDecFromStr("0.0009"), FeeDest: valid.FeeDest}}, sdkerrors.ErrInvalidCoins},
		{"fee above max", []types.PoolUpdate{{PoolId: updPoolId, Fee: math.LegacyMustNewDecFromStr("0.06"), FeeDest: valid.FeeDest}}, sdkerrors.ErrInvalidCoins},
		{"zero fee", []types.PoolUpdate{{PoolId: updPoolId, Fee: math.LegacyZeroDec(), FeeDest: valid.FeeDest}}, sdkerrors.ErrInvalidCoins},
		{"negative share", []types.PoolUpdate{{PoolId: updPoolId, Fee: valid.Fee, FeeDest: updFeeDest("-0.1", "0.5", "0.6")}}, types.ErrNegativeFeeDestination},
		{"sum not one", []types.PoolUpdate{{PoolId: updPoolId, Fee: valid.Fee, FeeDest: updFeeDest("0.5", "0.5", "0.5")}}, types.ErrInvalidFeeDestination},
	}
	for _, tc := range cases {
		suite.Run(tc.name, func() {
			_, err := suite.msgServer.UpdatePools(suite.ctx, types.NewMsgUpdatePools(haltAuthority(), tc.updates))
			suite.Require().ErrorIs(err, tc.wantErr)

			after, _ := suite.k.GetLiquidityPool(suite.ctx, updPoolId)
			suite.Require().Equal(before, after)
		})
	}
	suite.Require().Zero(suite.countTypedEvents(&types.PoolFeesUpdatedEvent{}))
}

// An unknown pool anywhere in the batch refuses it whole: the valid entry before it is not written.
func (suite *IntegrationTestSuite) TestMsgUpdatePools_UnknownPoolRefusesBatch() {
	before := suite.seedUpdatablePool(updPoolId, denomStake, denomBze)
	msg := types.NewMsgUpdatePools(haltAuthority(), []types.PoolUpdate{
		{PoolId: updPoolId, Fee: math.LegacyMustNewDecFromStr("0.01"), FeeDest: updFeeDest("0.5", "0.25", "0.25")},
		{PoolId: "nope_ubze", Fee: math.LegacyMustNewDecFromStr("0.01"), FeeDest: updFeeDest("0.5", "0.25", "0.25")},
	})

	_, err := suite.msgServer.UpdatePools(suite.ctx, msg)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
	suite.Require().Contains(err.Error(), "nope_ubze")

	after, _ := suite.k.GetLiquidityPool(suite.ctx, updPoolId)
	suite.Require().Equal(before, after)
	suite.Require().Zero(suite.countTypedEvents(&types.PoolFeesUpdatedEvent{}))
	_, found := suite.k.GetLiquidityPool(suite.ctx, "nope_ubze")
	suite.Require().False(found, "an unknown pool is never created")
}

// The halt only blocks trading: governance can still change a halted pool's fees.
func (suite *IntegrationTestSuite) TestMsgUpdatePools_HaltedPoolUpdated() {
	suite.seedUpdatablePool(updPoolOther, denomHalted, denomBze)
	suite.k.SetHaltedDenom(suite.ctx, denomHalted)

	msg := types.NewMsgUpdatePools(haltAuthority(), []types.PoolUpdate{
		{PoolId: updPoolOther, Fee: math.LegacyMustNewDecFromStr("0.02"), FeeDest: updFeeDest("0", "1", "0")},
	})
	_, err := suite.msgServer.UpdatePools(suite.ctx, msg)
	suite.Require().NoError(err)

	after, _ := suite.k.GetLiquidityPool(suite.ctx, updPoolOther)
	suite.Require().True(after.Fee.Equal(math.LegacyMustNewDecFromStr("0.02")))
	suite.Require().True(after.FeeDest.Burner.Equal(math.LegacyOneDec()))
	suite.Require().True(suite.k.IsDenomHalted(suite.ctx, denomHalted), "the halt itself is untouched")
	suite.Require().Equal(1, suite.countTypedEvents(&types.PoolFeesUpdatedEvent{}))
}

// An entry equal to the pool's current values is accepted, writes nothing and emits no event; the
// changed entry of the same batch still emits its event.
func (suite *IntegrationTestSuite) TestMsgUpdatePools_NoOpEntrySilent() {
	same := suite.seedUpdatablePool(updPoolId, denomStake, denomBze)
	suite.seedUpdatablePool(updPoolOther, denomHalted, denomBze)

	msg := types.NewMsgUpdatePools(haltAuthority(), []types.PoolUpdate{
		// same values, written differently ("0.0030" == 0.003)
		{PoolId: updPoolId, Fee: math.LegacyMustNewDecFromStr("0.0030"), FeeDest: updFeeDest("0.30", "0.3", "0.4")},
		{PoolId: updPoolOther, Fee: math.LegacyMustNewDecFromStr("0.004"), FeeDest: *same.FeeDest},
	})
	_, err := suite.msgServer.UpdatePools(suite.ctx, msg)
	suite.Require().NoError(err)

	after, _ := suite.k.GetLiquidityPool(suite.ctx, updPoolId)
	suite.Require().Equal(same, after)

	events := suite.poolFeesUpdatedEvents()
	suite.Require().Len(events, 1)
	suite.Require().Equal(updPoolOther, events[0].PoolId)
	suite.Require().Equal(math.LegacyMustNewDecFromStr("0.003").String(), events[0].OldFee)
	suite.Require().Equal(math.LegacyMustNewDecFromStr("0.004").String(), events[0].NewFee)

	// running the same proposal again changes nothing and emits nothing more
	_, err = suite.msgServer.UpdatePools(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().Len(suite.poolFeesUpdatedEvents(), 1)
}

// The next swap after an update charges the new fee and splits it with the new destinations.
func (suite *IntegrationTestSuite) TestMsgUpdatePools_SwapUsesNewFeeAndSplit() {
	suite.seedUpdatablePool(updPoolId, denomStake, denomBze)
	_, err := suite.msgServer.UpdatePools(suite.ctx, types.NewMsgUpdatePools(haltAuthority(), []types.PoolUpdate{
		{PoolId: updPoolId, Fee: math.LegacyMustNewDecFromStr("0.01"), FeeDest: updFeeDest("0.5", "0.25", "0.25")},
	}))
	suite.Require().NoError(err)

	trader := sdk.AccAddress("trader______________")
	input := sdk.NewCoin(denomStake, math.NewInt(1_000_000))

	// capture where the swap fee (paid in the input denom) goes; the taker fee is in ubze
	sent := map[string]math.Int{}
	suite.bankMock.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), trader, types.ModuleName, gomock.Any()).Return(nil).AnyTimes()
	suite.bankMock.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, trader, gomock.Any()).Return(nil).AnyTimes()
	suite.bankMock.EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, _ string, to string, coins sdk.Coins) error {
			if amt := coins.AmountOf(denomStake); amt.IsPositive() {
				prev, ok := sent[to]
				if !ok {
					prev = math.ZeroInt()
				}
				sent[to] = prev.Add(amt)
			}
			return nil
		}).AnyTimes()
	suite.accountMock.EXPECT().GetModuleAccount(gomock.Any(), types.ModuleName).
		Return(authtypes.NewEmptyModuleAccount(types.ModuleName)).AnyTimes()

	_, err = suite.msgServer.MultiSwap(suite.ctx, &types.MsgMultiSwap{
		Creator:   trader.String(),
		Routes:    []string{updPoolId},
		Input:     input,
		MinOutput: sdk.NewCoin(denomBze, math.OneInt()),
	})
	suite.Require().NoError(err)

	// fee = 1% of 1,000,000 = 10,000 stake: 50% treasury, 25% burner, 25% stays with the providers
	suite.Require().Equal(math.NewInt(5_000), sent[txfeecollectortypes.CpFeeCollector])
	suite.Require().Equal(math.NewInt(2_500), sent[burnermoduletypes.ModuleName])

	after, _ := suite.k.GetLiquidityPool(suite.ctx, updPoolId)
	suite.Require().Equal(math.NewInt(1_000_000_000+1_000_000-7_500), after.ReserveBase)
}
