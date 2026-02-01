package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nicjohnson145/blankpage/internal/cliconfig"
	"github.com/nicjohnson145/blankpage/internal/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := Root().ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blankpage",
		Short: "Interact with blankpage service",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			if err := cliconfig.InitConfig(cmd); err != nil {
				return fmt.Errorf("error initializing config: %w", err)
			}

			// Validate top level flags
			if _, err := logging.ParseLogLevel(viper.GetString(cliconfig.FlagLogLevel)); err != nil {
				return err
			}
			if val := viper.GetString(cliconfig.FlagServerUrl); val == "" {
				return fmt.Errorf("server url must be defined either in config file, env var, or cli arg")
			}

			return nil
		},
	}

	cmd.PersistentFlags().Bool(
		cliconfig.FlagNoInteractive,
		cliconfig.DefaultFlagNoInteractive,
		"Do not prompt for any user input",
	)
	cmd.PersistentFlags().String(
		cliconfig.FlagLogLevel,
		cliconfig.DefaultFlagLogLevel,
		fmt.Sprintf(
			"Logging verbosity. Must be one of [%v]",
			strings.Join(logging.LogLevelNames(), ", "),
		),
	)
	cmd.PersistentFlags().String(
		cliconfig.FlagServerUrl,
		"",
		"Url of blankpage service",
	)

	cmd.AddCommand(
		AddBook(),
	)

	return cmd
}
