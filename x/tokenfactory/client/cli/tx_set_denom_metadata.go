package cli

import (
	"strconv"

	"github.com/bze-alphateam/bze/x/tokenfactory/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

// string description = 1;
// // denom_units represents the list of DenomUnit's for a given coin
// repeated DenomUnit denom_units = 2;
// // base represents the base denom (should be the DenomUnit with exponent = 0).
// string base = 3;
// // display indicates the suggested denom that should be
// // displayed in clients.
// string display = 4;
// // name defines the name of the token (eg: Cosmos Atom)
// //
// // Since: cosmos-sdk 0.43
// string name = 5;
// // symbol is the token symbol usually shown on exchanges (eg: ATOM). This can
// // be the same as the display.
// //
// // Since: cosmos-sdk 0.43
// string symbol = 6;
// // URI to a document (on or off-chain) that contains additional information. Optional.
// //
// // Since: cosmos-sdk 0.46
// string uri = 7 [(gogoproto.customname) = "URI"];
// // URIHash is a sha256 hash of a document pointed by URI. It's used to verify that
// // the document didn't change. Optional.
// //
// // Since: cosmos-sdk 0.46
// string uri_hash = 8 [(gogoproto.customname) = "URIHash"];
func CmdSetDenomMetadata() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-denom-metadata [metadata_json]",
		Short: "Broadcast message SetDenomMetadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argMetadataJsonString := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgSetDenomMetadata(
				clientCtx.GetFromAddress().String(),
				argMetadataJsonString,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
