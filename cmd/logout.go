package cmd

import (
	"fmt"

	"github.com/quicdb/quic-cli/internal/auth"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from QuicDB",
	Run: func(cmd *cobra.Command, args []string) {
		if err := auth.ClearAllTokens(); err != nil {
			fmt.Printf("Warning: Failed to logout: %v\n", err)
		}
	},
}
