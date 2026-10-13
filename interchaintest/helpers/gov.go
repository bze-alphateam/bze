package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"

	"github.com/bze-alphateam/bze/interchaintest/chain"
)

const proposalTimeout = 2 * time.Minute

// Proposal is the JSON file `bzed tx gov submit-proposal` reads. Messages are
// the proto-JSON messages, each with its "@type" (e.g. "/bze.tradebin.MsgHaltDenoms")
// and the gov authority (GovAuthority) as signer. An empty Deposit means the
// minimum deposit, so the proposal enters voting immediately.
type Proposal struct {
	Messages  []any  `json:"messages"`
	Metadata  string `json:"metadata"`
	Deposit   string `json:"deposit"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Expedited bool   `json:"expedited"`
}

type proposalStatus struct {
	Proposal struct {
		ID           Int64  `json:"id"`
		Status       string `json:"status"`
		FailedReason string `json:"failed_reason"`
	} `json:"proposal"`
}

func withDefaultDeposit(p Proposal) Proposal {
	if p.Deposit == "" {
		p.Deposit = chain.MinDeposit + chain.Denom
		if p.Expedited {
			p.Deposit = chain.ExpeditedMinDeposit + chain.Denom
		}
	}
	return p
}

// SubmitAndPassProposal submits a proposal (a Proposal or *Proposal, or any value that
// marshals to the submit-proposal JSON) from proposer, votes yes with every
// validator and waits until it passed. It fails if the proposal is rejected or
// its messages fail on execution (with the chain's failed_reason).
func SubmitAndPassProposal(ctx context.Context, c *cosmos.CosmosChain, proposer ibc.Wallet, proposal any) (uint64, error) {
	return SubmitAndPassProposalWith(ctx, c, proposer, nil, proposal)
}

// SubmitAndPassProposalWith is SubmitAndPassProposal with options for the
// submit tx (e.g. WithGas for releases where --gas auto does not work).
func SubmitAndPassProposalWith(ctx context.Context, c *cosmos.CosmosChain, proposer ibc.Wallet, opts []TxOption, proposal any) (uint64, error) {
	switch p := proposal.(type) {
	case Proposal:
		proposal = withDefaultDeposit(p)
	case *Proposal:
		proposal = withDefaultDeposit(*p)
	}
	content, err := json.Marshal(proposal)
	if err != nil {
		return 0, fmt.Errorf("marshal proposal: %w", err)
	}

	node := c.GetNode()
	rel := path.Join("proposals", strconv.FormatInt(time.Now().UnixNano(), 10)+".json")
	if err := writeNodeFile(ctx, node, content, rel); err != nil {
		return 0, fmt.Errorf("write proposal file: %w", err)
	}

	res, err := ExecTxWith(ctx, c, proposer, opts, "gov", "submit-proposal", path.Join(node.HomeDir(), rel))
	if err != nil {
		return 0, fmt.Errorf("submit proposal: %w", err)
	}
	if res.Code != 0 {
		return 0, fmt.Errorf("submit proposal: code %d: %s", res.Code, res.RawLog)
	}
	var id uint64
	for _, e := range TxEvents(res, "submit_proposal") {
		if v, ok := e.Attributes["proposal_id"]; ok {
			id, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parse proposal_id %q: %w", v, err)
			}
		}
	}
	if id == 0 {
		return 0, fmt.Errorf("no proposal_id in submit_proposal events of tx %s", res.TxHash)
	}

	if err := voteYesAllValidators(ctx, c, id, opts); err != nil {
		return id, fmt.Errorf("vote on proposal %d: %w", id, err)
	}
	return id, WaitForProposalPassed(ctx, c, id)
}

// validatorKey is the key interchaintest gives each validator in its node's keyring.
const validatorKey = "validator"

// voteYesAllValidators votes yes with every validator, each with its own key
// on its own node, with the submit tx's options (--gas auto by default; a
// fixed gas limit or fee for releases where --gas auto does not work).
func voteYesAllValidators(ctx context.Context, c *cosmos.CosmosChain, id uint64, opts []TxOption) error {
	for _, v := range c.Validators {
		res, err := execTxOn(ctx, c, v, validatorKey, opts, "gov", "vote", strconv.FormatUint(id, 10), "yes")
		if err != nil {
			return fmt.Errorf("validator %s: %w", v.Name(), err)
		}
		if res.Code != 0 {
			return fmt.Errorf("validator %s: vote refused: code %d: %s", v.Name(), res.Code, res.RawLog)
		}
	}
	return nil
}

// WaitForProposalPassed polls the proposal until PROPOSAL_STATUS_PASSED (nil),
// REJECTED or FAILED (error with failed_reason), or two minutes pass.
func WaitForProposalPassed(ctx context.Context, c *cosmos.CosmosChain, id uint64) error {
	ctx, cancel := context.WithTimeout(ctx, proposalTimeout)
	defer cancel()

	idArg := strconv.FormatUint(id, 10)
	for {
		res, err := Query[proposalStatus](ctx, c, "gov", "proposal", idArg)
		if err == nil {
			switch res.Proposal.Status {
			case "PROPOSAL_STATUS_PASSED":
				return nil
			case "PROPOSAL_STATUS_REJECTED", "PROPOSAL_STATUS_FAILED":
				return fmt.Errorf("proposal %d ended %s: %s", id, res.Proposal.Status, res.Proposal.FailedReason)
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("proposal %d did not pass (status %q, last error %v): %w", id, res.Proposal.Status, err, ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}
