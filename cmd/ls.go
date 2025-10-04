package cmd

import (
	"context"
	"fmt"

	"github.com/quicdb/quic-cli/internal/api"
	"github.com/quicdb/quic-cli/internal/auth"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all database branches",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if user is authenticated
		_, err := auth.LoadToken(auth.AccessToken)
		if err != nil {
			fmt.Println("You are not logged in. Please run 'quic login' first.")
			return
		}

		client := api.NewClient()
		ctx := context.Background()

		branches, err := client.ListBranches(ctx)
		if err != nil {
			fmt.Printf("Failed to list branches: %v\n", err)
			return
		}

		if len(branches) == 0 {
			fmt.Println("No branches found. Create one with 'quic checkout <branch-name>'")
			return
		}

		// Print table header
		fmt.Printf("  %-20s %-30s %-30s %-20s\n", "Branch", "Cluster", "Created by", "Created at")
		fmt.Printf("  %-20s %-30s %-30s %-20s\n", "--------------------", "------------------------------", "------------------------------", "--------------------")

		// Print each branch
		for _, branch := range branches {
			fmt.Printf("  %-20s %-30s %-30s %-20s\n", branch.Name, branch.Cluster, branch.CreatedBy, branch.CreatedAt)
		}
	},
}
