package simulation

import (
	"math/rand"

	"github.com/bze-alphateam/bze/x/cointrunk/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgSavePublisher(
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgSavePublisher{
			Authority: simAccount.Address.String(),
		}

		// TODO: Handling the SavePublisher simulation

		return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "SavePublisher simulation not implemented"), nil, nil
	}
}
