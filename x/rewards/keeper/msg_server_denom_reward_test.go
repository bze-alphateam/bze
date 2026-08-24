package keeper_test

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"
	keeper2 "github.com/bze-alphateam/bze/testutil/keeper"
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/testutil"
	"github.com/bze-alphateam/bze/x/rewards/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/mock/gomock"
)

// --- event helpers ---

func (suite *IntegrationTestSuite) findTypedEvent(evType string) (sdk.Event, bool) {
	for _, e := range suite.ctx.EventManager().Events() {
		if e.Type == evType {
			return e, true
		}
	}

	return sdk.Event{}, false
}

// requireEventAttr asserts a typed-event attribute equals the expected value. Typed-event attribute
// values are JSON-encoded, so a string field arrives wrapped in quotes — they are trimmed here.
func (suite *IntegrationTestSuite) requireEventAttr(e sdk.Event, key, expected string) {
	for _, a := range e.Attributes {
		if a.Key == key {
			suite.Require().Equal(expected, strings.Trim(a.Value, "\""))
			return
		}
	}

	suite.Require().Failf("attribute missing", "event %s has no attribute %q", e.Type, key)
}

// --- CreateDenomReward ---

// Happy path: the denom is validated, the exact creation fee is captured-and-swapped and forwarded to
// the fee collector, the DR is written with lock/min_stake snapshotted from params and staked_amount
// "0", no creator is stored, and the create event is emitted.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_Success_ExactFee() {
	creator := sdk.AccAddress("dr-creator-01")
	fee := sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(25000)))

	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
		CreateDenomRewardFee: sdk.NewCoin("ubze", math.NewInt(25000)),
		DenomRewardLock:      7,
		DenomRewardMinStake:  100,
	}))

	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1_000_000)))).Times(1)
	// exact fee flows through capture-and-swap, then module->module to the fee collector
	suite.trade.EXPECT().
		CaptureAndSwapUserFee(suite.ctx, creator, fee, types.ModuleName).
		Return(fee, nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToModule(suite.ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, fee).
		Return(nil).Times(1)

	msg := types.NewMsgCreateDenomReward(creator.String(), "ubze")
	res, err := suite.msgServer.CreateDenomReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	dr, found := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().True(found)
	suite.Require().Equal("ubze", dr.StakingDenom)
	suite.Require().Equal(uint32(7), dr.Lock)
	suite.Require().Equal(uint64(100), dr.MinStake)
	suite.Require().Equal("0", dr.StakedAmount.String())

	e, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardCreateEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "denom", "ubze")
}

// When the creation fee param is zero the creation is free: no balance check, no capture, no fee
// collector transfer — but the DR is still created with the snapshotted config.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_FreeWhenFeeZero() {
	creator := sdk.AccAddress("dr-creator-02")

	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
		CreateDenomRewardFee: sdk.NewCoin("ubze", math.NewInt(0)),
		DenomRewardLock:      7,
		DenomRewardMinStake:  0,
	}))

	// only HasSupply is expected; any capture/transfer call would be an unexpected mock call and panic
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)

	msg := types.NewMsgCreateDenomReward(creator.String(), "ubze")
	res, err := suite.msgServer.CreateDenomReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	dr, found := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().True(found)
	suite.Require().Equal(uint32(7), dr.Lock)
	suite.Require().Equal("0", dr.StakedAmount.String())
}

// One DR per denom, chain-wide: creating a DR for a denom that already has one is rejected.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_DuplicateRejected() {
	creator := sdk.AccAddress("dr-creator-03")

	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
		CreateDenomRewardFee: sdk.NewCoin("ubze", math.NewInt(0)),
	}))
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})

	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)

	msg := types.NewMsgCreateDenomReward(creator.String(), "ubze")
	res, err := suite.msgServer.CreateDenomReward(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrDenomRewardExists)
	suite.Require().Nil(res)
}

// A denom with no supply cannot host a DR.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_NoSupplyRejected() {
	creator := sdk.AccAddress("dr-creator-04")

	suite.bank.EXPECT().HasSupply(suite.ctx, "nosupply").Return(false).Times(1)

	msg := types.NewMsgCreateDenomReward(creator.String(), "nosupply")
	res, err := suite.msgServer.CreateDenomReward(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrInvalidStakingDenom)
	suite.Require().Nil(res)

	_, found := suite.k.GetDenomReward(suite.ctx, "nosupply")
	suite.Require().False(found)
}

func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_NilRequest() {
	res, err := suite.msgServer.CreateDenomReward(suite.ctx, nil)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_InvalidCreator() {
	msg := &types.MsgCreateDenomReward{Creator: "not-a-bech32", Denom: "ubze"}
	res, err := suite.msgServer.CreateDenomReward(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Nil(res)
}

// A creator who cannot cover the fee is rejected before any state is written.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_InsufficientFunds() {
	creator := sdk.AccAddress("dr-creator-05")

	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
		CreateDenomRewardFee: sdk.NewCoin("ubze", math.NewInt(25000)),
	}))

	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)
	suite.bank.EXPECT().SpendableCoins(suite.ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1)))).Times(1)

	msg := types.NewMsgCreateDenomReward(creator.String(), "ubze")
	res, err := suite.msgServer.CreateDenomReward(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "user balance is too low")
	suite.Require().Nil(res)

	_, found := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().False(found)
}

// With a positive fee but no trade keeper wired, creation cannot capture the fee and is rejected.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_NilTradeKeeper() {
	t := suite.T()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBank := testutil.NewMockBankKeeper(mockCtrl)
	mockEpoch := testutil.NewMockEpochKeeper(mockCtrl)
	mockAcc := testutil.NewMockAccountKeeper(mockCtrl)

	k, ctx := keeper2.RewardsKeeper(t, mockBank, mockEpoch, nil, mockAcc)
	msgServer := keeper.NewMsgServerImpl(k)

	creator := sdk.AccAddress("dr-creator-06")
	suite.Require().NoError(k.SetParams(ctx, types.Params{
		CreateDenomRewardFee: sdk.NewCoin("ubze", math.NewInt(25000)),
	}))

	mockBank.EXPECT().HasSupply(ctx, "ubze").Return(true).Times(1)
	mockBank.EXPECT().SpendableCoins(ctx, creator).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1_000_000)))).Times(1)

	msg := types.NewMsgCreateDenomReward(creator.String(), "ubze")
	res, err := msgServer.CreateDenomReward(ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "trade keeper is not available")
	suite.Require().Nil(res)
}

// Param snapshot (Business Logic rule 3): lock/min_stake are frozen on the DR at creation. Changing
// module params afterwards leaves an existing DR untouched, while a newly created DR picks up the new
// values.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_CreateDenomReward_ParamSnapshot() {
	creator := sdk.AccAddress("dr-creator-07")

	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
		CreateDenomRewardFee: sdk.NewCoin("ubze", math.NewInt(0)),
		DenomRewardLock:      7,
		DenomRewardMinStake:  0,
	}))
	suite.bank.EXPECT().HasSupply(suite.ctx, "ubze").Return(true).Times(1)
	_, err := suite.msgServer.CreateDenomReward(suite.ctx, types.NewMsgCreateDenomReward(creator.String(), "ubze"))
	suite.Require().NoError(err)

	// change the template params
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{
		CreateDenomRewardFee: sdk.NewCoin("ubze", math.NewInt(0)),
		DenomRewardLock:      14,
		DenomRewardMinStake:  500,
	}))
	suite.bank.EXPECT().HasSupply(suite.ctx, "uother").Return(true).Times(1)
	_, err = suite.msgServer.CreateDenomReward(suite.ctx, types.NewMsgCreateDenomReward(creator.String(), "uother"))
	suite.Require().NoError(err)

	// the pre-existing DR kept its original snapshot
	first, found := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().True(found)
	suite.Require().Equal(uint32(7), first.Lock)
	suite.Require().Equal(uint64(0), first.MinStake)

	// the new DR got the new values
	second, found := suite.k.GetDenomReward(suite.ctx, "uother")
	suite.Require().True(found)
	suite.Require().Equal(uint32(14), second.Lock)
	suite.Require().Equal(uint64(500), second.MinStake)
}

// --- JoinDenomReward ---

// A first join creates the participant (and its address-first marker), escrows the stake, grows the
// DR's staked total and emits the join event with the added amount.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_FreshJoinSuccess() {
	joiner := sdk.AccAddress("dr-joiner-01")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 100, StakedAmount: math.NewInt(0)})

	suite.bank.EXPECT().SpendableCoins(suite.ctx, joiner).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10_000)))).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, joiner, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).Times(1)

	msg := types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(500))
	res, err := suite.msgServer.JoinDenomReward(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	participant, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", joiner.String())
	suite.Require().True(found)
	suite.Require().Equal("500", participant.Amount.String())
	suite.Require().True(hasDrParticipantMarker(*suite.k, suite.ctx, joiner.String(), "ubze"))

	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("500", dr.StakedAmount.String())

	e, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardJoinEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "denom", "ubze")
	suite.requireEventAttr(e, "address", joiner.String())
	suite.requireEventAttr(e, "amount", "500")
}

// A fresh joiner is stamped an index equal to S for every existing accumulator — including one whose
// S is already > 0 — so nothing distributed before they joined accrues to them (rule 10 / I3). No
// payout is made on join, and a later settle at the unchanged S would pay zero.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_FreshJoinStampsEveryAccumulator() {
	joiner := sdk.AccAddress("dr-joiner-02")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(1000)})
	// two accumulators, one already advanced past zero
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uatom", DistributedStake: math.LegacyMustNewDecFromStr("3")})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ubtc", DistributedStake: math.LegacyMustNewDecFromStr("0")})

	suite.bank.EXPECT().SpendableCoins(suite.ctx, joiner).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10_000)))).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, joiner, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10)))).
		Return(nil).Times(1)
	// NB: no SendCoinsFromModuleToAccount is expected — a fresh joiner is never paid on join

	msg := types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(10))
	_, err := suite.msgServer.JoinDenomReward(suite.ctx, msg)
	suite.Require().NoError(err)

	// every accumulator was stamped to its own S
	atomIdx, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, joiner.String(), "ubze", "uatom")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("3").Equal(atomIdx.Index))
	btcIdx, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, joiner.String(), "ubze", "ubtc")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("0").Equal(btcIdx.Index))

	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("1010", dr.StakedAmount.String())
}

// A top-up settles every prize at the CURRENT amount before the amount grows (rule 6 / I1). The payout
// equals old_amount × (S − index); if the code settled after the increase it would pay
// new_amount × (S − index), so pinning the exact payout proves the ordering.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_TopUpSettlesBeforeAmountChange() {
	joiner := sdk.AccAddress("dr-joiner-03")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(1000)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("1")})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: joiner.String(), StakingDenom: "ubze", Amount: math.NewInt(100)})
	// index missing -> lazy zero -> pending = 100 * (1 - 0) = 100 at the OLD amount

	suite.bank.EXPECT().SpendableCoins(suite.ctx, joiner).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10_000)))).Times(1)
	// settle pays the pre-top-up amount's reward (100), not 150
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, joiner, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(100)))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, joiner, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(50)))).
		Return(nil).Times(1)

	msg := types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(50))
	_, err := suite.msgServer.JoinDenomReward(suite.ctx, msg)
	suite.Require().NoError(err)

	participant, _ := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", joiner.String())
	suite.Require().Equal("150", participant.Amount.String())
	idx, _ := suite.k.GetDenomRewardParticipantIndex(suite.ctx, joiner.String(), "ubze", "uprize")
	suite.Require().True(math.LegacyMustNewDecFromStr("1").Equal(idx.Index))
	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("1050", dr.StakedAmount.String())
}

// The A2-analog: a top-up must not let pre-top-up accrual be captured at the post-top-up amount.
// A dust pending (positive but truncating to zero) is NOT paid on the top-up, and its index is closed
// to S so the fraction is forfeited — never recaptured at the larger amount. A second join with no new
// distribution proves it: it pays nothing (a payout here would be an unexpected mock call and panic).
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_TopUpDustForfeited_NoRecapture() {
	joiner := sdk.AccAddress("dr-joiner-04")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(1)})
	// S = 0.5, amount 1 -> pending 0.5 -> dust
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("0.5")})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: joiner.String(), StakingDenom: "ubze", Amount: math.NewInt(1)})

	suite.bank.EXPECT().SpendableCoins(suite.ctx, joiner).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100_000_000)))).AnyTimes()
	// only escrow sends are allowed; NO SendCoinsFromModuleToAccount is set up on purpose
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, joiner, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1_000_000)))).
		Return(nil).Times(1)

	_, err := suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(1000000)))
	suite.Require().NoError(err)

	// dust forfeited: the index is closed to S, and the amount grew
	participant, _ := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", joiner.String())
	suite.Require().Equal("1000001", participant.Amount.String())
	idx, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, joiner.String(), "ubze", "uprize")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("0.5").Equal(idx.Index))
	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("1000001", dr.StakedAmount.String())

	// second top-up with no new distribution: pending = 1000001 * (0.5 - 0.5) = 0 -> nothing paid.
	// If the pre-top-up 0.5 had been recapturable it would now be 1000001*0.5 = 500000 -> a payout
	// mock call would fire and panic. Its absence is the assertion.
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, joiner, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1)))).
		Return(nil).Times(1)
	_, err = suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(1)))
	suite.Require().NoError(err)

	participant, _ = suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", joiner.String())
	suite.Require().Equal("1000002", participant.Amount.String())
}

// Min stake is enforced on the resulting amount: a first join below it is rejected and no state moves.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_MinStakeFirstJoinRejected() {
	joiner := sdk.AccAddress("dr-joiner-05")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 1000, StakedAmount: math.NewInt(0)})

	suite.bank.EXPECT().SpendableCoins(suite.ctx, joiner).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10_000)))).Times(1)

	msg := types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(500))
	res, err := suite.msgServer.JoinDenomReward(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "min stake")
	suite.Require().Nil(res)

	_, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", joiner.String())
	suite.Require().False(found)
}

// Once the first join meets min stake, a top-up of any positive amount is accepted — the resulting
// (already-compliant) amount only grows, so min stake never blocks a top-up.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_MinStakeNotEnforcedOnTopUp() {
	joiner := sdk.AccAddress("dr-joiner-06")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 1000, StakedAmount: math.NewInt(1000)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: joiner.String(), StakingDenom: "ubze", Amount: math.NewInt(1000)})

	suite.bank.EXPECT().SpendableCoins(suite.ctx, joiner).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10_000)))).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, joiner, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(1)))).
		Return(nil).Times(1)

	// top-up of 1, far below the 1000 min stake, is accepted
	msg := types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(1))
	_, err := suite.msgServer.JoinDenomReward(suite.ctx, msg)
	suite.Require().NoError(err)

	participant, _ := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", joiner.String())
	suite.Require().Equal("1001", participant.Amount.String())
}

// A joiner who cannot cover the stake is rejected before any state is written.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_BalanceTooLow() {
	joiner := sdk.AccAddress("dr-joiner-07")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})

	suite.bank.EXPECT().SpendableCoins(suite.ctx, joiner).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10)))).Times(1)

	msg := types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(500))
	res, err := suite.msgServer.JoinDenomReward(suite.ctx, msg)
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "user balance is too low")
	suite.Require().Nil(res)

	_, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", joiner.String())
	suite.Require().False(found)
}

// Joining a denom with no DR is rejected.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_NotFound() {
	joiner := sdk.AccAddress("dr-joiner-08")
	msg := types.NewMsgJoinDenomReward(joiner.String(), "ubze", math.NewInt(500))
	res, err := suite.msgServer.JoinDenomReward(suite.ctx, msg)
	suite.Require().ErrorIs(err, types.ErrDenomRewardNotFound)
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_NilRequest() {
	res, err := suite.msgServer.JoinDenomReward(suite.ctx, nil)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
	suite.Require().Nil(res)
}

// staked_amount is the exact running sum across joiners and top-ups.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_JoinDenomReward_StakedAmountExactAcrossJoiners() {
	a := sdk.AccAddress("dr-joiner-09a")
	b := sdk.AccAddress("dr-joiner-09b")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})

	suite.bank.EXPECT().SpendableCoins(suite.ctx, a).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10_000)))).AnyTimes()
	suite.bank.EXPECT().SpendableCoins(suite.ctx, b).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10_000)))).AnyTimes()
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, a, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100)))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, b, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(250)))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, a, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(50)))).
		Return(nil).Times(1)

	_, err := suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(a.String(), "ubze", math.NewInt(100)))
	suite.Require().NoError(err)
	_, err = suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(b.String(), "ubze", math.NewInt(250)))
	suite.Require().NoError(err)
	_, err = suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(a.String(), "ubze", math.NewInt(50)))
	suite.Require().NoError(err)

	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("400", dr.StakedAmount.String()) // 100 + 250 + 50
	pa, _ := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", a.String())
	suite.Require().Equal("150", pa.Amount.String())
	pb, _ := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", b.String())
	suite.Require().Equal("250", pb.Amount.String())
}

// --- ClaimDenomRewards ---

// A single claim pays EVERY prize denom of the DR in one tx, each with the exact amount
// amount × (S − index), advances each paid index to its accumulator's S, and leaves the position's
// amount and the DR's staked total untouched (claiming never unstakes — Business Logic rule 11).
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ClaimDenomRewards_MultiPrizeExactAmounts() {
	claimer := sdk.AccAddress("dr-claimer-01")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(1000)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: claimer.String(), StakingDenom: "ubze", Amount: math.NewInt(100)})
	// uatom: index missing -> lazy zero -> pending = 100 × 2 = 200
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uatom", DistributedStake: math.LegacyMustNewDecFromStr("2")})
	// ubtc: index 0.1 -> pending = 100 × (0.5 − 0.1) = 40
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "ubtc", DistributedStake: math.LegacyMustNewDecFromStr("0.5")})
	suite.k.SetDenomRewardParticipantIndex(suite.ctx, types.DenomRewardParticipantIndex{
		Address: claimer.String(), StakingDenom: "ubze", PrizeDenom: "ubtc", Index: math.LegacyMustNewDecFromStr("0.1"),
	})

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, claimer, sdk.NewCoins(sdk.NewCoin("uatom", math.NewInt(200)))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, claimer, sdk.NewCoins(sdk.NewCoin("ubtc", math.NewInt(40)))).
		Return(nil).Times(1)

	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: claimer.String(), Denom: "ubze"})
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(
		sdk.NewCoins(sdk.NewCoin("uatom", math.NewInt(200)), sdk.NewCoin("ubtc", math.NewInt(40))),
		sdk.Coins(res.Amounts),
	)

	// both indexes advanced to their accumulator's S
	atomIdx, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, claimer.String(), "ubze", "uatom")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("2").Equal(atomIdx.Index))
	btcIdx, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, claimer.String(), "ubze", "ubtc")
	suite.Require().True(found)
	suite.Require().True(math.LegacyMustNewDecFromStr("0.5").Equal(btcIdx.Index))

	// position and pool untouched
	participant, _ := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", claimer.String())
	suite.Require().Equal("100", participant.Amount.String())
	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("1000", dr.StakedAmount.String())

	e, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardClaimEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "denom", "ubze")
	suite.requireEventAttr(e, "address", claimer.String())
	suite.requireEventAttr(e, "amounts", "200uatom,40ubtc")
}

// A pure-dust claim (every pending fraction truncates to zero whole units) is rejected with
// ErrNoRewardsToClaim and advances NO index, so the fractions keep accruing (rule 12). A payout or a
// stamped index here would silently forfeit the dust.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ClaimDenomRewards_PureDustErrors_NoIndexAdvance() {
	claimer := sdk.AccAddress("dr-claimer-02")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(1)})
	// amount 1 × S 0.5 -> pending 0.5 -> dust. no bank send is mocked: one would panic.
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("0.5")})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: claimer.String(), StakingDenom: "ubze", Amount: math.NewInt(1)})

	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: claimer.String(), Denom: "ubze"})
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)
	suite.Require().Nil(res)

	// the index was NOT stamped — the dust interval stays open
	_, found := suite.k.GetDenomRewardParticipantIndex(suite.ctx, claimer.String(), "ubze", "uprize")
	suite.Require().False(found)
}

// Claiming twice without a new distribution pays only once: the second claim finds every interval
// closed and is rejected with ErrNoRewardsToClaim.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ClaimDenomRewards_SecondClaimPaysNothing() {
	claimer := sdk.AccAddress("dr-claimer-03")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(100)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("3")})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: claimer.String(), StakingDenom: "ubze", Amount: math.NewInt(100)})

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, claimer, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(300)))).
		Return(nil).Times(1)

	_, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: claimer.String(), Denom: "ubze"})
	suite.Require().NoError(err)

	// second claim: every index == S, nothing pending. a bank send here would be unexpected and panic.
	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: claimer.String(), Denom: "ubze"})
	suite.Require().ErrorIs(err, types.ErrNoRewardsToClaim)
	suite.Require().Nil(res)
}

// Claiming a denom with no DR is rejected.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ClaimDenomRewards_NotFound() {
	claimer := sdk.AccAddress("dr-claimer-04")
	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: claimer.String(), Denom: "ubze"})
	suite.Require().ErrorIs(err, types.ErrDenomRewardNotFound)
	suite.Require().Nil(res)
}

// Claiming without a position is rejected.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ClaimDenomRewards_NotParticipant() {
	claimer := sdk.AccAddress("dr-claimer-05")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(0)})

	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: claimer.String(), Denom: "ubze"})
	suite.Require().ErrorIs(err, types.ErrDenomRewardNotFound)
	suite.Require().Contains(err.Error(), "not a participant")
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ClaimDenomRewards_NilRequest() {
	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, nil)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
	suite.Require().Nil(res)
}

// --- ExitDenomReward ---

// An exit settles the pending prizes BEFORE removing anything (rule 7) — the pending payout lands in
// the exit tx itself — and with a snapshotted lock of 0 the stake is returned immediately. Afterwards
// the position is fully erased: indexes, participant and its address-first marker; staked_amount drops
// by exactly the exited amount.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_LockZero_SettlesAndReturnsImmediately() {
	exiter := sdk.AccAddress("dr-exiter-01")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 0, MinStake: 0, StakedAmount: math.NewInt(800)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("1")})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: exiter.String(), StakingDenom: "ubze", Amount: math.NewInt(500)})
	// index missing -> lazy zero -> pending = 500 × 1 = 500 uprize paid during the exit

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, exiter, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(500)))).
		Return(nil).Times(1)
	// lock 0: the stake comes back in the same tx
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, exiter, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).Times(1)

	res, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// position fully erased
	_, found := suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", exiter.String())
	suite.Require().False(found)
	suite.Require().False(hasDrParticipantMarker(*suite.k, suite.ctx, exiter.String(), "ubze"))
	_, found = suite.k.GetDenomRewardParticipantIndex(suite.ctx, exiter.String(), "ubze", "uprize")
	suite.Require().False(found)

	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("300", dr.StakedAmount.String())

	e, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardExitEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "denom", "ubze")
	suite.requireEventAttr(e, "address", exiter.String())
	suite.requireEventAttr(e, "amount", "500")

	// no pending-unlock entry was left behind
	suite.Require().Empty(suite.k.GetAllPendingUnlockParticipant(suite.ctx))
}

// The exit consumes the extra gas configured in params (mirrors ExitStaking): the exit schedules
// future unlock work the tx's own gas would not otherwise account for.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_ConsumesExtraGas() {
	exiter := sdk.AccAddress("dr-exiter-02")
	suite.Require().NoError(suite.k.SetParams(suite.ctx, types.Params{ExtraGasForDenomExit: 12345}))
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 0, MinStake: 0, StakedAmount: math.NewInt(100)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: exiter.String(), StakingDenom: "ubze", Amount: math.NewInt(100)})

	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, exiter, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100)))).
		Return(nil).Times(1)

	gasBefore := suite.ctx.GasMeter().GasConsumed()
	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().NoError(err)
	suite.Require().GreaterOrEqual(suite.ctx.GasMeter().GasConsumed()-gasBefore, uint64(12345))
}

// With a positive snapshotted lock the stake is NOT returned: a pending-unlock entry is written at
// hour-epoch now + lock×24 under the DR key part "dr/{denom}/{address}", and the position is erased.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_LockPositive_PendingUnlockAtRightEpoch() {
	exiter := sdk.AccAddress("dr-exiter-03")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(500)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: exiter.String(), StakingDenom: "ubze", Amount: math.NewInt(500)})

	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100), nil).Times(1)
	// no bank send is mocked: an immediate return of the stake would panic

	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().NoError(err)

	expectedKey := fmt.Sprintf("%d/dr/%s/%s", 100+7*24, "ubze", exiter.String())
	pending, found := suite.k.GetPendingUnlockParticipant(suite.ctx, expectedKey)
	suite.Require().True(found)
	suite.Require().Equal(exiter.String(), pending.Address)
	suite.Require().Equal("500", pending.Amount)
	suite.Require().Equal("ubze", pending.Denom)

	_, found = suite.k.GetDenomRewardParticipant(suite.ctx, "ubze", exiter.String())
	suite.Require().False(found)
	suite.Require().False(hasDrParticipantMarker(*suite.k, suite.ctx, exiter.String(), "ubze"))
	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("0", dr.StakedAmount.String())

	// the exit event carries the exited amount on the locked path too
	e, ok := suite.findTypedEvent(proto.MessageName(&types.DenomRewardExitEvent{}))
	suite.Require().True(ok)
	suite.requireEventAttr(e, "denom", "ubze")
	suite.requireEventAttr(e, "address", exiter.String())
	suite.requireEventAttr(e, "amount", "500")
}

// Exit → re-join → exit within the same hour-epoch: the second exit lands on the SAME pending-unlock
// key and the amounts are merged, so everything unlocks at once (mirrors beginUnlock).
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_SecondExitSameEpochMerges() {
	exiter := sdk.AccAddress("dr-exiter-04")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(500)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: exiter.String(), StakingDenom: "ubze", Amount: math.NewInt(500)})

	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100), nil).Times(2)

	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().NoError(err)

	// re-join with a fresh position...
	suite.bank.EXPECT().SpendableCoins(suite.ctx, exiter).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10_000)))).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, exiter, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(300)))).
		Return(nil).Times(1)
	_, err = suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(exiter.String(), "ubze", math.NewInt(300)))
	suite.Require().NoError(err)

	// ...and exit again in the same hour-epoch: the entry is merged, not overwritten
	_, err = suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().NoError(err)

	expectedKey := fmt.Sprintf("%d/dr/%s/%s", 100+7*24, "ubze", exiter.String())
	pending, found := suite.k.GetPendingUnlockParticipant(suite.ctx, expectedKey)
	suite.Require().True(found)
	suite.Require().Equal("800", pending.Amount) // 500 + 300

	dr, _ := suite.k.GetDenomReward(suite.ctx, "ubze")
	suite.Require().Equal("0", dr.StakedAmount.String())
}

// DR and SR pending-unlock entries share the same store and epoch prefix without colliding: SR key
// parts start with a digits-only reward id, DR key parts with the literal "dr/". Both survive side by
// side at the same epoch.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_CoexistsWithSrPendingUnlock() {
	exiter := sdk.AccAddress("dr-exiter-05")
	// a pre-existing SR pending unlock at the same target epoch
	srKey := types.CreatePendingUnlockParticipantKey(268, fmt.Sprintf("%s/%s", "00123", exiter.String()))
	suite.k.SetPendingUnlockParticipant(suite.ctx, types.PendingUnlockParticipant{
		Index: srKey, Address: exiter.String(), Amount: "77", Denom: "ubze",
	})

	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(500)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: exiter.String(), StakingDenom: "ubze", Amount: math.NewInt(500)})
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100), nil).Times(1)

	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().NoError(err)

	// both entries exist at epoch 268, untouched by each other
	entries := suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, 268)
	suite.Require().Len(entries, 2)
	srEntry, found := suite.k.GetPendingUnlockParticipant(suite.ctx, srKey)
	suite.Require().True(found)
	suite.Require().Equal("77", srEntry.Amount)
	drEntry, found := suite.k.GetPendingUnlockParticipant(suite.ctx, fmt.Sprintf("268/dr/ubze/%s", exiter.String()))
	suite.Require().True(found)
	suite.Require().Equal("500", drEntry.Amount)
}

// The existing EndBlock unlock queue pays a DR pending-unlock entry out with no DR-specific code:
// enqueue the epoch, process the queue, and the stake arrives while the entry is removed.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_UnlockQueuePaysOut() {
	exiter := sdk.AccAddress("dr-exiter-06")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(500)})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: exiter.String(), StakingDenom: "ubze", Amount: math.NewInt(500)})
	suite.epoch.EXPECT().
		SafeGetEpochCountByIdentifier(suite.ctx, "hour").
		Return(int64(100), nil).Times(1)

	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().NoError(err)

	// the unlock epoch arrives: the generic queue pays the entry out. RemovePendingUnlockParticipant
	// runs only after a successful performUnlock, so the entry's disappearance proves the payout.
	// the queue processes inside a cached context (ApplyFuncIfNoError), hence gomock.Any() for ctx.
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, exiter, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(500)))).
		Return(nil).Times(1)
	suite.k.EnqueueUnlockParticipants(suite.ctx, 268)
	suite.k.ProcessUnlockParticipantsQueue(suite.ctx)

	suite.Require().Empty(suite.k.GetAllEpochPendingUnlockParticipant(suite.ctx, 268))
}

// The A13 analog: exit → re-join → claim pays only what was distributed AFTER the re-join. The old
// position's history is gone (indexes removed at exit, fresh stamps at re-join), so the claim spans
// exactly the post-re-join accrual.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_RejoinStartsFresh() {
	user := sdk.AccAddress("dr-exiter-07")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 0, MinStake: 0, StakedAmount: math.NewInt(100)})
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("2")})
	suite.k.SetDenomRewardParticipant(suite.ctx, types.DenomRewardParticipant{Address: user.String(), StakingDenom: "ubze", Amount: math.NewInt(100)})

	// exit: settles 100 × (2 − 0) = 200 uprize and returns the 100 ubze stake (lock 0)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, user, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(200)))).
		Return(nil).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, user, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100)))).
		Return(nil).Times(1)
	_, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: user.String(), Denom: "ubze"})
	suite.Require().NoError(err)

	// re-join: fresh stamps at the current S = 2
	suite.bank.EXPECT().SpendableCoins(suite.ctx, user).
		Return(sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(10_000)))).Times(1)
	suite.bank.EXPECT().
		SendCoinsFromAccountToModule(suite.ctx, user, types.ModuleName, sdk.NewCoins(sdk.NewCoin("ubze", math.NewInt(100)))).
		Return(nil).Times(1)
	_, err = suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(user.String(), "ubze", math.NewInt(100)))
	suite.Require().NoError(err)

	// the accumulator advances after the re-join: S 2 -> 3
	suite.k.SetDenomRewardPrize(suite.ctx, types.DenomRewardPrize{StakingDenom: "ubze", PrizeDenom: "uprize", DistributedStake: math.LegacyMustNewDecFromStr("3")})

	// claim pays exactly 100 × (3 − 2) = 100, not a unit of the pre-exit history
	suite.bank.EXPECT().
		SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, user, sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(100)))).
		Return(nil).Times(1)
	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: user.String(), Denom: "ubze"})
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewCoins(sdk.NewCoin("uprize", math.NewInt(100))), sdk.Coins(res.Amounts))
}

// Exiting a denom with no DR is rejected.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_NotFound() {
	exiter := sdk.AccAddress("dr-exiter-08")
	res, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().ErrorIs(err, types.ErrDenomRewardNotFound)
	suite.Require().Nil(res)
}

// Exiting without a position is rejected.
func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_NotParticipant() {
	exiter := sdk.AccAddress("dr-exiter-09")
	suite.k.SetDenomReward(suite.ctx, types.DenomReward{StakingDenom: "ubze", Lock: 7, MinStake: 0, StakedAmount: math.NewInt(100)})

	res, err := suite.msgServer.ExitDenomReward(suite.ctx, &types.MsgExitDenomReward{Creator: exiter.String(), Denom: "ubze"})
	suite.Require().ErrorIs(err, types.ErrDenomRewardNotFound)
	suite.Require().Contains(err.Error(), "not a participant")
	suite.Require().Nil(res)
}

func (suite *IntegrationTestSuite) TestMsgServerDenomReward_ExitDenomReward_NilRequest() {
	res, err := suite.msgServer.ExitDenomReward(suite.ctx, nil)
	suite.Require().ErrorIs(err, sdkerrors.ErrInvalidRequest)
	suite.Require().Nil(res)
}
