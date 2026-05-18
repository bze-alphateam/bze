package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
	rewardstypes "github.com/bze-alphateam/bze/x/rewards/types"
)

// rewardStakedVotingBackend reads voting power live from x/rewards.
//
// Power(addr): the participant's `Amount` field (uint64-parsed).
// TotalPower:  the program's `StakedAmount` field (uint64-parsed).
//
// SnapshotAll is intentionally a stub in Epic 2: REWARD_STAKED DAOs are
// not creatable yet (MsgCreateDao rejects the variant; MsgUpdateVotingBackend
// arrives in Epic 5), so no snapshot path exercises this backend. Epic 3
// will add a per-reward iterator on the rewards keeper and wire it through
// here.
type rewardStakedVotingBackend struct {
	k Keeper
}

func (r rewardStakedVotingBackend) Power(ctx context.Context, dao types.Dao, addr sdk.AccAddress) (uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	participant, found := r.k.rewardsKeeper.GetStakingRewardParticipant(sdkCtx, addr.String(), dao.RewardId)
	if !found {
		return 0, nil
	}
	return parseUint64Amount(participant.Amount)
}

func (r rewardStakedVotingBackend) TotalPower(ctx context.Context, dao types.Dao) (uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	program, found := r.k.rewardsKeeper.GetStakingReward(sdkCtx, dao.RewardId)
	if !found {
		return 0, errorsmod.Wrapf(types.ErrDaoNotFound,
			"DAO %d references missing reward_id %q", dao.Id, dao.RewardId)
	}
	return parseUint64Amount(program.StakedAmount)
}

// SnapshotAll captures every reward-program participant's current stake
// into per-(address, snapshot) rows, plus a SnapshotTotal row equal to the
// program's StakedAmount at snapshot time. Cost: O(N participants per
// reward) plus the linear filter cost of
// IterateStakingRewardParticipantsByReward — see the comment on that
// method for why filtering is done in-process today.
//
// Errors:
//   - if the reward program no longer exists (e.g. closed by its owner
//     between proposal creation and snapshot — which in practice can't
//     happen since CreateProposal calls SnapshotAll synchronously), the
//     snapshot is rejected (a 0-total proposal would be unpassable anyway).
//   - if any participant Amount overflows uint64, surface the error so the
//     surrounding tx reverts.
func (r rewardStakedVotingBackend) SnapshotAll(ctx context.Context, dao types.Dao, snapshotID uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	program, found := r.k.rewardsKeeper.GetStakingReward(sdkCtx, dao.RewardId)
	if !found {
		return errorsmod.Wrapf(types.ErrDaoNotFound,
			"REWARD_STAKED DAO %d references missing reward_id %q", dao.Id, dao.RewardId)
	}

	root := runtime.KVStoreAdapter(r.k.storeService.OpenKVStore(ctx))

	var (
		iterErr   error
		sumPowers uint64 // belt-and-suspenders sanity check vs program.StakedAmount
	)
	r.k.rewardsKeeper.IterateStakingRewardParticipantsByReward(
		sdkCtx,
		dao.RewardId,
		func(p rewardstypes.StakingRewardParticipant) bool {
			addr, err := sdk.AccAddressFromBech32(p.Address)
			if err != nil {
				iterErr = errorsmod.Wrapf(types.ErrInvalidAddress,
					"snapshot iterator: participant %q: %s", p.Address, err.Error())
				return true
			}
			power, err := parseUint64Amount(p.Amount)
			if err != nil {
				iterErr = err
				return true
			}
			// Zero-power participants are dropped: they have no influence
			// and writing a row would only inflate the snapshot size.
			if power == 0 {
				return false
			}
			next, ok := safeAddU64(sumPowers, power)
			if !ok {
				iterErr = errorsmod.Wrap(types.ErrAmountOverflow,
					"snapshot iterator: per-participant sum overflows uint64")
				return true
			}
			sumPowers = next
			root.Set(types.SnapshotPowerKey(dao.Id, snapshotID, addr), sdk.Uint64ToBigEndian(power))
			return false
		},
	)
	if iterErr != nil {
		return iterErr
	}

	// Write the total from the program record (the authoritative number)
	// rather than the iterator sum: the program's StakedAmount is what the
	// rewards keeper maintains as canonical, and lock-step with the
	// per-participant entries is the rewards keeper's invariant — not
	// daodao's to second-guess.
	total, err := parseUint64Amount(program.StakedAmount)
	if err != nil {
		return err
	}
	root.Set(types.SnapshotTotalKey(dao.Id, snapshotID), sdk.Uint64ToBigEndian(total))
	return nil
}

// parseUint64Amount converts an x/rewards amount field (stored as a base-10
// string) into a uint64. Negative values and parse failures error out;
// values that exceed uint64 capacity error with ErrAmountOverflow.
//
// We use uint64 throughout the daodao voting math because realistic
// per-address voting power is well below 2^63 even for the largest
// realistic token supplies. If a chain ever needs larger we'd switch to
// math.Int.
func parseUint64Amount(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// Differentiate "overflow" (number too big) from "syntax" (not a
		// number) for nicer errors, both rolled up under ErrAmountOverflow.
		return 0, errorsmod.Wrapf(types.ErrAmountOverflow, "parse %q: %s", s, err.Error())
	}
	return v, nil
}
