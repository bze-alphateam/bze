package types

import (
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	ProposalTypeBurnCoins = "BurnCoinsProposal"
)

func init() {
	govtypes.RegisterProposalType(ProposalTypeBurnCoins)
}

var (
	_ govtypes.Content = &BurnCoinsProposal{}
)

func NewBurnCoinsProposal(title, description string) govtypes.Content {
	return &BurnCoinsProposal{
		Title:       title,
		Description: description,
	}
}

func (m *BurnCoinsProposal) ProposalRoute() string { return RouterKey }

func (m *BurnCoinsProposal) ProposalType() string { return ProposalTypeBurnCoins }

func (m *BurnCoinsProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(m)
	if err != nil {
		return err
	}

	return nil
}
