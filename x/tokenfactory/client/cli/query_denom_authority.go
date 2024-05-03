package cli

import (
	"strconv"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdQueryDenomAuthority() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denom-authority [denom]",
		Short: "Query DenomAuthority",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqDenom := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryDenomAuthorityRequest{

				Denom: reqDenom,
			}

			res, err := queryClient.DenomAuthority(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
