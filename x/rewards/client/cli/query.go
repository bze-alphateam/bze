package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/rewards/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group rewards queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdListStakingReward())
	cmd.AddCommand(CmdShowStakingReward())
	cmd.AddCommand(CmdListTradingReward())
	cmd.AddCommand(CmdShowTradingReward())
	cmd.AddCommand(CmdListStakingRewardParticipant())
	cmd.AddCommand(CmdShowStakingRewardParticipant())
	cmd.AddCommand(CmdGetTradingRewardLeaderboard())

	cmd.AddCommand(CmdGetMarketIdTradingRewardIdHandler())

	cmd.AddCommand(CmdAllPendingUnlockParticipant())

	// this line is used by starport scaffolding # 1

	return cmd
}
