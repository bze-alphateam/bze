package keeper

import (
	"crypto/sha256"
	"github.com/bze-alphateam/bze/x/burner/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"math/big"
)

const (
	randomRangeMax = 1_000_000
)

// GenerateDeterministicRandomNumber generates a deterministic random number between 0 and 1,000,000 from the provided string input
func generateDeterministicRandomNumber(input []byte, randomRangeMax uint64) uint64 {
	// Hash the input using SHA-256
	hash := sha256.Sum256(input)
	number := new(big.Int).SetBytes(hash[:])

	// Take the modulus to get the number between 0 and randomRangeMax-1
	modulus := new(big.Int).SetUint64(randomRangeMax)
	deterministicNumber := number.Mod(number, modulus).Uint64()

	return deterministicNumber
}

func (k Keeper) IsLucky(ctx sdk.Context, raffle *types.Raffle, address string) bool {
	seed := append(ctx.HeaderHash(), ctx.BlockHeader().AppHash...)
	seed = append(seed, []byte(address)...)

	return generateDeterministicRandomNumber(seed, randomRangeMax) < raffle.Chances
}
