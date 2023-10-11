package initialization

import (
	sdkmath "cosmossdk.io/math"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	tmjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	staketypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/gogo/protobuf/proto"

	"github.com/bze-alphateam/bze/tests/e2e/util"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

// NodeConfig is a confiuration for the node supplied from the test runner
// to initialization scripts. It should be backwards compatible with earlier
// versions. If this struct is updated, the change must be backported to earlier
// branches that might be used for upgrade testing.
type NodeConfig struct {
	Name               string // name of the config that will also be assigned to Docke container.
	Pruning            string // default, nothing, everything, or custom
	PruningKeepRecent  string // keep all of the last N states (only used with custom pruning)
	PruningInterval    string // delete old states from every Nth block (only used with custom pruning)
	SnapshotInterval   uint64 // statesync snapshot every Nth block (0 to disable)
	SnapshotKeepRecent uint32 // number of recent snapshots to keep and serve (0 to keep all)
	IsValidator        bool   // flag indicating whether a node should be a validator
}

const (
	// common
	BzeDenom            = "ubze"
	IonDenom            = "uion"
	StakeDenom          = "stake"
	AtomDenom           = "uatom"
	DaiDenom            = "ibc/0CD3A0285E1341859B5E86B6AB7682F023D03E97607CCC1DC95706411D866DF7"
	BzeIBCDenom         = "ibc/ED07A3391A112B175915CD8FAF43A2DA8E4790EDE12566649D0C2F97716B8518"
	StakeIBCDenom       = "ibc/C053D637CCA2A2BA030E2C5EE1B28A16F71CCB0E45E8BE52766DC1B241B7787"
	E2EFeeToken         = "ubze"
	UstIBCDenom         = "ibc/BE1BB42D4BE3C30D50B68D7C41DB4DFCE9678E8EF8C539F6E6A9345048894FCC"
	LuncIBCDenom        = "ibc/0EF15DF2F02480ADE0BB6E85D9EBB5DAEA2836D3860E9F97F9AADE4F57A31AA0"
	UsdcAxlDenom        = "ibc/D189335C6E4A68B513C10AB227BF1C1D38C746766278BA3EEB4FB14124F1D858"
	MinGasPrice         = "0.000"
	IbcSendAmount       = 3300000000
	ValidatorWalletName = "val"
	// chainA
	ChainAID        = "bze-test-a"
	BzeBalanceA     = 20000000000000
	UsdcAxlBalanceA = 20000000000000
	IonBalanceA     = 100000000000
	StakeBalanceA   = 110000000000
	StakeAmountA    = 100000000000
	UstBalanceA     = 500000000000000
	LuncBalanceA    = 500000000000000
	DaiBalanceA     = "100000000000000000000000"
	// chainB
	ChainBID          = "bze-test-b"
	BzeBalanceB       = 500000000000
	UsdcAxlBalanceB   = 500000000000
	IonBalanceB       = 100000000000
	StakeBalanceB     = 440000000000
	StakeAmountB      = 400000000000
	GenesisFeeBalance = 100000000000
	WalletFeeBalance  = 100000000

	EpochDayDuration      = time.Second * 60
	EpochWeekDuration     = time.Second * 120
	TWAPPruningKeepPeriod = EpochDayDuration / 4

	DaiBzePoolId = 674
)

var (
	StakeAmountIntA  = sdkmath.NewInt(StakeAmountA)
	StakeAmountCoinA = sdk.NewCoin(BzeDenom, StakeAmountIntA)
	StakeAmountIntB  = sdkmath.NewInt(StakeAmountB)
	StakeAmountCoinB = sdk.NewCoin(BzeDenom, StakeAmountIntB)

	DaiBzePoolBalances = fmt.Sprintf("%s%s", DaiBalanceA, DaiDenom)

	InitBalanceStrA = fmt.Sprintf("%d%s,%d%s,%d%s,%d%s,%d%s,%d%s", BzeBalanceA, BzeDenom, StakeBalanceA, StakeDenom, IonBalanceA, IonDenom, UstBalanceA, UstIBCDenom, LuncBalanceA, LuncIBCDenom, UsdcAxlBalanceA, UsdcAxlDenom)
	InitBalanceStrB = fmt.Sprintf("%d%s,%d%s,%d%s,%d%s", BzeBalanceB, BzeDenom, StakeBalanceB, StakeDenom, IonBalanceB, IonDenom, UsdcAxlBalanceA, UsdcAxlDenom)
	BzeToken        = sdk.NewInt64Coin(BzeDenom, IbcSendAmount)   // 3,300uosmo
	StakeToken      = sdk.NewInt64Coin(StakeDenom, IbcSendAmount) // 3,300ustake
	tenBze          = sdk.Coins{sdk.NewInt64Coin(BzeDenom, 10_000_000)}
	fiftyBze        = sdk.Coins{sdk.NewInt64Coin(BzeDenom, 50_000_000)}
	WalletFeeTokens = sdk.NewCoin(E2EFeeToken, sdkmath.NewInt(WalletFeeBalance))
)

func addAccount(path, moniker, amountStr string, accAddr sdk.AccAddress, forkHeight int) error {
	serverCtx := server.NewDefaultContext()
	config := serverCtx.Config

	config.SetRoot(path)
	config.Moniker = moniker

	coins, err := sdk.ParseCoinsNormalized(amountStr)
	if err != nil {
		return fmt.Errorf("failed to parse coins: %w", err)
	}
	coins = coins.Add(sdk.NewCoin(E2EFeeToken, sdkmath.NewInt(GenesisFeeBalance)))

	balances := banktypes.Balance{Address: accAddr.String(), Coins: coins.Sort()}
	genAccount := authtypes.NewBaseAccount(accAddr, nil, 0, 0)

	// TODO: Make the SDK make it far cleaner to add an account to GenesisState
	genFile := config.GenesisFile()
	appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
	if err != nil {
		return fmt.Errorf("failed to unmarshal genesis state: %w", err)
	}

	genDoc.InitialHeight = int64(forkHeight)

	authGenState := authtypes.GetGenesisStateFromAppState(util.Cdc, appState)

	accs, err := authtypes.UnpackAccounts(authGenState.Accounts)
	if err != nil {
		return fmt.Errorf("failed to get accounts from any: %w", err)
	}

	if accs.Contains(accAddr) {
		return fmt.Errorf("failed to add account to genesis state; account already exists: %s", accAddr)
	}

	// Add the new account to the set of genesis accounts and sanitize the
	// accounts afterwards.
	accs = append(accs, genAccount)
	accs = authtypes.SanitizeGenesisAccounts(accs)

	genAccs, err := authtypes.PackAccounts(accs)
	if err != nil {
		return fmt.Errorf("failed to convert accounts into any's: %w", err)
	}

	authGenState.Accounts = genAccs

	authGenStateBz, err := util.Cdc.MarshalJSON(&authGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal auth genesis state: %w", err)
	}

	appState[authtypes.ModuleName] = authGenStateBz

	bankGenState := banktypes.GetGenesisStateFromAppState(util.Cdc, appState)
	bankGenState.Balances = append(bankGenState.Balances, balances)
	bankGenState.Balances = banktypes.SanitizeGenesisBalances(bankGenState.Balances)

	bankGenStateBz, err := util.Cdc.MarshalJSON(bankGenState)
	if err != nil {
		return fmt.Errorf("failed to marshal bank genesis state: %w", err)
	}

	appState[banktypes.ModuleName] = bankGenStateBz

	appStateJSON, err := json.Marshal(appState)
	if err != nil {
		return fmt.Errorf("failed to marshal application genesis state: %w", err)
	}

	genDoc.AppState = appStateJSON
	return genutil.ExportGenesisFile(genDoc, genFile)
}

func updateModuleGenesis[V proto.Message](appGenState map[string]json.RawMessage, moduleName string, protoVal V, updateGenesis func(V)) error {
	if err := util.Cdc.UnmarshalJSON(appGenState[moduleName], protoVal); err != nil {
		return err
	}
	updateGenesis(protoVal)
	newGenState := protoVal

	bz, err := util.Cdc.MarshalJSON(newGenState)
	if err != nil {
		return err
	}
	appGenState[moduleName] = bz
	return nil
}

func initGenesis(chain *internalChain, votingPeriod time.Duration, forkHeight int) error {
	// initialize a genesis file
	configDir := chain.nodes[0].configDir()
	for _, val := range chain.nodes {
		if chain.chainMeta.Id == ChainAID {
			if err := addAccount(configDir, "", InitBalanceStrA+","+DaiBzePoolBalances, val.keyInfo.Address, forkHeight); err != nil {
				return err
			}
		} else if chain.chainMeta.Id == ChainBID {
			if err := addAccount(configDir, "", InitBalanceStrB+","+DaiBzePoolBalances, val.keyInfo.Address, forkHeight); err != nil {
				return err
			}
		}
	}

	// copy the genesis file to the remaining validators
	for _, val := range chain.nodes[1:] {
		_, err := util.CopyFile(
			filepath.Join(configDir, "config", "genesis.json"),
			filepath.Join(val.configDir(), "config", "genesis.json"),
		)
		if err != nil {
			return err
		}
	}

	serverCtx := server.NewDefaultContext()
	config := serverCtx.Config

	config.SetRoot(chain.nodes[0].configDir())
	config.Moniker = chain.nodes[0].moniker

	genFilePath := config.GenesisFile()
	appGenState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFilePath)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, staketypes.ModuleName, &staketypes.GenesisState{}, updateStakeGenesis)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, minttypes.ModuleName, &minttypes.GenesisState{}, updateMintGenesis)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, banktypes.ModuleName, &banktypes.GenesisState{}, updateBankGenesis(appGenState))
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, crisistypes.ModuleName, &crisistypes.GenesisState{}, updateCrisisGenesis)
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, govtypes.ModuleName, &govtypesv1.GenesisState{}, updateGovGenesis(votingPeriod))
	if err != nil {
		return err
	}

	err = updateModuleGenesis(appGenState, genutiltypes.ModuleName, &genutiltypes.GenesisState{}, updateGenUtilGenesis(chain))
	if err != nil {
		return err
	}

	bz, err := json.MarshalIndent(appGenState, "", "  ")
	if err != nil {
		return err
	}

	genDoc.AppState = bz

	genesisJson, err := tmjson.MarshalIndent(genDoc, "", "  ")
	if err != nil {
		return err
	}

	// write the updated genesis file to each validator
	for _, val := range chain.nodes {
		if err := util.WriteFile(filepath.Join(val.configDir(), "config", "genesis.json"), genesisJson); err != nil {
			return err
		}
	}
	return nil
}

func updateBankGenesis(appGenState map[string]json.RawMessage) func(s *banktypes.GenesisState) {
	return func(bankGenState *banktypes.GenesisState) {
		denomsToRegister := []string{StakeDenom, IonDenom, BzeDenom, AtomDenom, LuncIBCDenom, UstIBCDenom, DaiDenom}
		for _, denom := range denomsToRegister {
			setDenomMetadata(bankGenState, denom)
		}
	}
}

func updateStakeGenesis(stakeGenState *staketypes.GenesisState) {
	stakeGenState.Params = staketypes.Params{
		BondDenom:         BzeDenom,
		MaxValidators:     100,
		MaxEntries:        7,
		HistoricalEntries: 10000,
		UnbondingTime:     240000000000,
		MinCommissionRate: sdkmath.LegacyZeroDec(),
	}
}

func updateMintGenesis(mintGenState *minttypes.GenesisState) {
	mintGenState.Params.MintDenom = BzeDenom
}

func updateCrisisGenesis(crisisGenState *crisistypes.GenesisState) {
	crisisGenState.ConstantFee.Denom = BzeDenom
}

func updateGovGenesis(votingPeriod time.Duration) func(*govtypesv1.GenesisState) {
	return func(govGenState *govtypesv1.GenesisState) {
		govGenState.VotingParams.VotingPeriod = votingPeriod
		govGenState.DepositParams.MinDeposit = tenBze
	}
}

func updateGenUtilGenesis(c *internalChain) func(*genutiltypes.GenesisState) {
	return func(genUtilGenState *genutiltypes.GenesisState) {
		// generate genesis txs
		genTxs := make([]json.RawMessage, 0, len(c.nodes))
		for _, node := range c.nodes {
			if !node.isValidator {
				continue
			}

			stakeAmountCoin := StakeAmountCoinA
			if c.chainMeta.Id != ChainAID {
				stakeAmountCoin = StakeAmountCoinB
			}
			createValmsg, err := node.buildCreateValidatorMsg(stakeAmountCoin)
			if err != nil {
				panic("genutil genesis setup failed: " + err.Error())
			}

			signedTx, err := node.signMsg(createValmsg)
			if err != nil {
				panic("genutil genesis setup failed: " + err.Error())
			}

			txRaw, err := util.Cdc.MarshalJSON(signedTx)
			if err != nil {
				panic("genutil genesis setup failed: " + err.Error())
			}
			genTxs = append(genTxs, txRaw)
		}
		genUtilGenState.GenTxs = genTxs
	}
}

func setDenomMetadata(genState *banktypes.GenesisState, denom string) {
	genState.DenomMetadata = append(genState.DenomMetadata, banktypes.Metadata{
		Description: fmt.Sprintf("Registered denom %s for e2e testing", denom),
		Display:     denom,
		Base:        denom,
		Symbol:      denom,
		Name:        denom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denom,
				Exponent: 0,
			},
		},
	})
}
