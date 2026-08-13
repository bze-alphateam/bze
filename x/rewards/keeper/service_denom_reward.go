package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	deposited, err := math.LegacyNewDecFromStr(participant.Amount)
	if err != nil {
		return nil, fmt.Errorf("could not parse participant amount %q: %w", participant.Amount, err)
	}

	acc, err := sdk.AccAddressFromBech32(participant.Address)
	if err != nil {
		return nil, err
	}

	paid := sdk.NewCoins()
	var iterErr error
	k.IterateDenomRewardPrizes(ctx, dr.StakingDenom, func(ctx sdk.Context, prize types.DenomRewardPrize) (stop bool) {
		s, sErr := math.LegacyNewDecFromStr(prize.DistributedStake)
		if sErr != nil {
			iterErr = fmt.Errorf("could not parse accumulator %s/%s: %w", prize.StakingDenom, prize.PrizeDenom, sErr)
			return true
		}

		// missing index means zero (lazy-zero rule, Business Logic rule 10)
		index := math.LegacyZeroDec()
		if stored, found := k.GetDenomRewardParticipantIndex(ctx, participant.Address, dr.StakingDenom, prize.PrizeDenom); found {
			index, sErr = math.LegacyNewDecFromStr(stored.Index)
			if sErr != nil {
				iterErr = fmt.Errorf("could not parse index for %s/%s: %w", dr.StakingDenom, prize.PrizeDenom, sErr)
				return true
			}
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
		if sErr = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, acc, sdk.NewCoins(toSend)); sErr != nil {
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

// distributeToDenomPrize bumps a prize accumulator by amount/T and stamps the distribution epoch,
// then saves the prize. It mirrors the audited distributeStakingRewards: the accumulator only ever
// grows and T is the live staked total at the moment of distribution. Both the daily schedule pass
// and instant airdrops route every payout through this one primitive so the accumulator math lives
// in a single place.
//
// The guards enforce invariant I6 (no distribution may run with T = 0 — the payout would credit
// nobody and strand escrow) and reject a non-positive amount. Callers escrow `amount` into the
// module account before calling; a returned error leaves the accumulator untouched.
func (k Keeper) distributeToDenomPrize(ctx sdk.Context, prize types.DenomRewardPrize, amount, stakedTotal math.Int) error {
	if !stakedTotal.IsPositive() {
		return fmt.Errorf("no stakers found in denom reward %s", prize.StakingDenom)
	}

	if !amount.IsPositive() {
		return fmt.Errorf("distribution amount should be positive")
	}

	s, err := math.LegacyNewDecFromStr(prize.DistributedStake)
	if err != nil {
		return fmt.Errorf("could not parse accumulator %s/%s: %w", prize.StakingDenom, prize.PrizeDenom, err)
	}

	epoch, err := k.epochKeeper.SafeGetEpochCountByIdentifier(ctx, distributionEpoch)
	if err != nil {
		return err
	}

	// S = S + amount / T
	s = s.Add(math.LegacyNewDecFromInt(amount).Quo(math.LegacyNewDecFromInt(stakedTotal)))
	prize.DistributedStake = s.String()
	prize.LastDistributionEpoch = epoch

	k.SetDenomRewardPrize(ctx, prize)

	return nil
}
