package configurer

import (
	sdkmath "cosmossdk.io/math"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/tests/e2e/configurer/chain"
	"github.com/bze-alphateam/bze/tests/e2e/configurer/config"
	"github.com/bze-alphateam/bze/tests/e2e/containers"
	"github.com/bze-alphateam/bze/tests/e2e/initialization"
)

type UpgradeSettings struct {
	IsEnabled  bool
	Version    string
	ForkHeight int64 // non-zero height implies that this is a fork upgrade.
}

type UpgradeConfigurer struct {
	baseConfigurer
	upgradeVersion string
	forkHeight     int64 // forkHeight > 0 implies that this is a fork upgrade. Otherwise, proposal upgrade.
}

var _ Configurer = (*UpgradeConfigurer)(nil)

func NewUpgradeConfigurer(t *testing.T, chainConfigs []*chain.Config, setupTests setupFn, containerManager *containers.Manager, upgradeVersion string, forkHeight int64) Configurer {
	t.Helper()
	return &UpgradeConfigurer{
		baseConfigurer: baseConfigurer{
			chainConfigs:     chainConfigs,
			containerManager: containerManager,
			setupTests:       setupTests,
			syncUntilHeight:  forkHeight + defaultSyncUntilHeight,
			t:                t,
		},
		forkHeight:     forkHeight,
		upgradeVersion: upgradeVersion,
	}
}

func (uc *UpgradeConfigurer) ConfigureChains() error {
	for _, chainConfig := range uc.chainConfigs {
		if err := uc.ConfigureChain(chainConfig); err != nil {
			return err
		}
	}
	return nil
}

func (uc *UpgradeConfigurer) ConfigureChain(chainConfig *chain.Config) error {
	uc.t.Logf("starting upgrade e2e infrastructure for chain-id: %s", chainConfig.Id)
	tmpDir, err := os.MkdirTemp("", "bze-e2e-testnet-")
	if err != nil {
		return err
	}

	validatorConfigBytes, err := json.Marshal(chainConfig.ValidatorInitConfigs)
	if err != nil {
		return err
	}

	forkHeight := uc.forkHeight
	if forkHeight > 0 {
		forkHeight = forkHeight - config.ForkHeightPreUpgradeOffset
	}

	chainInitResource, err := uc.containerManager.RunChainInitResource(chainConfig.Id, int(chainConfig.VotingPeriod), int(chainConfig.ExpeditedVotingPeriod), validatorConfigBytes, tmpDir, int(forkHeight))
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%v/%v-encode", tmpDir, chainConfig.Id)
	uc.t.Logf("serialized init file for chain-id %v: %v", chainConfig.Id, fileName)

	// loop through the reading and unmarshaling of the init file a total of maxRetries or until error is nil
	// without this, test attempts to unmarshal file before docker container is finished writing
	var initializedChain initialization.Chain
	for i := 0; i < config.MaxRetries; i++ {
		initializedChainBytes, _ := os.ReadFile(fileName)
		err = json.Unmarshal(initializedChainBytes, &initializedChain)
		if err == nil {
			break
		}

		if i == config.MaxRetries-1 {
			if err != nil {
				return err
			}
		}

		if i > 0 {
			time.Sleep(1 * time.Second)
		}
	}
	if err := uc.containerManager.PurgeResource(chainInitResource); err != nil {
		return err
	}
	uc.initializeChainConfigFromInitChain(&initializedChain, chainConfig)
	return nil
}

func (uc *UpgradeConfigurer) CreatePreUpgradeState() error {
	// Create a WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup
	chainA := uc.chainConfigs[0]
	chainANode, err := chainA.GetDefaultNode()
	if err != nil {
		return err
	}
	chainB := uc.chainConfigs[1]
	chainBNode, err := chainB.GetDefaultNode()
	if err != nil {
		return err
	}

	wg.Add(2)

	go func() {
		defer wg.Done()
		chainA.SendIBC(chainB, chainBNode.PublicAddress, initialization.BzeToken)
		chainA.SendIBC(chainB, chainBNode.PublicAddress, initialization.StakeToken)
	}()

	go func() {
		defer wg.Done()
		chainB.SendIBC(chainA, chainANode.PublicAddress, initialization.BzeToken)
		chainB.SendIBC(chainA, chainANode.PublicAddress, initialization.StakeToken)
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	var (
		poolShareDenom             = make([]string, 2)
		preUpgradePoolId           = make([]uint64, 2)
		preUpgradeStableSwapPoolId = make([]uint64, 2)
	)

	config.PreUpgradePoolId = preUpgradePoolId
	config.PreUpgradeStableSwapPoolId = preUpgradeStableSwapPoolId

	var (
		lockupWallet           = make([]string, 2)
		lockupWalletSuperfluid = make([]string, 2)
		stableswapWallet       = make([]string, 2)
	)

	wg.Add(6)

	// Chain A
	go func() {
		defer wg.Done()
		// Setup wallets and send tokens to wallets (only chainA)
		lockupWallet[0] = chainANode.CreateWalletAndFund(config.LockupWallet[0], []string{
			"10000000000000000000" + poolShareDenom[0],
		}, chainA)
	}()

	go func() {
		defer wg.Done()
		lockupWalletSuperfluid[0] = chainANode.CreateWalletAndFund(config.LockupWalletSuperfluid[0], []string{
			"10000000000000000000" + poolShareDenom[0],
		}, chainA)
	}()

	go func() {
		defer wg.Done()
		stableswapWallet[0] = chainANode.CreateWalletAndFund(config.StableswapWallet[0], []string{
			"100000stake",
		}, chainA)
	}()

	// Chain B
	go func() {
		defer wg.Done()
		// Setup wallets and send tokens to wallets (only chainA)
		lockupWallet[1] = chainBNode.CreateWalletAndFund(config.LockupWallet[1], []string{
			"10000000000000000000" + poolShareDenom[1],
		}, chainB)
	}()

	go func() {
		defer wg.Done()
		lockupWalletSuperfluid[1] = chainBNode.CreateWalletAndFund(config.LockupWalletSuperfluid[1], []string{
			"10000000000000000000" + poolShareDenom[1],
		}, chainB)
	}()

	go func() {
		defer wg.Done()
		stableswapWallet[1] = chainBNode.CreateWalletAndFund(config.StableswapWallet[1], []string{
			"100000stake",
		}, chainB)
	}()

	wg.Wait()

	config.LockupWallet = lockupWallet
	config.LockupWalletSuperfluid = lockupWalletSuperfluid
	config.StableswapWallet = stableswapWallet

	return nil
}

func (uc *UpgradeConfigurer) RunSetup() error {
	return uc.setupTests(uc)
}

func (uc *UpgradeConfigurer) RunUpgrade() error {
	var err error
	if uc.forkHeight > 0 {
		uc.runForkUpgrade()
	} else {
		err = uc.runProposalUpgrade()
	}
	if err != nil {
		return err
	}

	// Check if the nodes are running
	for chainIndex, chainConfig := range uc.chainConfigs {
		chain := uc.baseConfigurer.GetChainConfig(chainIndex)
		for validatorIdx := range chainConfig.NodeConfigs {
			node := chain.NodeConfigs[validatorIdx]
			// Check node status
			_, err = node.Status()
			if err != nil {
				uc.t.Errorf("node is not running after upgrade, chain-id %s, node %s", chainConfig.Id, node.Name)
				return err
			}
			uc.t.Logf("node %s upgraded successfully, address %s", node.Name, node.PublicAddress)
		}
	}
	return nil
}

func (uc *UpgradeConfigurer) runProposalUpgrade() error {
	// submit, deposit, and vote for upgrade proposal
	// prop height = current height + voting period + time it takes to submit proposal + small buffer
	for _, chainConfig := range uc.chainConfigs {
		node, err := chainConfig.GetDefaultNode()
		if err != nil {
			return err
		}
		currentHeight, err := node.QueryCurrentHeight()
		if err != nil {
			return err
		}
		chainConfig.UpgradePropHeight = currentHeight + int64(chainConfig.VotingPeriod) + int64(config.PropSubmitBlocks) + int64(config.PropBufferBlocks)
		propNumber := node.SubmitUpgradeProposal(uc.upgradeVersion, chainConfig.UpgradePropHeight, sdk.NewCoin(initialization.BzeDenom, sdkmath.NewInt(config.InitialMinDeposit)))

		node.DepositProposal(propNumber, false)

		var wg sync.WaitGroup

		for _, node := range chainConfig.NodeConfigs {
			wg.Add(1)
			go func(nodeConfig *chain.NodeConfig) {
				defer wg.Done()
				nodeConfig.VoteYesProposal(initialization.ValidatorWalletName, propNumber)
			}(node)
		}

		wg.Wait()
	}

	// wait till all chains halt at upgrade height
	for _, chainConfig := range uc.chainConfigs {
		uc.t.Logf("waiting to reach upgrade height on chain %s", chainConfig.Id)
		chainConfig.WaitUntilHeight(chainConfig.UpgradePropHeight)
		uc.t.Logf("upgrade height reached on chain %s", chainConfig.Id)
	}

	// remove all containers so we can upgrade them to the new version
	for _, chainConfig := range uc.chainConfigs {
		for _, validatorConfig := range chainConfig.NodeConfigs {
			err := uc.containerManager.RemoveNodeResource(validatorConfig.Name)
			if err != nil {
				return err
			}
		}
	}

	// remove all containers so we can upgrade them to the new version
	for _, chainConfig := range uc.chainConfigs {
		if err := uc.upgradeContainers(chainConfig, chainConfig.UpgradePropHeight); err != nil {
			return err
		}
	}
	return nil
}

func (uc *UpgradeConfigurer) runForkUpgrade() {
	for _, chainConfig := range uc.chainConfigs {
		uc.t.Logf("waiting to reach fork height on chain %s", chainConfig.Id)
		chainConfig.WaitUntilHeight(uc.forkHeight)
		uc.t.Logf("fork height reached on chain %s", chainConfig.Id)
	}
}

func (uc *UpgradeConfigurer) upgradeContainers(chainConfig *chain.Config, propHeight int64) error {
	// upgrade containers to the locally compiled daemon
	uc.t.Logf("starting upgrade for chain-id: %s...", chainConfig.Id)
	uc.containerManager.BzeRepository = containers.CurrentBranchBzeRepository
	uc.containerManager.BzeTag = containers.CurrentBranchBzeTag

	for _, node := range chainConfig.NodeConfigs {
		if err := node.Run(); err != nil {
			return err
		}
	}

	uc.t.Logf("waiting to upgrade containers on chain %s", chainConfig.Id)
	chainConfig.WaitUntilHeight(propHeight)
	uc.t.Logf("upgrade successful on chain %s", chainConfig.Id)
	return nil
}
