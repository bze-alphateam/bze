package helpers

import (
	"context"
	"encoding/binary"
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

// StorePrefixKeys counts the keys under prefix in the KV store named store, at
// height (0 = latest), through the store's raw subspace query. It answers
// whether a module still holds any state under a prefix.
func StorePrefixKeys(ctx context.Context, chain *cosmos.CosmosChain, store string, prefix []byte, height int64) (int, error) {
	addr := chain.GetHostRPCAddress()
	client, err := rpchttp.New(addr, "/websocket")
	if err != nil {
		return 0, fmt.Errorf("rpc client %s: %w", addr, err)
	}
	res, err := client.ABCIQueryWithOptions(ctx, "/store/"+store+"/subspace", prefix, rpcclient.ABCIQueryOptions{Height: height})
	if err != nil {
		return 0, fmt.Errorf("abci subspace query store %s: %w", store, err)
	}
	if res.Response.Code != 0 {
		return 0, fmt.Errorf("subspace query store %s: code %d: %s", store, res.Response.Code, res.Response.Log)
	}
	return countKVPairs(res.Response.Value)
}

// countKVPairs counts the entries of a protobuf-encoded store kv.Pairs
// (repeated Pair pairs = 1), the answer of a subspace query.
func countKVPairs(b []byte) (int, error) {
	n := 0
	for len(b) > 0 {
		if b[0] != 0x0a { // field 1, length-delimited
			return 0, fmt.Errorf("unexpected tag %#x in kv pairs", b[0])
		}
		size, read := binary.Uvarint(b[1:])
		if read <= 0 || uint64(len(b)-1-read) < size {
			return 0, fmt.Errorf("truncated kv pair")
		}
		b = b[1+read+int(size):]
		n++
	}
	return n, nil
}
