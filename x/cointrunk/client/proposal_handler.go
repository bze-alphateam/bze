package client

import (
	"github.com/bze-alphateam/bze/x/cointrunk/client/cli"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/rest"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
	"net/http"
)

var AcceptedDomainProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitAcceptedDomainProposal, emptyRestHandler)
var PublisherProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitPublisherProposal, emptyRestHandler)
var BurnCoinsProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitBurnCoinsProposal, emptyRestHandler)

func emptyRestHandler(client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "unsupported-cointrunk",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			rest.WriteErrorResponse(w, http.StatusBadRequest, "Legacy REST Routes are not supported for Cointrunk proposals")
		},
	}
}
