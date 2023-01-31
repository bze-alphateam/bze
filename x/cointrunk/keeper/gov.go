package keeper

import (
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) HandlePublisherProposal(ctx sdk.Context, proposal *types.PublisherProposal) error {
	_ = sdk.MustAccAddressFromBech32(proposal.Address)
	publisher, found := k.GetPublisher(ctx, proposal.Address)
	publisher.Name = proposal.Name
	publisher.Active = proposal.Active
	if !found {
		publisher.Address = proposal.Address
		publisher.CreatedAt = ctx.BlockHeader().Time.Unix()
		publisher.ArticlesCount = 0
		publisher.Respect = 0
	}

	k.SetPublisher(ctx, publisher)

	if found {
		event := types.PublisherUpdatedEvent{Publisher: &publisher}
		if err := ctx.EventManager().EmitTypedEvent(&event); err != nil {
			return err
		}
	} else {
		event := types.PublisherAddedEvent{Publisher: &publisher}
		if err := ctx.EventManager().EmitTypedEvent(&event); err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) HandleAcceptedDomainProposal(ctx sdk.Context, proposal *types.AcceptedDomainProposal) error {
	acceptedDomain, found := k.GetAcceptedDomain(ctx, proposal.Domain)
	acceptedDomain.Domain = proposal.Domain
	acceptedDomain.Active = proposal.Active
	k.SetAcceptedDomain(ctx, acceptedDomain)

	if found {
		event := types.AcceptedDomainUpdatedEvent{AcceptedDomain: &acceptedDomain}
		if err := ctx.EventManager().EmitTypedEvent(&event); err != nil {
			return err
		}
	} else {
		event := types.AcceptedDomainAddedEvent{AcceptedDomain: &acceptedDomain}
		if err := ctx.EventManager().EmitTypedEvent(&event); err != nil {
			return err
		}
	}

	return nil
}
