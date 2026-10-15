package helpers

import (
	"context"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
)

// StoreQuery runs a raw ABCI query against the KV store named store, at height
// (0 = latest), for an arbitrary key. It proves whether the store is mounted:
// a mounted store answers with code 0 (an empty value for a missing key), a
// store that does not exist fails with "no such store: <store>". Transport
// errors are returned as errors; the store's answer is in the response.
func StoreQuery(ctx context.Context, chain *cosmos.CosmosChain, store string, height int64) (abci.ResponseQuery, error) {
	addr := chain.GetHostRPCAddress()
	client, err := rpchttp.New(addr, "/websocket")
	if err != nil {
		return abci.ResponseQuery{}, fmt.Errorf("rpc client %s: %w", addr, err)
	}
	res, err := client.ABCIQueryWithOptions(ctx, "/store/"+store+"/key", []byte{0x00}, rpcclient.ABCIQueryOptions{Height: height})
	if err != nil {
		return abci.ResponseQuery{}, fmt.Errorf("abci query store %s: %w", store, err)
	}
	return res.Response, nil
}
