package e2e

import (
	sdkmath "cosmossdk.io/math"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/tests/e2e/configurer/chain"
	"github.com/bze-alphateam/bze/tests/e2e/configurer/config"
	"github.com/bze-alphateam/bze/tests/e2e/initialization"
)

var (
	// minDecTolerance minimum tolerance for osmomath.Dec, given its precision of 18.
	minDecTolerance = sdkmath.LegacyMustNewDecFromStr("0.000000000000000001")
)

// TODO: Find more scalable way to do this
func (s *IntegrationTestSuite) TestAllE2E() {

	// Test currently disabled
	// s.T().Run("ArithmeticTWAP", func(t *testing.T) {
	// 	t.Parallel()
	// 	s.ArithmeticTWAP()
	// })

	// State Sync Dependent Tests

	if s.skipStateSync {
		s.T().Skip()
	} else {
		s.T().Run("StateSync", func(t *testing.T) {
			t.Parallel()
			s.StateSync()
		})
	}

	if s.skipIBC {
		s.T().Skip("Skipping IBC tests")
	} else {
		s.T().Run("IBCTokenTransferAndCreatePool", func(t *testing.T) {
			t.Parallel()
			s.IBCTokenTransferAndCreatePool()
		})
	}
}

// TestIBCTokenTransfer tests that IBC token transfers work as expected.
// Additionally, it attempst to create a pool with IBC denoms.
func (s *IntegrationTestSuite) IBCTokenTransferAndCreatePool() {
	if s.skipIBC {
		s.T().Skip("Skipping IBC tests")
	}
	chainA, chainANode, err := s.getChainACfgs()
	s.Require().NoError(err)
	chainB, chainBNode, err := s.getChainBCfgs()
	s.Require().NoError(err)

	chainANode.SendIBC(chainA, chainB, chainBNode.PublicAddress, initialization.BzeToken)
	chainBNode.SendIBC(chainB, chainA, chainANode.PublicAddress, initialization.BzeToken)
	chainANode.SendIBC(chainA, chainB, chainBNode.PublicAddress, initialization.StakeToken)
	chainBNode.SendIBC(chainB, chainA, chainANode.PublicAddress, initialization.StakeToken)

	chainANode.CreateBalancerPool("ibcDenomPool.json", initialization.ValidatorWalletName)
}

func (s *IntegrationTestSuite) StateSync() {
	if s.skipStateSync {
		s.T().Skip()
	}

	// This test benefits from the use of chainA's default node, since it has
	// the shortest snapshot interval.
	chainA := s.configurer.GetChainConfig(0)
	chainANode, err := chainA.GetDefaultNode()
	s.Require().NoError(err)

	persistentPeers := chainA.GetPersistentPeers()

	stateSyncHostPort := fmt.Sprintf("%s:26657", chainANode.Name)
	stateSyncRPCServers := []string{stateSyncHostPort, stateSyncHostPort}

	// get trust height and trust hash.
	trustHeight, err := chainANode.QueryCurrentHeight()
	s.Require().NoError(err)

	trustHash, err := chainANode.QueryHashFromBlock(trustHeight)
	s.Require().NoError(err)

	stateSynchingNodeConfig := &initialization.NodeConfig{
		Name:               "state-sync",
		Pruning:            "default",
		PruningKeepRecent:  "0",
		PruningInterval:    "0",
		SnapshotInterval:   1500,
		SnapshotKeepRecent: 2,
	}

	tempDir, err := os.MkdirTemp("", "osmosis-e2e-statesync-")
	s.Require().NoError(err)

	// configure genesis and config files for the state-synchin node.
	nodeInit, err := initialization.InitSingleNode(
		chainA.Id,
		tempDir,
		filepath.Join(chainANode.ConfigDir, "config", "genesis.json"),
		stateSynchingNodeConfig,
		time.Duration(chainA.VotingPeriod),
		// time.Duration(chainA.ExpeditedVotingPeriod),
		trustHeight,
		trustHash,
		stateSyncRPCServers,
		persistentPeers,
	)
	s.Require().NoError(err)

	// Call tempNode method here to not add the node to the list of nodes.
	// This messes with the nodes running in parallel if we add it to the regular list.
	stateSynchingNode := chainA.CreateNodeTemp(nodeInit)

	// ensure that the running node has snapshots at a height > trustHeight.
	hasSnapshotsAvailable := func(syncInfo coretypes.SyncInfo) bool {
		snapshotHeight := chainANode.SnapshotInterval
		if uint64(syncInfo.LatestBlockHeight) < snapshotHeight {
			s.T().Logf("snapshot height is not reached yet, current (%d), need (%d)", syncInfo.LatestBlockHeight, snapshotHeight)
			return false
		}

		snapshots, err := chainANode.QueryListSnapshots()
		s.Require().NoError(err)

		for _, snapshot := range snapshots {
			if snapshot.Height > uint64(trustHeight) {
				s.T().Log("found state sync snapshot after trust height")
				return true
			}
		}
		s.T().Log("state sync snashot after trust height is not found")
		return false
	}
	chainANode.WaitUntil(hasSnapshotsAvailable)

	// start the state synchin node.
	err = stateSynchingNode.Run()
	s.Require().NoError(err)

	// ensure that the state synching node cathes up to the running node.
	s.Require().Eventually(func() bool {
		stateSyncNodeHeight, err := stateSynchingNode.QueryCurrentHeight()
		s.Require().NoError(err)
		runningNodeHeight, err := chainANode.QueryCurrentHeight()
		s.Require().NoError(err)
		return stateSyncNodeHeight == runningNodeHeight
	},
		1*time.Minute,
		10*time.Millisecond,
	)

	// stop the state synching node.
	err = chainA.RemoveTempNode(stateSynchingNode.Name)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) ExpeditedProposals() {
	chainAB, chainABNode, err := s.getChainCfgs()
	s.Require().NoError(err)

	propNumber := chainABNode.SubmitTextProposal("expedited text proposal", sdk.NewCoin(initialization.BzeDenom, sdkmath.NewInt(config.InitialMinExpeditedDeposit)), true)

	chainABNode.DepositProposal(propNumber, true)
	totalTimeChan := make(chan time.Duration, 1)
	go chainABNode.QueryPropStatusTimed(propNumber, "PROPOSAL_STATUS_PASSED", totalTimeChan)

	var wg sync.WaitGroup

	for _, n := range chainAB.NodeConfigs {
		wg.Add(1)
		go func(nodeConfig *chain.NodeConfig) {
			defer wg.Done()
			nodeConfig.VoteYesProposal(initialization.ValidatorWalletName, propNumber)
		}(n)
	}

	wg.Wait()

	// if querying proposal takes longer than timeoutPeriod, stop the goroutine and error
	var elapsed time.Duration
	timeoutPeriod := 2 * time.Minute
	select {
	case elapsed = <-totalTimeChan:
	case <-time.After(timeoutPeriod):
		err := fmt.Errorf("go routine took longer than %s", timeoutPeriod)
		s.Require().NoError(err)
	}

	// compare the time it took to reach pass status to expected expedited voting period
	expeditedVotingPeriodDuration := time.Duration(chainAB.ExpeditedVotingPeriod * float32(time.Second))
	timeDelta := elapsed - expeditedVotingPeriodDuration
	// ensure delta is within two seconds of expected time
	s.Require().Less(timeDelta, 2*time.Second)
	s.T().Logf("expeditedVotingPeriodDuration within two seconds of expected time: %v", timeDelta)
	close(totalTimeChan)
}
