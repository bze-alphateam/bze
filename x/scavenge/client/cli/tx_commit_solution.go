package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/bze-alphateam/bze/x/scavenge/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
)

var _ = strconv.Itoa(0)

func CmdCommitSolution() *cobra.Command {
	cmd := &cobra.Command{
		// pass a solution as the only argument
		Use:   "commit-solution [scavengeIndex] [solution]",
		Short: "Broadcast message commit-solution",
		// set the number of arguments to 1
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			solution := args[1]
			scavengeIndex := args[0]
			// find a hash of the solution
			solutionHash := sha256.Sum256([]byte(solution))
			// convert the solution hash to string
			solutionHashString := hex.EncodeToString(solutionHash[:])
			// convert a scavenger address to string
			var scavenger = clientCtx.GetFromAddress().String()
			// find the hash of solution and scavenger address
			var solutionScavengerHash = sha256.Sum256([]byte(scavengeIndex + solution + scavenger))
			// convert the hash to string
			var solutionScavengerHashString = hex.EncodeToString(solutionScavengerHash[:])
			// create a new message
			msg := types.NewMsgCommitSolution(clientCtx.GetFromAddress().String(), solutionHashString, solutionScavengerHashString)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// broadcast the transaction with the message to the blockchain
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
