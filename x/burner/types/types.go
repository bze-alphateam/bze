package types

const (
	// RaffleModuleName used to mimic a module in SDK's account module.
	RaffleModuleName = "burner_raffle"
	// BlackHoleModuleName used to mimic a module in SDK's account module.
	BlackHoleModuleName = "burner_black_hole"

	// MaxDenomsBurnPerBlock is the maximum number of denominations to process
	// in a single block during periodic burn queue processing.
	MaxDenomsBurnPerBlock = 100
)
