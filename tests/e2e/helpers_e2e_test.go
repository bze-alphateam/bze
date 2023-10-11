package e2e

import (
	sdkmath "cosmossdk.io/math"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/tests/e2e/configurer/chain"
)

var defaultFeePerTx = sdkmath.NewInt(1000)

// calculateSpreadRewardGrowthGlobal calculates spread reward growth global per unit of virtual liquidity based on swap parameters:
// amountIn - amount being swapped
// spreadFactor - pool's spread factor
// poolLiquidity - current pool liquidity
func calculateSpreadRewardGrowthGlobal(spreadRewardChargeTotal, poolLiquidity sdkmath.LegacyDec) sdkmath.LegacyDec {
	// First we get total spread reward charge for the swap (ΔY * spreadFactor)

	// Calculating spread reward growth global (dividing by pool liquidity to find spread reward growth per unit of virtual liquidity)
	spreadRewardGrowthGlobal := spreadRewardChargeTotal.QuoTruncate(poolLiquidity)
	return spreadRewardGrowthGlobal
}

// calculateSpreadRewardGrowthInside calculates spread reward growth inside range per unit of virtual liquidity
// spreadRewardGrowthGlobal - global spread reward growth per unit of virtual liquidity
// spreadRewardGrowthBelow - spread reward growth below lower tick
// spreadRewardGrowthAbove - spread reward growth above upper tick
// Formula: spreadRewardGrowthGlobal - spreadRewardGrowthBelowLowerTick - spreadRewardGrowthAboveUpperTick
func calculateSpreadRewardGrowthInside(spreadRewardGrowthGlobal, spreadRewardGrowthBelow, spreadRewardGrowthAbove sdkmath.LegacyDec) sdkmath.LegacyDec {
	return spreadRewardGrowthGlobal.Sub(spreadRewardGrowthBelow).Sub(spreadRewardGrowthAbove)
}

// Assert balances that are not affected by swap:
// * same amount of `stake` in balancesBefore and balancesAfter
// * amount of `e2e-default-feetoken` dropped by 1000 (default amount for fee per tx)
// * depending on `assertUosmoBalanceIsConstant` and `assertUionBalanceIsConstant` parameters, check that those balances have also not been changed
func (s *IntegrationTestSuite) assertBalancesInvariants(balancesBefore, balancesAfter sdk.Coins, assertUosmoBalanceIsConstant, assertUionBalanceIsConstant bool) {
	s.Require().True(balancesAfter.AmountOf("stake").Equal(balancesBefore.AmountOf("stake")))
	s.Require().True(balancesAfter.AmountOf("e2e-default-feetoken").Equal(balancesBefore.AmountOf("e2e-default-feetoken").Sub(defaultFeePerTx)))
	if assertUionBalanceIsConstant {
		s.Require().True(balancesAfter.AmountOf("uion").Equal(balancesBefore.AmountOf("uion")))
	}
	if assertUosmoBalanceIsConstant {
		s.Require().True(balancesAfter.AmountOf("uosmo").Equal(balancesBefore.AmountOf("uosmo")))
	}
}

// Get balances for address
func (s *IntegrationTestSuite) addrBalance(node *chain.NodeConfig, address string) sdk.Coins {
	addrBalances, err := node.QueryBalances(address)
	s.Require().NoError(err)
	return addrBalances
}

var currentNodeIndexA int

func (s *IntegrationTestSuite) getChainACfgs() (*chain.Config, *chain.NodeConfig, error) {
	chainA := s.configurer.GetChainConfig(0)

	chainANodes := chainA.GetAllChainNodes()

	chosenNode := chainANodes[currentNodeIndexA]
	currentNodeIndexA = (currentNodeIndexA + 1) % len(chainANodes)
	return chainA, chosenNode, nil
}

var currentNodeIndexB int

func (s *IntegrationTestSuite) getChainBCfgs() (*chain.Config, *chain.NodeConfig, error) {
	chainB := s.configurer.GetChainConfig(1)

	chainBNodes := chainB.GetAllChainNodes()

	chosenNode := chainBNodes[currentNodeIndexB]
	currentNodeIndexB = (currentNodeIndexB + 1) % len(chainBNodes)
	return chainB, chosenNode, nil
}

var useChainA bool

func (s *IntegrationTestSuite) getChainCfgs() (*chain.Config, *chain.NodeConfig, error) {
	if useChainA {
		useChainA = false
		return s.getChainACfgs()
	} else {
		useChainA = true
		return s.getChainBCfgs()
	}
}

// Helper function for calculating uncollected spread rewards since the time that spreadRewardGrowthInsideLast corresponds to
// positionLiquidity - current position liquidity
// spreadRewardGrowthBelow - spread reward growth below lower tick
// spreadRewardGrowthAbove - spread reward growth above upper tick
// spreadRewardGrowthInsideLast - amount of spread reward growth inside range at the time from which we want to calculate the amount of uncollected spread rewards
// spreadRewardGrowthGlobal - variable for tracking global spread reward growth
func calculateUncollectedSpreadRewards(positionLiquidity, spreadRewardGrowthBelow, spreadRewardGrowthAbove, spreadRewardGrowthInsideLast sdkmath.LegacyDec, spreadRewardGrowthGlobal sdkmath.LegacyDec) sdkmath.LegacyDec {
	// Calculating spread reward growth inside range [-1200; 400]
	spreadRewardGrowthInside := calculateSpreadRewardGrowthInside(spreadRewardGrowthGlobal, spreadRewardGrowthBelow, spreadRewardGrowthAbove)

	// Calculating uncollected spread rewards
	// Formula for finding uncollected spread rewards in time range [t1; t2]:
	// F_u = position_liquidity * (spread_rewards_growth_inside_t2 - spread_rewards_growth_inside_t1).
	spreadRewardsUncollected := positionLiquidity.Mul(spreadRewardGrowthInside.Sub(spreadRewardGrowthInsideLast))

	return spreadRewardsUncollected
}

func (s *IntegrationTestSuite) CallCheckBalance(node *chain.NodeConfig, addr, denom string, amount int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.CheckBalance(node, addr, denom, amount)
}

// CheckBalance Checks the balance of an address
func (s *IntegrationTestSuite) CheckBalance(node *chain.NodeConfig, addr, denom string, amount int64) {
	// check the balance of the contract
	s.Require().Eventually(func() bool {
		// TODO: Change to QueryBalance(addr, denom)
		balance, err := node.QueryBalances(addr)
		s.Require().NoError(err)
		if len(balance) == 0 {
			return false
		}
		// check that the amount is in one of the balances inside the balance list
		for _, b := range balance {
			if b.Denom == denom && b.Amount.Int64() == amount {
				return true
			}
		}
		return false
	},
		1*time.Minute,
		10*time.Millisecond,
	)
}

func (s *IntegrationTestSuite) getChainIndex(chain *chain.Config) int {
	if chain.Id == "bze-test-a" {
		return 0
	} else {
		return 1
	}
}
