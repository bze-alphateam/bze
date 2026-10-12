// Package helpers drives a running bze e2e chain through the bzed CLI: it
// sends transactions, runs queries, reads tx and block events, waits for
// epochs and passes governance proposals. It never imports the chain module;
// every message goes through `bzed tx` and every read through `bzed q` (JSON
// output) or CometBFT RPC, so the suites test the binary exactly as users run it.
package helpers
