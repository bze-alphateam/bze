package simulation

import (
	"math/rand"

	"github.com/bze-alphateam/bze/x/tokenfactory/keeper"
	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgBurn(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgBurn{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the Burn simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "Burn simulation not implemented"), nil, nil
	}
}
