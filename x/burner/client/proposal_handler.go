package client

import (
	"net/http"

	"github.com/bze-alphateam/bze/x/burner/client/cli"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/rest"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
)

var BurnCoinsProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitBurnCoinsProposal, emptyRestHandler)

func emptyRestHandler(client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "unsupported-burner",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			rest.WriteErrorResponse(w, http.StatusBadRequest, "Legacy REST Routes are not supported for Burner proposals")
		},
	}
}
