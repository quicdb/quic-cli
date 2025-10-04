package cmd

import (
	"fmt"
	"os"

	"github.com/quicdb/quic-cli/releases"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "quic",
	Short: "QuicDB CLI",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		checkForUpdateNotification()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(checkoutCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(dashCmd)
	rootCmd.AddCommand(configCmd)
}

func checkForUpdateNotification() {
	latest, err := releases.GetLatestVersion()
	if err != nil {
		return
	}

	if releases.IsNewerVersion(releases.Version, latest) {
		fmt.Printf("> Newer version available: v%s -> v%s\n", releases.Version, latest)
		fmt.Println("> $ quic update")
	}
}
