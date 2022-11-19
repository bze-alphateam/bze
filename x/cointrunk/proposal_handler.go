package cointrunk

import (
	"github.com/bze-alphateam/bze/x/cointrunk/keeper"
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func NewCointrunkProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.PublisherProposal:
			return handlePublisherProposal(ctx, k, c)
		case *types.AcceptedDomainProposal:
			return handleAcceptedDomainProposal(ctx, k, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized cointrunk proposal content type: %T", c)
		}
	}
}

func handlePublisherProposal(ctx sdk.Context, k keeper.Keeper, proposal *types.PublisherProposal) error {
	_ = sdk.MustAccAddressFromBech32(proposal.Address)
	publisher, _ := k.GetPublisher(ctx, proposal.Address)
	publisher.Name = proposal.Name
	publisher.Active = proposal.Active
	publisher.Address = proposal.Address
	k.SetPublisher(ctx, publisher)
	return nil
}

func handleAcceptedDomainProposal(ctx sdk.Context, k keeper.Keeper, proposal *types.AcceptedDomainProposal) error {
	acceptedDomain, _ := k.GetAcceptedDomain(ctx, proposal.Domain)
	acceptedDomain.Domain = proposal.Domain
	acceptedDomain.Active = proposal.Active
	k.SetAcceptedDomain(ctx, acceptedDomain)
	return nil
}
