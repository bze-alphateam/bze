package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/dao/types"
)

func (k msgServer) UpdateElectorate(goCtx context.Context, msg *types.MsgUpdateElectorate) (*types.MsgUpdateElectorateResponse, error) {
	// the router runs it too; repeated so the server is safe when called directly
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	dao, found := k.GetDao(ctx, msg.DaoId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrDaoNotFound, "dao %d", msg.DaoId)
	}

	// only the DAO account: its voters through a proposal, or its admin
	// through MsgAdminExecute, never the admin signing directly
	authority := sdk.MustAccAddressFromBech32(msg.Authority)
	if authority.String() != dao.Address {
		return nil, errorsmod.Wrapf(types.ErrUnauthorizedSigner, "expected dao %d's address %s, got %s", dao.Id, dao.Address, authority)
	}

	if dao.Type != types.DaoType_DAO_TYPE_STAKE {
		return nil, errorsmod.Wrapf(types.ErrInvalidDaoType, "dao %d is not a stake dao", dao.Id)
	}

	if err := k.validateStakeSources(ctx, msg.Electorate, dao.Electorate, k.GetParams(ctx)); err != nil {
		return nil, err
	}

	previous := dao.Electorate
	dao.Electorate = nil
	if !msg.Electorate.IsEmpty() {
		dao.Electorate = msg.Electorate
	}

	// a self-owned DAO must keep voters able to replace a broken electorate (D11)
	if dao.Admin == dao.Address && !k.isLiveElectorate(ctx, dao) {
		return nil, errorsmod.Wrapf(types.ErrElectorateNotLive, "self-owned dao %d needs a live electorate", dao.Id)
	}

	// open proposals keep their pinned_electorate and their own pins (D25):
	// only the DAO's pin moves, off removed sources and onto added ones
	k.repinSources(ctx, previous, dao.Electorate)
	k.setDao(ctx, dao)

	event := &types.EventDaoElectorateUpdated{Id: dao.Id}
	if dao.Electorate != nil {
		event.SrIds = dao.Electorate.SrIds
		event.DrDenom = dao.Electorate.DrDenom
	}
	if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
		return nil, err
	}

	return &types.MsgUpdateElectorateResponse{}, nil
}

// repinSources drops the pin of each source in previous but not in next and
// adds one to each source in next but not in previous.
func (k Keeper) repinSources(ctx sdk.Context, previous, next *types.StakeElectorate) {
	nextSources := make(map[string]struct{})
	for _, source := range electorateSources(next) {
		nextSources[string(source)] = struct{}{}
	}

	previousSources := make(map[string]struct{})
	for _, source := range electorateSources(previous) {
		previousSources[string(source)] = struct{}{}
		if _, ok := nextSources[string(source)]; !ok {
			k.unpinSource(ctx, source)
		}
	}

	for _, source := range electorateSources(next) {
		if _, ok := previousSources[string(source)]; !ok {
			k.pinSource(ctx, source)
		}
	}
}
