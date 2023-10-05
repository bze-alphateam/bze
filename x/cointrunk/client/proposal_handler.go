package client

import (
	"github.com/bze-alphateam/bze/x/cointrunk/client/cli"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var AcceptedDomainProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitAcceptedDomainProposal)
var PublisherProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitPublisherProposal)
