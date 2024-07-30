package main

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "fgd",
		Short:         "fgd - finality gadget server",
		SilenceErrors: false,
	}
	rootCmd.PersistentFlags().String("cfg", "config.toml", "config file path")

	return rootCmd
}
func main() {
	cmd := NewRootCmd()
	cmd.AddCommand(CommandStart())
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error to run fgd CLI: %v\n", err)
	}
}

// RunEWithClientCtx runs cmd with client context and returns an error.
func RunEWithClientCtx(
	fRunWithCtx func(ctx client.Context, cmd *cobra.Command, args []string) error,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		clientCtx, err := client.GetClientQueryContext(cmd)
		if err != nil {
			return err
		}

		return fRunWithCtx(clientCtx, cmd, args)
	}
}
