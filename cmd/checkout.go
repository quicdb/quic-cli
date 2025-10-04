package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/quicdb/quic-cli/internal/api"
	"github.com/quicdb/quic-cli/internal/auth"
	"github.com/quicdb/quic-cli/internal/instance"
	"github.com/spf13/cobra"
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout <branch-name>",
	Short: "Create a new database branch",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		branchName := args[0]

		// Check if user is authenticated
		_, err := auth.LoadToken(auth.AccessToken)
		if err != nil {
			fmt.Println("You are not logged in. Please run 'quic login' first.")
			return
		}

		// Get instance ID from flag
		flagInstanceID, err := cmd.Flags().GetString("instance")
		if err != nil {
			fmt.Printf("Error getting instance flag: %v\n", err)
			return
		}

		// Resolve which instance to use
		client := api.NewClient()
		ctx := context.Background()
		instanceID, err := instance.ResolveInstance(ctx, flagInstanceID, client)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Create the branch
		branch, err := client.CreateBranch(ctx, instanceID, branchName, 90*time.Second)
		if err != nil {
			var apiErr *api.APIError
			if e, ok := err.(*api.APIError); ok {
				apiErr = e
			}

			if apiErr != nil && apiErr.StatusCode == 409 {
				// Conflict - likely cluster not ready
				fmt.Printf("Error: Cannot create branch '%s'\n", branchName)
				fmt.Printf("%s\n\n", apiErr.Message)
				fmt.Println("Please wait for the cluster to be ready before creating branches.")
			} else {
				fmt.Printf("Failed to create branch: %v\n", err)
			}
			return
		}

		// Output PostgreSQL connection string
		connectionString := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
			branch.User,
			branch.Password,
			branch.Host,
			branch.Port,
			branch.Database,
		)

		fmt.Println(connectionString)
	},
}

func init() {
	checkoutCmd.Flags().StringP("instance", "i", "", "Instance ID to create the branch from")
}
