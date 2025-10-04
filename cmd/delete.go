package cmd

import (
	"context"
	"fmt"

	"github.com/quicdb/quic-cli/internal/api"
	"github.com/quicdb/quic-cli/internal/auth"
	"github.com/quicdb/quic-cli/internal/cluster"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <branch-name>",
	Short: "Delete a database branch",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		branchName := args[0]

		// Check if user is authenticated
		_, err := auth.LoadToken(auth.AccessToken)
		if err != nil {
			fmt.Println("You are not logged in. Please run 'quic login' first.")
			return
		}

		// Get cluster ID from flag
		flagClusterID, err := cmd.Flags().GetString("cluster")
		if err != nil {
			fmt.Printf("Error getting cluster flag: %v\n", err)
			return
		}

		// Resolve which cluster to use
		client := api.NewClient()
		ctx := context.Background()
		clusterID, err := cluster.ResolveCluster(ctx, flagClusterID, client)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Delete the branch
		err = client.DeleteBranch(ctx, clusterID, branchName)
		if err != nil {
			fmt.Printf("Failed to delete branch: %v\n", err)
			return
		}

		fmt.Printf("Branch '%s' scheduled for deletion\n", branchName)
	},
}

func init() {
	deleteCmd.Flags().StringP("cluster", "c", "", "Cluster ID to delete the branch from")
}
