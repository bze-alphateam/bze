package chain

import (
	"context"
	sdkmath "cosmossdk.io/math"
	"encoding/json"
	"fmt"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"io"
	"net/http"
	"time"

	tmabcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/bze-alphateam/bze/tests/e2e/util"
)

// PropTallyResult is the result of a proposal tally.
type PropTallyResult struct {
	Yes        sdkmath.Int
	No         sdkmath.Int
	Abstain    sdkmath.Int
	NoWithVeto sdkmath.Int
}

func (n *NodeConfig) QueryGRPCGateway(path string, parameters ...string) ([]byte, error) {
	if len(parameters)%2 != 0 {
		return nil, fmt.Errorf("invalid number of parameters, must follow the format of key + value")
	}

	// add the URL for the given validator ID, and pre-pend to to path.
	hostPort, err := n.containerManager.GetHostPort(n.Name, "1317/tcp")
	require.NoError(n.t, err)
	endpoint := fmt.Sprintf("http://%s", hostPort)
	fullQueryPath := fmt.Sprintf("%s/%s", endpoint, path)

	var resp *http.Response
	require.Eventually(n.t, func() bool {
		req, err := http.NewRequest("GET", fullQueryPath, nil)
		if err != nil {
			return false
		}

		if len(parameters) > 0 {
			q := req.URL.Query()
			for i := 0; i < len(parameters); i += 2 {
				q.Add(parameters[i], parameters[i+1])
			}
			req.URL.RawQuery = q.Encode()
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			n.t.Logf("error while executing HTTP request: %s", err.Error())
			return false
		}

		return resp.StatusCode != http.StatusServiceUnavailable
	}, time.Minute, 10*time.Millisecond, "failed to execute HTTP request")

	defer resp.Body.Close()

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bz))
	}
	return bz, nil
}

// QueryBalancer returns balances at the address.
func (n *NodeConfig) QueryBalances(address string) (sdk.Coins, error) {
	path := fmt.Sprintf("cosmos/bank/v1beta1/balances/%s", address)
	bz, err := n.QueryGRPCGateway(path)
	require.NoError(n.t, err)

	var balancesResp banktypes.QueryAllBalancesResponse
	if err := util.Cdc.UnmarshalJSON(bz, &balancesResp); err != nil {
		return sdk.Coins{}, err
	}
	return balancesResp.GetBalances(), nil
}

func (n *NodeConfig) QueryBalance(address, denom string) (sdk.Coin, error) {
	path := fmt.Sprintf("cosmos/bank/v1beta1/balances/%s/by_denom?denom=%s", address, denom)
	bz, err := n.QueryGRPCGateway(path)
	require.NoError(n.t, err)

	var balancesResp banktypes.QueryBalanceResponse
	if err := util.Cdc.UnmarshalJSON(bz, &balancesResp); err != nil {
		return sdk.Coin{}, err
	}
	return *balancesResp.GetBalance(), nil
}

func (n *NodeConfig) QuerySupplyOf(denom string) (sdkmath.Int, error) {
	path := fmt.Sprintf("cosmos/bank/v1beta1/supply/%s", denom)
	bz, err := n.QueryGRPCGateway(path)
	require.NoError(n.t, err)

	var supplyResp banktypes.QuerySupplyOfResponse
	if err := util.Cdc.UnmarshalJSON(bz, &supplyResp); err != nil {
		return sdkmath.NewInt(0), err
	}
	return supplyResp.Amount.Amount, nil
}

func (n *NodeConfig) QuerySupply() (sdk.Coins, error) {
	path := "cosmos/bank/v1beta1/supply"
	bz, err := n.QueryGRPCGateway(path)
	require.NoError(n.t, err)

	var supplyResp banktypes.QueryTotalSupplyResponse
	if err := util.Cdc.UnmarshalJSON(bz, &supplyResp); err != nil {
		return nil, err
	}
	return supplyResp.Supply, nil
}

func (n *NodeConfig) QueryPropTally(proposalNumber int) (PropTallyResult, error) {
	path := fmt.Sprintf("cosmos/gov/v1beta1/proposals/%d/tally", proposalNumber)
	bz, err := n.QueryGRPCGateway(path)
	require.NoError(n.t, err)

	var balancesResp govtypesv1.QueryTallyResultResponse
	if err := util.Cdc.UnmarshalJSON(bz, &balancesResp); err != nil {
		return PropTallyResult{
			Yes:        sdkmath.ZeroInt(),
			No:         sdkmath.ZeroInt(),
			Abstain:    sdkmath.ZeroInt(),
			NoWithVeto: sdkmath.ZeroInt(),
		}, err
	}
	noTotal := balancesResp.Tally.No
	yesTotal := balancesResp.Tally.Yes
	noWithVetoTotal := balancesResp.Tally.NoWithVeto
	abstainTotal := balancesResp.Tally.Abstain

	return PropTallyResult{
		Yes:        yesTotal,
		No:         noTotal,
		Abstain:    abstainTotal,
		NoWithVeto: noWithVetoTotal,
	}, nil
}

func (n *NodeConfig) QueryPropStatus(proposalNumber int) (string, error) {
	path := fmt.Sprintf("cosmos/gov/v1beta1/proposals/%d", proposalNumber)
	bz, err := n.QueryGRPCGateway(path)
	require.NoError(n.t, err)

	var propResp govtypesv1.QueryProposalResponse
	if err := util.Cdc.UnmarshalJSON(bz, &propResp); err != nil {
		return "", err
	}
	proposalStatus := propResp.Proposal.Status

	return proposalStatus.String(), nil
}

// QueryHashFromBlock gets block hash at a specific height. Otherwise, error.
func (n *NodeConfig) QueryHashFromBlock(height int64) (string, error) {
	block, err := n.rpcClient.Block(context.Background(), &height)
	if err != nil {
		return "", err
	}
	return block.BlockID.Hash.String(), nil
}

// QueryCurrentHeight returns the current block height of the node or error.
func (n *NodeConfig) QueryCurrentHeight() (int64, error) {
	status, err := n.rpcClient.Status(context.Background())
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

// QueryLatestBlockTime returns the latest block time.
func (n *NodeConfig) QueryLatestBlockTime() time.Time {
	status, err := n.rpcClient.Status(context.Background())
	require.NoError(n.t, err)
	return status.SyncInfo.LatestBlockTime
}

// QueryListSnapshots gets all snapshots currently created for a node.
func (n *NodeConfig) QueryListSnapshots() ([]*tmabcitypes.Snapshot, error) {
	abciResponse, err := n.rpcClient.ABCIQuery(context.Background(), "/app/snapshots", nil)
	if err != nil {
		return nil, err
	}

	var listSnapshots tmabcitypes.ResponseListSnapshots
	if err := json.Unmarshal(abciResponse.Response.Value, &listSnapshots); err != nil {
		return nil, err
	}

	return listSnapshots.Snapshots, nil
}

func (n *NodeConfig) QueryCommunityPoolModuleAccount() string {
	cmd := []string{"bzed", "query", "auth", "module-accounts", "--output=json"}

	out, _, err := n.containerManager.ExecCmd(n.t, n.Name, cmd, "")
	require.NoError(n.t, err)
	var result map[string][]interface{}
	err = json.Unmarshal(out.Bytes(), &result)
	require.NoError(n.t, err)
	for _, acc := range result["accounts"] {
		account, ok := acc.(map[string]interface{})
		require.True(n.t, ok)
		if account["name"] == "distribution" {
			moduleAccount, ok := account["base_account"].(map[string]interface{})["address"].(string)
			require.True(n.t, ok)
			return moduleAccount
		}
	}
	require.True(n.t, false, "distribution module account not found")
	return ""
}
