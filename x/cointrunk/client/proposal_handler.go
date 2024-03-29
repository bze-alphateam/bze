package client

import (
	"net/http"

	"github.com/bze-alphateam/bze/x/cointrunk/client/cli"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/rest"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
)

var AcceptedDomainProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitAcceptedDomainProposal, emptyRestHandler)
var PublisherProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitPublisherProposal, emptyRestHandler)

func emptyRestHandler(client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "unsupported-cointrunk",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			rest.WriteErrorResponse(w, http.StatusBadRequest, "Legacy REST Routes are not supported for Cointrunk proposals")
		},
	}
}
