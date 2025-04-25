package keeper

import (
	"context"
	errorsmod "cosmossdk.io/errors"

	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AcceptDomain(goCtx context.Context, msg *types.MsgAcceptDomain) (*types.MsgAcceptDomainResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	acceptedDomain, found := k.GetAcceptedDomain(ctx, msg.Domain)
	acceptedDomain.Domain = msg.Domain
	acceptedDomain.Active = msg.Active
	k.SetAcceptedDomain(ctx, acceptedDomain)

	if found {
		event := types.AcceptedDomainUpdatedEvent{AcceptedDomain: &acceptedDomain}
		if err := ctx.EventManager().EmitTypedEvent(&event); err != nil {
			return nil, err
		}
	} else {
		event := types.AcceptedDomainAddedEvent{AcceptedDomain: &acceptedDomain}
		if err := ctx.EventManager().EmitTypedEvent(&event); err != nil {
			return nil, err
		}
	}

	return &types.MsgAcceptDomainResponse{}, nil
}

func (k msgServer) SavePublisher(goCtx context.Context, msg *types.MsgSavePublisher) (*types.MsgSavePublisherResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	publisher, found := k.GetPublisher(ctx, msg.Address)
	publisher.Name = msg.Name
	publisher.Active = msg.Active
	if !found {
		publisher.Address = msg.Address
		publisher.CreatedAt = ctx.BlockHeader().Time.Unix()
		publisher.ArticlesCount = 0
		publisher.Respect = 0
	}

	k.SetPublisher(ctx, publisher)

	if found {
		event := types.PublisherUpdatedEvent{Publisher: &publisher}
		if err := ctx.EventManager().EmitTypedEvent(&event); err != nil {
			return nil, err
		}
	} else {
		event := types.PublisherAddedEvent{Publisher: &publisher}
		if err := ctx.EventManager().EmitTypedEvent(&event); err != nil {
			return nil, err
		}
	}

	return &types.MsgSavePublisherResponse{}, nil
}
