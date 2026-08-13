package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CreateDenomReward creates the (unique, chain-wide) denom reward pool for a staking denom.
//
// Creation is permissionless and grants no rights: no creator identity is stored (Business Logic
// rule 2). The creation fee is captured through the same capture-and-swap path as CreateStakingReward
// (trade keeper swaps the fee into the module, then it is forwarded to the fee collector). `lock` and
// `min_stake` are a PARAM SNAPSHOT (rule 3): they are copied from the module params at creation and
// frozen on the DR, so later param changes only affect future DRs.
func (k msgServer) CreateDenomReward(goCtx context.Context, msg *types.MsgCreateDenomReward) (*types.MsgCreateDenomRewardResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	// the staking denom must have supply (Business Logic rule 1)
	if !k.bankKeeper.HasSupply(ctx, msg.Denom) {
		return nil, types.ErrInvalidStakingDenom
	}

	// at most one DR per denom, chain-wide (Business Logic rule 1)
	if k.HasDenomReward(ctx, msg.Denom) {
		return nil, types.ErrDenomRewardExists
	}

	params := k.GetParams(ctx)

	// capture the creation fee exactly like CreateStakingReward: trade-keeper capture-and-swap into
	// the module account, then forward to the fee collector. A zero/invalid fee param makes creation
	// free (getRewardCreationFee returns nil).
	fee := k.getRewardCreationFee(ctx, params.CreateDenomRewardFee)
	if fee != nil {
		if err = k.checkUserBalances(ctx, fee, acc); err != nil {
			return nil, err
		}

		// the fee can be captured only if the trade keeper is available to swap it
		if k.tradeKeeper == nil {
			return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "trade keeper is not available")
		}
		capturedFee, err := k.tradeKeeper.CaptureAndSwapUserFee(ctx, acc, fee, types.ModuleName)
		if err != nil {
			return nil, err
		}

		if err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, capturedFee); err != nil {
			return nil, err
		}
	}

	// param snapshot: lock and min_stake are frozen from the current params (Business Logic rule 3)
	dr := types.DenomReward{
		StakingDenom: msg.Denom,
		Lock:         params.DenomRewardLock,
		MinStake:     params.DenomRewardMinStake,
		StakedAmount: "0",
	}
	k.SetDenomReward(ctx, dr)

	err = ctx.EventManager().EmitTypedEvent(
		&types.DenomRewardCreateEvent{
			Denom: dr.StakingDenom,
		},
	)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgCreateDenomRewardResponse{}, nil
}

// JoinDenomReward stakes into a denom reward, handling both a first join and a top-up in one message
// (mirroring JoinStaking). The ordering is a consensus-critical security invariant (I1 / Business
// Logic rule 6): a participant's rewards are settled and every settlement interval is closed BEFORE
// their staked amount changes. Settling after an amount increase would multiply the unsettled span by
// coins that were not staked during it — an escrow drain.
func (k msgServer) JoinDenomReward(goCtx context.Context, msg *types.MsgJoinDenomReward) (*types.MsgJoinDenomRewardResponse, error) {
	if msg == nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	acc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	dr, found := k.GetDenomReward(ctx, msg.Denom)
	if !found {
		return nil, types.ErrDenomRewardNotFound
	}

	stakedAmount := math.ZeroInt()
	if dr.StakedAmount != "" {
		var ok bool
		stakedAmount, ok = math.NewIntFromString(dr.StakedAmount)
		if !ok {
			return nil, fmt.Errorf("could not transform staked amount from storage into int")
		}
	}

	// the coins to escrow: `amount` of the staking denom
	toCapture, err := k.getAmountToCapture(dr.StakingDenom, msg.Amount, int64(1))
	if err != nil {
		return nil, err
	}

	if err = k.checkUserBalances(ctx, toCapture, acc); err != nil {
		return nil, err
	}

	participant, found := k.GetDenomRewardParticipant(ctx, dr.StakingDenom, msg.Creator)
	if found {
		// existing position: settle every whole-unit prize at the CURRENT amount first (rule 6), then
		// close every remaining interval to S. settleDenomParticipant leaves the index of a dust prize
		// (positive pending truncating to zero) untouched so the fraction keeps accruing; but the
		// amount is about to change, and an interval that spans an amount change is unaccountable — so
		// the dust is forfeited here, exactly as SR's unconditional `JoinedAt = S` reset does on a
		// top-up. Without this, pre-top-up accrual could later be captured at the larger post-top-up
		// amount (an escrow drain).
		if _, err = k.settleDenomParticipant(ctx, dr, participant); err != nil {
			return nil, err
		}
		k.stampParticipantIndexes(ctx, msg.Creator, dr.StakingDenom)
	} else {
		// fresh position: create it at amount 0 and stamp an index for every existing accumulator, so
		// the newcomer accrues only from now on and never inherits the pool's history (rule 10 / I3).
		participant = types.DenomRewardParticipant{
			Address:      msg.Creator,
			StakingDenom: dr.StakingDenom,
			Amount:       "0",
		}
		k.stampParticipantIndexes(ctx, msg.Creator, dr.StakingDenom)
	}

	amtInt, ok := math.NewIntFromString(participant.Amount)
	if !ok {
		return nil, fmt.Errorf("could not transform amount from storage into int")
	}
	added := toCapture.AmountOf(dr.StakingDenom)
	amtInt = amtInt.Add(added)

	// the DR's snapshotted min_stake is enforced on the RESULTING amount. A first join must reach it;
	// a top-up only ever increases an already-compliant amount, so it can never be blocked by it.
	if amtInt.LT(math.NewIntFromUint64(dr.MinStake)) {
		return nil, fmt.Errorf("amount is smaller than denom reward min stake")
	}

	participant.Amount = amtInt.String()

	stakedAmount = stakedAmount.Add(added)
	dr.StakedAmount = stakedAmount.String()

	if err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, toCapture); err != nil {
		return nil, err
	}

	k.SetDenomRewardParticipant(ctx, participant)
	k.SetDenomReward(ctx, dr)

	err = ctx.EventManager().EmitTypedEvent(
		&types.DenomRewardJoinEvent{
			Denom:   dr.StakingDenom,
			Address: msg.Creator,
			Amount:  added.String(),
		},
	)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgJoinDenomRewardResponse{}, nil
}
