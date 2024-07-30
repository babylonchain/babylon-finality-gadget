package main

import (
	"github.com/babylonchain/babylon-finality-gadget/config"
	"github.com/babylonchain/babylon-finality-gadget/service"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

func CommandStart() *cobra.Command {
	startCmd := &cobra.Command{
		Use:     "start",
		Short:   "start the finality gadget server",
		Example: `fgd start --cfg config.toml`,
		Args:    cobra.NoArgs,
		RunE:    RunEWithClientCtx(start),
	}
	return startCmd
}

func start(ctx client.Context, cmd *cobra.Command, args []string) error {
	cfgPath, err := cmd.Flags().GetString("cfg")
	if err != nil {
		return err
	}
	config, err := config.LoadConfig(cfgPath)
	if err != nil {
		return err
	}

	// Start the finality gadget server
	srv := service.NewService(config)
	return srv.Start()
}
