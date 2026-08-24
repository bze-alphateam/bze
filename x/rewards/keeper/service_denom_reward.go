package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// settleDenomParticipant pays out and settles every pending prize of a participant's position in a
// denom reward. It is the DR analog of the audited single-prize claimPending and the module's
// settle-before-amount-change primitive (security invariant I1): callers MUST run it to completion
// before mutating participant.Amount or dr.StakedAmount, otherwise the unsettled span would be
// multiplied by coins that were not staked during it — an escrow drain (Business Logic rule 6).
//
// Per accumulator (iterated in deterministic lexicographic prize-denom order, bounded by
// max_prize_denoms_per_dr):
//   - the participant's index is read; a missing index reads as "0" — the lazy-zero rule, safe by
//     construction because joining stamps every existing accumulator (Business Logic rule 10 / I3);
//   - pending = amount × (S − index) is computed in LegacyDec (S only ever grows, so the span is
//     never negative);
//   - reward = pending.TruncateInt(); when reward > 0 a single payout is sent and the index is
//     advanced to S. Dust (pending > 0 but truncating to zero whole units) sends nothing and does
//     NOT advance the index — the fraction keeps accruing until a whole unit is payable, exactly
//     like claimPending (Business Logic rule 12). pending == 0 is a no-op and is skipped.
//
// Any bank error aborts the whole tx; because message execution is atomic, no partial payout
// survives. Returns the total paid across all prize denoms.
func (k Keeper) settleDenomParticipant(ctx sdk.Context, dr types.DenomReward, participant types.DenomRewardParticipant) (sdk.Coins, error) {
	deposited := math.LegacyNewDecFromInt(participant.Amount)

	acc, err := sdk.AccAddressFromBech32(participant.Address)
	if err != nil {
		return nil, err
	}

	paid := sdk.NewCoins()
	var iterErr error
	k.IterateDenomRewardPrizes(ctx, dr.StakingDenom, func(ctx sdk.Context, prize types.DenomRewardPrize) (stop bool) {
		s := prize.DistributedStake

		// missing index means zero (lazy-zero rule, Business Logic rule 10)
		index := math.LegacyZeroDec()
		if stored, found := k.GetDenomRewardParticipantIndex(ctx, participant.Address, dr.StakingDenom, prize.PrizeDenom); found {
			index = stored.Index
		}

		// pending = amount × (S − index)
		pending := deposited.Mul(s.Sub(index))
		if !pending.IsPositive() {
			// nothing accrued for this accumulator: stamping would be a no-op. skip.
			return false
		}

		reward := pending.TruncateInt()
		if !reward.IsPositive() {
			// dust: positive pending truncating to zero whole units. no send, NO stamp — the
			// fraction keeps accruing until a whole unit is payable (Business Logic rule 12).
			return false
		}

		toSend := sdk.NewCoin(prize.PrizeDenom, reward)
		if sErr := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, sdk.NewCoins(toSend)); sErr != nil {
			iterErr = sErr
			return true
		}

		// advance the settlement snapshot to S only after a real payout was sent
		k.SetDenomRewardParticipantIndex(ctx, types.DenomRewardParticipantIndex{
			Address:      participant.Address,
			StakingDenom: dr.StakingDenom,
			PrizeDenom:   prize.PrizeDenom,
			Index:        prize.DistributedStake,
		})

		paid = paid.Add(toSend)

		return false
	})

	if iterErr != nil {
		return nil, iterErr
	}

	return paid, nil
}

// stampParticipantIndexes writes index = S for EVERY existing accumulator of a denom reward, giving
// a freshly joined participant a settlement snapshot on each current prize. This is what makes the
// lazy-zero rule sound (Business Logic rule 10 / invariant I3): after this call a missing index can
// only mean the accumulator was born after the participant joined, so reading it as zero credits the
// participant exactly the rewards distributed since they joined — never earlier ones.
//
// It is called immediately before a position's amount changes, in two places: on a fresh join (the
// position is created at amount 0), and on a top-up right AFTER settleDenomParticipant has paid out
// every whole-unit prize. It must never be called on a live position WITHOUT a preceding full settle:
// re-stamping unsettled indexes would silently discard real pending. After a full settle the only
// thing it discards is sub-unit dust, which — like SR's unconditional JoinedAt reset on a top-up — is
// forfeited so that no settlement interval ever spans an amount change (security invariant I1).
func (k Keeper) stampParticipantIndexes(ctx sdk.Context, address, stakingDenom string) {
	k.IterateDenomRewardPrizes(ctx, stakingDenom, func(ctx sdk.Context, prize types.DenomRewardPrize) (stop bool) {
		k.SetDenomRewardParticipantIndex(ctx, types.DenomRewardParticipantIndex{
			Address:      address,
			StakingDenom: stakingDenom,
			PrizeDenom:   prize.PrizeDenom,
			Index:        prize.DistributedStake,
		})

		return false
	})
}

// ensureDenomRewardPrize returns the (staking denom, prize denom) accumulator, creating it when the
// prize denom is new to the denom reward. Creation is the paid, capped event both money-in paths
// share (Business Logic rules 20–22): the cap check uses >= against the LIVE param so a DR left
// over-cap by a later param decrease also refuses new prize denoms, and the creation fee is captured
// from feePayer through the same capture-and-swap flow as the other reward fees. The fee is
// deliberately trigger-agnostic — a schedule and an airdrop introducing the same denom pay
// identically, otherwise the fee could be bypassed via a throwaway schedule. An existing accumulator
// is returned as-is: no fee, no cap check (only NEW prize denoms consume slots).
func (k msgServer) ensureDenomRewardPrize(ctx sdk.Context, stakingDenom, prizeDenom string, feePayer sdk.AccAddress) (types.DenomRewardPrize, error) {
	prize, found := k.GetDenomRewardPrize(ctx, stakingDenom, prizeDenom)
	if found {
		return prize, nil
	}

	params := k.GetParams(ctx)
	if k.CountDenomRewardPrizes(ctx, stakingDenom) >= params.MaxPrizeDenomsPerDr {
		return prize, types.ErrPrizeDenomCapReached
	}

	fee := k.getRewardCreationFee(ctx, params.CreateDenomRewardPrizeFee)
	if fee != nil {
		if err := k.checkUserBalances(ctx, fee, feePayer); err != nil {
			return prize, err
		}

		// the fee can be captured only if the trade keeper is available to swap it
		if k.tradeKeeper == nil {
			return prize, errors.Wrapf(sdkerrors.ErrInvalidRequest, "trade keeper is not available")
		}
		capturedFee, err := k.tradeKeeper.CaptureAndSwapUserFee(ctx, feePayer, fee, types.ModuleName)
		if err != nil {
			return prize, err
		}

		if err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, txfeecollectortypes.CpFeeCollector, capturedFee); err != nil {
			return prize, err
		}
	}

	// the accumulator starts a fresh era at S = 0; existing stakers accrue from this point only
	// (rule 10's lazy-zero), and the first distribution stamps last_distribution_epoch
	prize = types.DenomRewardPrize{
		StakingDenom:     stakingDenom,
		PrizeDenom:       prizeDenom,
		DistributedStake: math.LegacyZeroDec(),
	}
	k.SetDenomRewardPrize(ctx, prize)

	err := ctx.EventManager().EmitTypedEvent(
		&types.DenomRewardPrizeCreateEvent{
			Denom:      stakingDenom,
			PrizeDenom: prizeDenom,
		},
	)
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return prize, nil
}

// distributeToDenomPrize bumps a prize accumulator by amount/T and stamps the distribution epoch,
// then saves the prize. It mirrors the audited distributeStakingRewards: the accumulator only ever
// grows and T is the live staked total at the moment of distribution. Both the daily schedule pass
// and instant airdrops route every payout through this one primitive so the accumulator math lives
// in a single place.
//
// The guards (types.ValidateDenomDistribution) enforce invariant I6 (no distribution may run with
// T = 0 — the payout would credit nobody and strand escrow) and reject a non-positive amount; they
// run BEFORE the epoch is read. Callers escrow `amount` into the module account before calling; a
// returned error leaves the accumulator untouched.
func (k Keeper) distributeToDenomPrize(ctx sdk.Context, prize types.DenomRewardPrize, amount, stakedTotal math.Int) error {
	if err := types.ValidateDenomDistribution(prize.StakingDenom, amount, stakedTotal); err != nil {
		return err
	}

	epoch, err := k.epochKeeper.SafeGetEpochCountByIdentifier(ctx, distributionEpoch)
	if err != nil {
		return err
	}

	// S = S + amount / T
	k.SetDenomRewardPrize(ctx, prize.WithDistribution(amount, stakedTotal, epoch))

	return nil
}

// EnqueueDenomRewardsDistribution checks if any denom reward schedules exist and enqueues
// a distribution request. The module will process the queue in the following blocks.
// Mirrors EnqueueStakingRewardsDistribution: a single first-record probe (never a full scan)
// decides whether there is any work, and an already-pending queue is left untouched so a day
// tick arriving mid-drain cannot reset the cursor.
func (k Keeper) EnqueueDenomRewardsDistribution(ctx sdk.Context) {
	if len(k.GetBatchDenomRewardSchedules(ctx, "", 1)) == 0 {
		return
	}

	queue, found := k.GetDenomRewardsDistributionQueue(ctx)
	if found && queue.Pending {
		// distribution already pending, skip
		return
	}

	queue = types.DenomRewardsDistributionQueue{
		Pending: true,
		Cursor:  "",
	}
	k.SetDenomRewardsDistributionQueue(ctx, queue)
}

// ProcessDenomRewardsDistributionQueue processes denom reward schedule payouts in bounded batches.
// It mirrors ProcessStakingRewardsDistributionQueue: entries are batch-collected (the store
// iterator is closed before any mutation, so the per-schedule pass may delete finished schedules
// safely — the audited SR collect-then-mutate pattern) and the cursor persists progress so more
// than MaxDenomRewardDistributionsPerBlock schedules simply span multiple blocks. Work scales with
// the schedule count only — never with participants (invariant I5).
func (k Keeper) ProcessDenomRewardsDistributionQueue(ctx sdk.Context) {
	queue, found := k.GetDenomRewardsDistributionQueue(ctx)
	if !found || !queue.Pending {
		return
	}

	schedules := k.GetBatchDenomRewardSchedules(ctx, queue.Cursor, types.MaxDenomRewardDistributionsPerBlock)
	if len(schedules) == 0 {
		// no more schedules to process, distribution is complete
		k.RemoveDenomRewardsDistributionQueue(ctx)
		return
	}

	finished := len(schedules) < types.MaxDenomRewardDistributionsPerBlock

	// Process collected entries in a safe context
	lastProcessedKey := queue.Cursor
	for _, schedule := range schedules {
		lastProcessedKey = string(types.DenomRewardScheduleKey(schedule.StakingDenom, schedule.ScheduleId))
		k.distributeDenomRewardSchedule(ctx, schedule)
	}

	if finished {
		k.RemoveDenomRewardsDistributionQueue(ctx)
	} else {
		queue.Cursor = lastProcessedKey
		k.SetDenomRewardsDistributionQueue(ctx, queue)
	}
}

// distributeDenomRewardSchedule pays one day of a schedule into its prize accumulator. A day with
// zero stakers is skipped WITHOUT counting a payout (Business Logic rule 15): the escrowed budget
// is preserved and the schedule stretches until stakers return, so the total distributed over the
// schedule's life is always daily_amount × duration. The finishing payout (payouts reaching
// duration) deletes the schedule and emits the finish event; the accumulator keeps its value
// forever, so already-accrued (and dust) claims are unaffected.
func (k Keeper) distributeDenomRewardSchedule(ctx sdk.Context, schedule types.DenomRewardSchedule) {
	logger := k.Logger().With("denom_reward_schedule", schedule)

	logger.Debug("preparing to distribute denom reward schedule")

	dr, found := k.GetDenomReward(ctx, schedule.StakingDenom)
	if !found {
		logger.Error("denom reward not found for schedule. skipping distribution")
		return
	}

	if !dr.StakedAmount.IsPositive() {
		logger.Debug("denom reward has no staked coins. skipping distribution")
		return
	}

	if schedule.Payouts >= schedule.Duration {
		// defensive: schedules are deleted on their finishing payout, so this should not happen
		logger.Debug("denom reward schedule finished. skipping distribution")
		return
	}

	prize, found := k.GetDenomRewardPrize(ctx, schedule.StakingDenom, schedule.PrizeDenom)
	if !found {
		logger.Error("denom reward prize not found for schedule. skipping distribution")
		return
	}

	if err := k.distributeToDenomPrize(ctx, prize, schedule.DailyAmount, dr.StakedAmount); err != nil {
		logger.Error(err.Error())
		return
	}

	//increment payouts to know when the schedule finished (a.k.a. all payouts calculated)
	schedule.Payouts++
	if schedule.Payouts == schedule.Duration {
		k.RemoveDenomRewardSchedule(ctx, schedule.StakingDenom, schedule.ScheduleId)

		err := ctx.EventManager().EmitTypedEvent(
			&types.DenomRewardScheduleFinishEvent{
				ScheduleId: schedule.ScheduleId,
			},
		)
		if err != nil {
			k.Logger().Error(err.Error())
		}

		return
	}

	k.SetDenomRewardSchedule(ctx, schedule)
}
