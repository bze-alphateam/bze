package cli

import (
	"github.com/bze-alphateam/bze/x/cointrunk/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/spf13/cobra"
)

const (
	ActiveFlag = "active"
)

func NewCmdSubmitAcceptedDomainProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accepted-domain [domain-name]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit an accepted domain proposal",
		Long: "Submit an accepted domain proposal along with an initial deposit.\n" +
			"Please specify the domain you want to add (ex: domain.com)\n",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(govcli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription)
			if err != nil {
				return err
			}

			domain := args[0]
			active, err := cmd.Flags().GetBool(ActiveFlag)
			if err != nil {
				return err
			}

			content := types.NewAcceptedDomainProposal(title, description, domain, active)

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().Bool(ActiveFlag, true, "whether or not the proposed domain to be active")

	return cmd
}

func NewCmdSubmitPublisherProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publisher [name] [address]",
		Args:  cobra.ExactArgs(2),
		Short: "Submit a publisher proposal",
		Long: "Submit a publishers proposal along with an initial deposit.\n" +
			"Please specify the public name of the publisher\n" +
			"Please specify the address to become a publisher\n",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(govcli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription)
			if err != nil {
				return err
			}

			name := args[0]
			address := args[1]
			active, err := cmd.Flags().GetBool(ActiveFlag)
			if err != nil {
				return err
			}

			content := types.NewPublisherProposal(title, description, name, address, active)

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().Bool(ActiveFlag, true, "whether or not the proposed publisher to be active")

	return cmd
}

func NewCmdSubmitBurnCoinsProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coins-burn",
		Args:  cobra.ExactArgs(0),
		Short: "Submit coins burning proposal",
		Long:  "Submit coins burning proposal along with an initial deposit.\n",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(govcli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription)
			if err != nil {
				return err
			}

			content := types.NewBurnCoinsProposal(title, description)

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	return cmd
}
