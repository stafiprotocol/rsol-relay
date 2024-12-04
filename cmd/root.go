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
		mintManagerCmd(),
		stakeManagerCmd(),
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

func stakeManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-manager",
		Short: "stake-manager settings",
	}

	cmd.AddCommand(
		stakeManagerInitCmd(),
		stakeManagerSetRateLimitCmd(),
		stakeManagerSetUnbondingDurationCmd(),
		stakeManagerSetUnstakeFeeCommissionCmd(),
		upgradeStakeManagerCmd(),
		stakeManagerAddValidator(),
		stakeManagerRemoveValidator(),
		stakeManagerTransferAdminCmd(),
		stakeManagerTransferFeeRecipientCmd(),
	)
	return cmd
}

func mintManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint-manager",
		Short: "mint-manager settings",
	}

	cmd.AddCommand(
		mintManagerInitCmd(),
		mintManagerSetMintAuth(),
		mintManagerTransferAdminCmd(),
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
