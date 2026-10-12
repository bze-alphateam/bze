package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
)

// Query runs `bzed q <args> --output json` on the first validator and decodes
// the output into T. Suites declare small local structs for T; 64-bit integers
// come back as strings (use Int64 or parse them with cosmossdk.io/math).
func Query[T any](ctx context.Context, chain *cosmos.CosmosChain, args ...string) (T, error) {
	var out T
	stdout, stderr, err := chain.GetNode().ExecQuery(ctx, args...)
	if err != nil {
		return out, fmt.Errorf("bzed q %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(stderr)))
	}
	if err := json.Unmarshal(stdout, &out); err != nil {
		return out, fmt.Errorf("decode bzed q %s output %q: %w", strings.Join(args, " "), stdout, err)
	}
	return out, nil
}

// Balance returns the address's balance of denom.
func Balance(ctx context.Context, chain *cosmos.CosmosChain, addr, denom string) (sdkmath.Int, error) {
	return chain.GetBalance(ctx, addr, denom)
}

// ModuleAddress returns the bech32 address of a module account. It is derived
// the same way the chain derives it (the first 20 bytes of sha256(name)), so it
// also works for module accounts that do not exist on chain yet.
func ModuleAddress(_ context.Context, chain *cosmos.CosmosChain, moduleName string) (string, error) {
	return sdk.Bech32ifyAddressBytes(chain.Config().Bech32Prefix, authtypes.NewModuleAddress(moduleName))
}

// GovAuthority returns the gov module address, the authority of every
// governance-gated message (MsgUpdateParams, MsgHaltDenoms, …).
func GovAuthority(ctx context.Context, chain *cosmos.CosmosChain) (string, error) {
	return ModuleAddress(ctx, chain, "gov")
}
