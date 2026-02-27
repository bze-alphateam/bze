package types

const (
	// RaffleModuleName used to mimic a module in SDK's account module.
	RaffleModuleName = "burner_raffle"
	// BlackHoleModuleName used to mimic a module in SDK's account module.
	BlackHoleModuleName = "burner_black_hole"

	// MaxDenomsBurnPerBlock is the maximum number of denominations to process
	// in a single block during periodic burn queue processing.
	MaxDenomsBurnPerBlock = 100

	// MaxRafflesCleanupPerBlock is the maximum number of raffles to clean up
	// in a single block during raffle cleanup queue processing.
	MaxRafflesCleanupPerBlock = 50
)
