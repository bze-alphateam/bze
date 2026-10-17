package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

// sourceShare is the voting weight one electorate source gives an address.
type sourceShare struct {
	source []byte
	amount math.Int
}

// eligibleWeight sums the address's stake over the electorate's sources and
// returns the total and the non-zero share of every source. A source counts
// only when the address has no join record for it, or joined strictly before
// refHeight (D3, D4): stake that arrived in the reference block or later
// cannot weigh on a proposal submitted at it.
func (k Keeper) eligibleWeight(ctx sdk.Context, electorate *types.StakeElectorate, addr sdk.AccAddress, refHeight int64) (math.Int, []sourceShare) {
	total := math.ZeroInt()
	if electorate.IsEmpty() {
		return total, nil
	}

	var shares []sourceShare
	add := func(source []byte, amount math.Int) {
		if amount.IsNil() || !amount.IsPositive() {
			return
		}

		if record, found := k.GetJoinRecord(ctx, source, addr); found && record.Height >= refHeight {
			return
		}

		total = total.Add(amount)
		shares = append(shares, sourceShare{source: source, amount: amount})
	}

	for _, id := range electorate.SrIds {
		participant, found := k.rewardsKeeper.GetStakingRewardParticipant(ctx, addr.String(), id)
		if !found {
			continue
		}

		// the amount is a plain string in x/rewards; an unparsable one adds nothing
		amount, ok := math.NewIntFromString(participant.Amount)
		if !ok {
			continue
		}

		add(types.StakingRewardSource(id), amount)
	}

	if electorate.DrDenom != "" {
		participant, found := k.rewardsKeeper.GetDenomRewardParticipant(ctx, electorate.DrDenom, addr.String())
		if found {
			add(types.DenomRewardSource(electorate.DrDenom), participant.Amount)
		}
	}

	return total, shares
}
