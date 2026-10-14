package helpers

import (
	"context"
	"fmt"
	"sync"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
)

// The wait helpers follow each chain's blocks over one CometBFT WebSocket
// subscription (NewBlock) instead of polling the CLI: every waiter of a chain
// reads the same stream, so a long wait (an epoch, a proposal's voting period)
// costs no CLI process until the block it waits for arrives. The CLI still
// runs every action and reads every result the suites assert on.
//
// The stream is gapless: a block missed by the WebSocket (a dropped message, a
// reconnect, a node restart) is fetched over RPC before the next one is
// handed out. It starts on the first wait and stops after streamIdle without
// waiters.

const (
	// streamLookback is how many blocks before a waiter's start it also checks,
	// so a block committed just before the waiter subscribed (and not yet in the
	// tx index) is not missed.
	streamLookback = 5
	// streamKeep is how many recent blocks the stream holds.
	streamKeep = 200
	// streamIdle stops a stream nobody waited on for this long.
	streamIdle = time.Minute
	// streamStall is how long the stream waits for a WebSocket block before it
	// asks the node for the latest height over RPC.
	streamStall = 3 * time.Second
	// streamReconnectAfter is how many failed RPC catch-ups in a row make the
	// stream reconnect, re-reading the node's host address (a restarted node
	// may have a new host port).
	streamReconnectAfter = 3
	// streamSubscriber names the stream's subscription on the node.
	streamSubscriber = "bze-e2e-blocks"
)

// blockTx is one transaction of a streamed block.
type blockTx struct {
	Hash   string // upper-case hex, the form `q tx` prints
	Code   uint32
	Events []Event
}

// streamBlock is a committed block: its finalize-block events (BeginBlock,
// EndBlock, epoch hooks) and its transactions.
type streamBlock struct {
	Height int64
	Events []Event
	Txs    []blockTx
}

// blockStream follows one chain's blocks for every waiter of that chain.
type blockStream struct {
	chain *cosmos.CosmosChain

	mu       sync.Mutex
	running  bool
	waiters  int
	lastUsed time.Time
	height   int64
	blocks   map[int64]streamBlock
	next     chan struct{} // closed and replaced when a block arrives
}

var (
	streamsMu sync.Mutex
	streams   = map[*cosmos.CosmosChain]*blockStream{}
)

func streamOf(chain *cosmos.CosmosChain) *blockStream {
	streamsMu.Lock()
	defer streamsMu.Unlock()
	s, ok := streams[chain]
	if !ok {
		s = &blockStream{chain: chain, blocks: map[int64]streamBlock{}, next: make(chan struct{})}
		streams[chain] = s
	}
	return s
}

// blockWatch is one waiter on a chain's block stream. Its first until call
// also sees the streamLookback blocks before it subscribed.
type blockWatch struct {
	stream *blockStream
	from   int64
	once   sync.Once
}

// watchBlocks subscribes a waiter to the chain's block stream, starting the
// stream if it is not running. Close the watch when done.
func watchBlocks(ctx context.Context, chain *cosmos.CosmosChain) (*blockWatch, error) {
	s := streamOf(chain)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		if err := s.startLocked(ctx); err != nil {
			return nil, err
		}
	}
	s.waiters++
	s.lastUsed = time.Now()
	return &blockWatch{stream: s, from: s.height - streamLookback}, nil
}

// Close releases the watch.
func (w *blockWatch) Close() {
	w.once.Do(func() {
		s := w.stream
		s.mu.Lock()
		s.waiters--
		s.lastUsed = time.Now()
		s.mu.Unlock()
	})
}

// until hands match every block after the watch's position, in height order,
// and returns the first block it accepts. It fails when match fails or ctx
// ends.
func (w *blockWatch) until(ctx context.Context, match func(streamBlock) (bool, error)) (streamBlock, error) {
	s := w.stream
	for {
		s.mu.Lock()
		height, next := s.height, s.next
		var pending []streamBlock
		for h := w.from + 1; h <= height; h++ {
			if b, ok := s.blocks[h]; ok {
				pending = append(pending, b)
			}
		}
		s.mu.Unlock()

		for _, b := range pending {
			w.from = b.Height
			ok, err := match(b)
			if err != nil {
				return streamBlock{}, err
			}
			if ok {
				return b, nil
			}
		}
		if height > w.from {
			w.from = height
		}
		select {
		case <-ctx.Done():
			return streamBlock{}, ctx.Err()
		case <-next:
		}
	}
}

// startLocked loads the latest blocks over RPC and starts following the
// chain. The caller holds s.mu.
func (s *blockStream) startLocked(ctx context.Context) error {
	client, err := s.newClient()
	if err != nil {
		return err
	}
	status, err := client.Status(ctx)
	if err != nil {
		return fmt.Errorf("block stream of %s: status: %w", s.chain.Config().ChainID, err)
	}
	latest := status.SyncInfo.LatestBlockHeight
	start := max(latest-streamLookback, 1)
	if s.height < start-1 || s.height > latest {
		// Stale or never started: begin again from the recent blocks.
		s.blocks = map[int64]streamBlock{}
		s.height = start - 1
	}
	for h := s.height + 1; h <= latest; h++ {
		b, err := fetchBlock(ctx, client, h)
		if err != nil {
			return fmt.Errorf("block stream of %s: %w", s.chain.Config().ChainID, err)
		}
		s.addLocked(b)
	}
	s.running = true
	go s.run()
	return nil
}

func (s *blockStream) newClient() (*rpchttp.HTTP, error) {
	addr := s.chain.GetHostRPCAddress()
	client, err := rpchttp.New(addr, "/websocket")
	if err != nil {
		return nil, fmt.Errorf("block stream of %s: rpc client %s: %w", s.chain.Config().ChainID, addr, err)
	}
	return client, nil
}

// run follows the chain until the stream is idle, reconnecting when the
// WebSocket or the node goes away.
func (s *blockStream) run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		if s.stopIfIdle() {
			return
		}
		client, err := s.newClient()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		s.follow(ctx, client)
		_ = client.Stop()
	}
}

// follow reads NewBlock events from one connection. Without a WebSocket (the
// node is down or refused the subscription) it keeps catching up over RPC.
// It returns when the stream is idle or the connection keeps failing.
func (s *blockStream) follow(ctx context.Context, client *rpchttp.HTTP) {
	var events <-chan coretypes.ResultEvent
	if err := client.Start(); err == nil {
		subCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		out, err := client.Subscribe(subCtx, streamSubscriber, comettypes.QueryForEvent(comettypes.EventNewBlock).String(), 100)
		cancel()
		if err == nil {
			events = out
		}
	}

	ticker := time.NewTicker(streamStall)
	defer ticker.Stop()
	lastBlock := time.Now()
	failures := 0
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return
			}
			data, isBlock := ev.Data.(comettypes.EventDataNewBlock)
			if !isBlock || data.Block == nil {
				continue
			}
			if err := s.catchUp(ctx, client, data.Block.Height-1); err != nil {
				failures++
				continue
			}
			s.add(newBlockFromEvent(data))
			lastBlock, failures = time.Now(), 0
		case <-ticker.C:
			if s.stopIfIdle() {
				return
			}
			if time.Since(lastBlock) < streamStall {
				continue
			}
			status, err := client.Status(ctx)
			if err == nil {
				err = s.catchUp(ctx, client, status.SyncInfo.LatestBlockHeight)
			}
			if err != nil {
				failures++
				if failures >= streamReconnectAfter {
					return
				}
				continue
			}
			failures = 0
			if events == nil {
				// Polling only: try the WebSocket again.
				return
			}
		}
	}
}

// catchUp fetches every block after the stream's height up to h over RPC.
func (s *blockStream) catchUp(ctx context.Context, client rpcclient.Client, h int64) error {
	s.mu.Lock()
	from := s.height + 1
	s.mu.Unlock()
	for height := from; height <= h; height++ {
		b, err := fetchBlock(ctx, client, height)
		if err != nil {
			return err
		}
		s.add(b)
	}
	return nil
}

func (s *blockStream) stopIfIdle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.waiters == 0 && time.Since(s.lastUsed) > streamIdle {
		s.running = false
		return true
	}
	return false
}

func (s *blockStream) add(b streamBlock) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addLocked(b)
}

func (s *blockStream) addLocked(b streamBlock) {
	if b.Height != s.height+1 {
		// Already seen, or a gap catchUp has to fill first.
		return
	}
	s.blocks[b.Height] = b
	s.height = b.Height
	delete(s.blocks, b.Height-streamKeep)
	close(s.next)
	s.next = make(chan struct{})
}

// fetchBlock reads one block and its results over RPC.
func fetchBlock(ctx context.Context, client rpcclient.Client, h int64) (streamBlock, error) {
	block, err := client.Block(ctx, &h)
	if err != nil {
		return streamBlock{}, fmt.Errorf("block %d: %w", h, err)
	}
	results, err := client.BlockResults(ctx, &h)
	if err != nil {
		return streamBlock{}, fmt.Errorf("block_results %d: %w", h, err)
	}
	return newStreamBlock(h, block.Block.Txs, results.TxsResults, results.FinalizeBlockEvents), nil
}

func newBlockFromEvent(data comettypes.EventDataNewBlock) streamBlock {
	return newStreamBlock(data.Block.Height, data.Block.Txs, data.ResultFinalizeBlock.TxResults, data.ResultFinalizeBlock.Events)
}

func newStreamBlock(height int64, txs comettypes.Txs, results []*abci.ExecTxResult, events []abci.Event) streamBlock {
	b := streamBlock{Height: height, Events: abciEvents(events)}
	for i, tx := range txs {
		t := blockTx{Hash: fmt.Sprintf("%X", tx.Hash())}
		if i < len(results) && results[i] != nil {
			t.Code = results[i].Code
			t.Events = abciEvents(results[i].Events)
		}
		b.Txs = append(b.Txs, t)
	}
	return b
}

func abciEvents(events []abci.Event) []Event {
	out := make([]Event, 0, len(events))
	for _, e := range events {
		attrs := make([]rawAttribute, 0, len(e.Attributes))
		for _, a := range e.Attributes {
			attrs = append(attrs, rawAttribute{Key: a.Key, Value: a.Value})
		}
		out = append(out, newEvent(e.Type, attrs))
	}
	return out
}
