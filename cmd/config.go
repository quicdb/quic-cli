package cmd

import (
	"fmt"

	"github.com/quicdb/quic-cli/internal/userconfig"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configClusterCmd = &cobra.Command{
	Use:   "cluster <cluster-id>",
	Short: "Set the selected cluster",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		clusterID := args[0]

		if err := userconfig.SetSelectedCluster(clusterID); err != nil {
			fmt.Printf("Failed to set selected cluster: %v\n", err)
			return
		}

		fmt.Printf("Selected cluster set to: %s\n", clusterID)
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := userconfig.Load()
		if err != nil {
			fmt.Printf("Failed to load configuration: %v\n", err)
			return
		}

		if config.SelectedCluster == "" {
			fmt.Println("No cluster selected")
		} else {
			fmt.Printf("Selected cluster: %s\n", config.SelectedCluster)
		}
	},
}

func init() {
	configCmd.AddCommand(configClusterCmd)
	configCmd.AddCommand(configShowCmd)
}
