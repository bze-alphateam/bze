package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/bze-alphateam/bze/x/tradebin/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group tradebin queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdListMarket())
	cmd.AddCommand(CmdShowMarket())
	cmd.AddCommand(CmdAssetMarkets())
	cmd.AddCommand(CmdUserOrders())
	cmd.AddCommand(CmdUserMarketOrders())
	cmd.AddCommand(CmdMarketAggregatedOrders())
	cmd.AddCommand(CmdMarketHistory())

	// this line is used by starport scaffolding # 1

	return cmd
}
