package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/bze-alphateam/bze/x/rewards/types"
)

var (
	DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())
)

const (
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	listSeparator              = ","
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdCreateStakingReward())
	cmd.AddCommand(CmdUpdateStakingReward())
	cmd.AddCommand(CmdCreateTradingReward())
	cmd.AddCommand(CmdJoinStaking())
	cmd.AddCommand(CmdExitStaking())
	cmd.AddCommand(CmdClaimStakingRewards())
	cmd.AddCommand(CmdDistributeStakingRewards())
	// this line is used by starport scaffolding # 1

	return cmd
}
