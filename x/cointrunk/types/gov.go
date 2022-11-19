package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"regexp"
)

const (
	ProposalTypeAcceptedDomain = "AcceptedDomainProposal"
	ProposalTypePublisher      = "PublisherProposal"
)

func init() {
	govtypes.RegisterProposalType(ProposalTypeAcceptedDomain)
	govtypes.RegisterProposalTypeCodec(&AcceptedDomainProposal{}, "cointrunk/AcceptedDomainProposal")
	govtypes.RegisterProposalType(ProposalTypePublisher)
	govtypes.RegisterProposalTypeCodec(&PublisherProposal{}, "cointrunk/PublisherProposal")
}

var (
	_ govtypes.Content = &AcceptedDomainProposal{}
	_ govtypes.Content = &PublisherProposal{}
)

func NewAcceptedDomainProposal(title, description, domain string, active bool) govtypes.Content {
	return &AcceptedDomainProposal{
		Title:       title,
		Description: description,
		Domain:      domain,
		Active:      active,
	}
}

func (m *AcceptedDomainProposal) ProposalRoute() string { return RouterKey }

func (m *AcceptedDomainProposal) ProposalType() string { return ProposalTypeAcceptedDomain }

func (m *AcceptedDomainProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(m)
	if err != nil {
		return err
	}
	RegExp := regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z
 ]{2,3})$`)

	isValidDomain := RegExp.MatchString(m.Domain)
	if !isValidDomain {
		return sdkerrors.Wrapf(ErrInvalidProposalContent, "proposal domain is invalid")
	}

	return nil
}

func NewPublisherProposal(title, description, name, address string, active bool) govtypes.Content {
	return &PublisherProposal{
		Title:       title,
		Description: description,
		Name:        name,
		Address:     address,
		Active:      active,
	}
}

func (m *PublisherProposal) ProposalRoute() string { return RouterKey }

func (m *PublisherProposal) ProposalType() string { return ProposalTypePublisher }

func (m *PublisherProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(m)
	if err != nil {
		return err
	}
	_, err = sdk.AccAddressFromBech32(m.Address)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidProposalContent, "proposal publisher address is invalid")
	}

	return nil
}
