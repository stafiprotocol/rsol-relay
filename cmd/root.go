package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var (
	appName = "rsol-relay"
)

const (
	flagLogLevel     = "log_level"
	flagConfigPath   = "config"
	flagKeystorePath = "keystore_path"

	defaultKeystorePath = "./keys/solana_keys.json"
	defaultConfigPath   = "./config.toml"
)

// NewRootCmd returns the root command.
func NewRootCmd() *cobra.Command {
	// RootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   appName,
		Short: "rsol-relay",
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, segments []string) error {
		return nil
	}

	rootCmd.AddCommand(
		keysCmd(),
		adminCmd(),
		startCmd(),
		versionCmd(),
	)

	return rootCmd
}

func keysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage keystore",
	}

	cmd.AddCommand(
		vaultAddCmd(),
		vaultCreateCmd(),
		vaultExportCmd(),
		vaultListCmd(),
	)
	return cmd
}

func adminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin operate",
	}

	cmd.AddCommand(

		rsolInitCmd(),
		minterInitCmd(),
		rsolSetRateLimitCmd(),
		rsolSetUnbondingDurationCmd(),
		rsolSetUnstakeFeeCommissionCmd(),
		upgradeStakeManagerCmd(),
	)
	return cmd
}

func Execute() {

	rootCmd := NewRootCmd()
	rootCmd.SilenceUsage = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	ctx := context.Background()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
