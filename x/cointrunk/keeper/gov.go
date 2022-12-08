package keeper

import (
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) HandlePublisherProposal(ctx sdk.Context, proposal *types.PublisherProposal) error {
	_ = sdk.MustAccAddressFromBech32(proposal.Address)
	publisher, _ := k.GetPublisher(ctx, proposal.Address)
	publisher.Name = proposal.Name
	publisher.Active = proposal.Active
	publisher.Address = proposal.Address
	k.SetPublisher(ctx, publisher)
	return nil
}

func (k Keeper) HandleAcceptedDomainProposal(ctx sdk.Context, proposal *types.AcceptedDomainProposal) error {
	acceptedDomain, _ := k.GetAcceptedDomain(ctx, proposal.Domain)
	acceptedDomain.Domain = proposal.Domain
	acceptedDomain.Active = proposal.Active
	k.SetAcceptedDomain(ctx, acceptedDomain)
	return nil
}
