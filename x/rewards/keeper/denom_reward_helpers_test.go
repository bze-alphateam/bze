package keeper_test

import (
	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// hasDrParticipantMarker reports whether (address, staking denom) is reachable through the
// address-first participant marker (drp/a/). The marker is unexported store state; its only public
// observation is IterateUserDenomRewards, which resolves markers back to participant records — so
// a participant is "marked" exactly when it shows up in its owner's position list.
func hasDrParticipantMarker(k keeper.Keeper, ctx sdk.Context, address, stakingDenom string) bool {
	found := false
	k.IterateUserDenomRewards(ctx, address, func(_ sdk.Context, p types.DenomRewardParticipant) bool {
		if p.StakingDenom == stakingDenom {
			found = true
			return true
		}
		return false
	})

	return found
}

// claimDr settles a participant's position through the public ClaimDenomRewards handler and
// returns the coins paid (nil when the handler refused the claim).
func (suite *IntegrationTestSuite) claimDr(addr sdk.AccAddress, stakingDenom string) (sdk.Coins, error) {
	res, err := suite.msgServer.ClaimDenomRewards(suite.ctx, &types.MsgClaimDenomRewards{Creator: addr.String(), Denom: stakingDenom})
	if err != nil {
		return nil, err
	}

	return sdk.NewCoins(res.Amounts...), nil
}

// joinDr stakes `amount` of the staking denom through the public JoinDenomReward handler, wiring
// the balance check and escrow transfer mocks the handler performs.
func (suite *IntegrationTestSuite) joinDr(addr sdk.AccAddress, stakingDenom string, amount int64) error {
	coins := sdk.NewCoins(sdk.NewInt64Coin(stakingDenom, amount))
	suite.bank.EXPECT().SpendableCoins(suite.ctx, addr).Return(coins).Times(1)
	suite.bank.EXPECT().SendCoinsFromAccountToModule(suite.ctx, addr, types.ModuleName, coins).Return(nil).Times(1)

	_, err := suite.msgServer.JoinDenomReward(suite.ctx, types.NewMsgJoinDenomReward(addr.String(), stakingDenom, coins.AmountOf(stakingDenom)))

	return err
}
