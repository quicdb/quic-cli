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

var configInstanceCmd = &cobra.Command{
	Use:   "instance <instance-id>",
	Short: "Set the selected instance",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		instanceID := args[0]

		if err := userconfig.SetSelectedInstance(instanceID); err != nil {
			fmt.Printf("Failed to set selected instance: %v\n", err)
			return
		}

		fmt.Printf("Selected instance set to: %s\n", instanceID)
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

		if config.SelectedInstance == "" {
			fmt.Println("No instance selected")
		} else {
			fmt.Printf("Selected instance: %s\n", config.SelectedInstance)
		}
	},
}

func init() {
	configCmd.AddCommand(configInstanceCmd)
	configCmd.AddCommand(configShowCmd)
}
