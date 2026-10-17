package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	txfeecollectortypes "github.com/bze-alphateam/bze/x/txfeecollector/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) CreateDao(goCtx context.Context, msg *types.MsgCreateDao) (*types.MsgCreateDaoResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	if uint64(len(msg.Members)) > params.MaxStaticMembers {
		return nil, errorsmod.Wrapf(types.ErrInvalidMembers, "a static-list dao can have at most %d members, got %d", params.MaxStaticMembers, len(msg.Members))
	}

	if err := msg.Config.ValidatePeriods(params.MinVotingPeriod, params.MaxVotingPeriod); err != nil {
		return nil, err
	}

	creator := sdk.MustAccAddressFromBech32(msg.Creator)
	daoId := k.GetDaoCounter(ctx) + 1
	daoAddr := types.DaoAddress(daoId)

	// read-only checks first, so nothing is charged or written for a DAO that can't be created
	existing := k.accountKeeper.GetAccount(ctx, daoAddr)
	if err := checkDaoAccount(existing); err != nil {
		return nil, err
	}

	dao := types.Dao{
		Id:          daoId,
		Address:     daoAddr.String(),
		Admin:       daoAddr.String(),
		Type:        msg.Type,
		Name:        msg.Name,
		Description: msg.Description,
		Config:      msg.Config,
		Members:     canonicalAddresses(msg.Members),
		CreatedAt:   ctx.BlockTime().Unix(),
	}
	if msg.Admin != "" {
		dao.Admin = sdk.MustAccAddressFromBech32(msg.Admin).String()
	}

	if !isLiveElectorate(dao) {
		return nil, errorsmod.Wrap(types.ErrInvalidElectorate, "the dao has no live electorate")
	}

	if err := k.chargeCreationFee(ctx, creator, params); err != nil {
		return nil, err
	}

	if existing == nil {
		k.accountKeeper.SetAccount(ctx, k.accountKeeper.NewAccountWithAddress(ctx, daoAddr))
	}

	k.SetDaoCounter(ctx, daoId)
	k.setNewDao(ctx, dao)

	err := ctx.EventManager().EmitTypedEvent(&types.EventDaoCreated{
		Id:      dao.Id,
		Address: dao.Address,
		Admin:   dao.Admin,
		Type:    dao.Type,
	})
	if err != nil {
		k.Logger().Error(err.Error())
	}

	return &types.MsgCreateDaoResponse{Id: dao.Id, Address: dao.Address}, nil
}

// checkDaoAccount accepts no account or a plain BaseAccount without a public key
// (e.g. one created by funds sent to the address ahead of the DAO). An account
// with a public key, or of any other kind, aborts the creation. Derived
// addresses have no private key, so this is a safety net, not an expected path.
func checkDaoAccount(acc sdk.AccountI) error {
	if acc == nil {
		return nil
	}

	if _, ok := acc.(*authtypes.BaseAccount); !ok {
		return errorsmod.Wrapf(types.ErrDaoAccountExists, "unexpected account type %T at %s", acc, acc.GetAddress())
	}

	if acc.GetPubKey() != nil {
		return errorsmod.Wrapf(types.ErrDaoAccountExists, "account %s has a public key", acc.GetAddress())
	}

	return nil
}

// chargeCreationFee sends the dao_creation_fee from the creator to the burner or
// the community pool fee collector. A zero fee is not charged.
func (k Keeper) chargeCreationFee(ctx sdk.Context, creator sdk.AccAddress, params types.Params) error {
	fee := params.DaoCreationFee
	if !fee.IsPositive() {
		return nil
	}

	destModule := txfeecollectortypes.BurnerFeeCollector
	if params.DaoCreationFeeDestination == types.FeeDestinationCommunityPool {
		destModule = txfeecollectortypes.CpFeeCollector
	}

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, creator, destModule, sdk.NewCoins(fee)); err != nil {
		return errorsmod.Wrap(err, "failed to pay the dao creation fee")
	}

	return nil
}

// canonicalAddresses rewrites already validated bech32 addresses in their canonical form.
func canonicalAddresses(addrs []string) []string {
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = sdk.MustAccAddressFromBech32(a).String()
	}

	return out
}
