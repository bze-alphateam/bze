package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
)

// typedEventPrefix marks typed (protobuf) events emitted by bze modules, e.g.
// bze.epochs.EpochStartEvent. Their attribute values are JSON encoded.
const typedEventPrefix = "bze."

// Event is an ABCI event with its attributes keyed by name. Attribute values of
// typed bze events (type bze.<module>.<EventName>) are unquoted, so a string
// field reads as `day`, not `"day"`. If an event repeats an attribute key, the
// last value wins.
type Event struct {
	Type       string
	Attributes map[string]string
}

// Unquote decodes a JSON string value (`"day"` → `day`). Values that are not
// JSON strings (numbers, objects, plain text) are returned unchanged.
func Unquote(v string) string {
	if len(v) < 2 || v[0] != '"' || v[len(v)-1] != '"' {
		return v
	}
	var s string
	if err := json.Unmarshal([]byte(v), &s); err != nil {
		return v
	}
	return s
}

type rawAttribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type rawEvent struct {
	Type       string         `json:"type"`
	Attributes []rawAttribute `json:"attributes"`
}

func newEvent(typ string, attrs []rawAttribute) Event {
	e := Event{Type: typ, Attributes: make(map[string]string, len(attrs))}
	typed := strings.HasPrefix(typ, typedEventPrefix)
	for _, a := range attrs {
		v := a.Value
		if typed {
			v = Unquote(v)
		}
		e.Attributes[a.Key] = v
	}
	return e
}

func convertEvents(raw []rawEvent) []Event {
	events := make([]Event, 0, len(raw))
	for _, r := range raw {
		events = append(events, newEvent(r.Type, r.Attributes))
	}
	return events
}

func filterEvents(events []Event, eventType string) []Event {
	var out []Event
	for _, e := range events {
		if e.Type == eventType {
			out = append(out, e)
		}
	}
	return out
}

// TxEvents returns the events of the given type emitted by a transaction.
func TxEvents(res TxResult, eventType string) []Event {
	return filterEvents(res.Events, eventType)
}

// BlockEvents returns the finalize-block events (BeginBlock/EndBlock, epoch
// hooks) of the given type at a height, read from CometBFT block_results.
// Transaction events are not included; use TxEvents for those.
func BlockEvents(ctx context.Context, chain *cosmos.CosmosChain, height int64, eventType string) ([]Event, error) {
	node := chain.GetNode()
	if node.Client == nil {
		return nil, fmt.Errorf("node %s has no RPC client", node.Name())
	}
	res, err := node.Client.BlockResults(ctx, &height)
	if err != nil {
		return nil, fmt.Errorf("block_results at height %d: %w", height, err)
	}
	var out []Event
	for _, e := range res.FinalizeBlockEvents {
		if e.Type != eventType {
			continue
		}
		attrs := make([]rawAttribute, 0, len(e.Attributes))
		for _, a := range e.Attributes {
			attrs = append(attrs, rawAttribute{Key: a.Key, Value: a.Value})
		}
		out = append(out, newEvent(e.Type, attrs))
	}
	return out, nil
}

// EventsWith returns the events whose attributes include every key/value pair
// of attrs, e.g. the DenomRewardDistributionEvent of one staking denom.
func EventsWith(events []Event, attrs map[string]string) []Event {
	var out []Event
	for _, e := range events {
		match := true
		for k, v := range attrs {
			if got, ok := e.Attributes[k]; !ok || got != v {
				match = false
				break
			}
		}
		if match {
			out = append(out, e)
		}
	}
	return out
}
