package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/daodao/types"
)

// assertAdmin verifies that `signer` is authorized to administer the DAO
// identified by `daoID`. Returns the DAO record on success.
//
// Authorization rule: signer == dao.admin. This is the ONLY authority
// concept (README D13). parent_dao_id confers no authority on its own —
// subDAOs typically set their admin to the parent's account_address at
// creation, after which the standard admin-handoff rules keep the parent in
// place.
//
// signer is taken as the bech32 string from the message's authority field;
// it is parsed here for comparison.
func (k Keeper) assertAdmin(ctx context.Context, daoID uint64, signer string) (types.Dao, error) {
	dao, found := k.GetDao(ctx, daoID)
	if !found {
		return types.Dao{}, errorsmod.Wrapf(types.ErrDaoNotFound, "id=%d", daoID)
	}

	if signer == "" {
		return types.Dao{}, errorsmod.Wrap(types.ErrUnauthorized, "empty signer")
	}

	// Parse both addresses to normalize formatting (no leading whitespace etc.)
	signerAddr, err := sdk.AccAddressFromBech32(signer)
	if err != nil {
		return types.Dao{}, errorsmod.Wrapf(types.ErrInvalidAddress, "signer: %s", err.Error())
	}
	adminAddr, err := sdk.AccAddressFromBech32(dao.Admin)
	if err != nil {
		// Stored admin must always be a valid bech32; if not, state is corrupt.
		return types.Dao{}, errorsmod.Wrapf(types.ErrInvalidAddress, "stored admin: %s", err.Error())
	}

	if !signerAddr.Equals(adminAddr) {
		return types.Dao{}, errorsmod.Wrapf(types.ErrUnauthorized,
			"signer %s is not the DAO admin (%s)", signer, dao.Admin)
	}

	return dao, nil
}
