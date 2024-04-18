package simulation

import (
	"math/rand"

	"github.com/bze-alphateam/bze/x/rewards/keeper"
	"github.com/bze-alphateam/bze/x/rewards/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgDistributeStakingRewards(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgDistributeStakingRewards{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the DistributeStakingRewards simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "DistributeStakingRewards simulation not implemented"), nil, nil
	}
}
