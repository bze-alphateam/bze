package types_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/x/rewards/types"
)

// TestRegisterInterfaces_ResolvesEveryMsg guards against a message being accidentally left
// out of RegisterInterfaces. An unregistered sdk.Msg cannot be unpacked from an Any, so it
// would be un-routable in a transaction — and nothing else in the suite would catch the
// omission. This asserts every rewards Msg (including all v8.2.0 Denom Rewards messages and
// the permissionless MsgDeleteStakingReward) resolves by its type URL.
func TestRegisterInterfaces_ResolvesEveryMsg(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	msgs := []sdk.Msg{
		// pre-existing
		&types.MsgCreateStakingReward{},
		&types.MsgUpdateStakingReward{},
		&types.MsgJoinStaking{},
		&types.MsgExitStaking{},
		&types.MsgClaimStakingRewards{},
		&types.MsgDistributeStakingRewards{},
		&types.MsgCreateTradingReward{},
		&types.MsgActivateTradingReward{},
		&types.MsgUpdateParams{},
		// v8.2.0: staking reward cleanup + hooks
		&types.MsgDeleteStakingReward{},
		// v8.2.0: denom rewards
		&types.MsgCreateDenomReward{},
		&types.MsgJoinDenomReward{},
		&types.MsgExitDenomReward{},
		&types.MsgClaimDenomRewards{},
		&types.MsgCreateDenomRewardSchedule{},
		&types.MsgUpdateDenomRewardSchedule{},
		&types.MsgDistributeDenomRewards{},
	}

	for _, msg := range msgs {
		url := sdk.MsgTypeURL(msg)
		resolved, err := registry.Resolve(url)
		require.NoErrorf(t, err, "message %s not registered in the interface registry", url)
		require.NotNil(t, resolved, "resolved nil for %s", url)
	}
}
