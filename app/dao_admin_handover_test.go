package app_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	daotypes "github.com/bze-alphateam/bze/x/dao/types"
)

// The two-step handover through the app's router: the admin nominates, the
// nominee accepts and then spends the treasury, the former admin no longer can.
func TestDaoAdminHandover_NominateThenAccept(t *testing.T) {
	bApp, ctx, admin, treasury := newAdminDaoApp(t)
	successor := sdk.AccAddress("dao_successor_______")

	_, err := runAsTx(bApp, ctx, daotypes.NewMsgNominateAdmin(admin.String(), 1, successor.String()))
	require.NoError(t, err)
	_, err = runAsTx(bApp, ctx, daotypes.NewMsgAcceptAdminNomination(successor.String(), 1))
	require.NoError(t, err)

	dao, found := bApp.DaoKeeper.GetDao(ctx, 1)
	require.True(t, found)
	require.Equal(t, successor.String(), dao.Admin)
	require.Empty(t, dao.PendingAdmin)

	spend := func(signer sdk.AccAddress) error {
		msg, err := daotypes.NewMsgAdminExecute(signer.String(), 1, []sdk.Msg{
			banktypes.NewMsgSend(treasury, signer, sdk.NewCoins(sdk.NewInt64Coin("ubze", 10))),
		})
		require.NoError(t, err)
		_, err = runAsTx(bApp, ctx, msg)

		return err
	}
	require.ErrorIs(t, spend(admin), daotypes.ErrNotDaoAdmin)
	require.NoError(t, spend(successor))
	require.Equal(t, int64(90), bApp.BankKeeper.GetBalance(ctx, treasury, "ubze").Amount.Int64())
}
