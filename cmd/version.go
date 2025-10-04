package cmd

import (
	"fmt"

	"github.com/quicdb/quic-cli/releases"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(releases.Version)
	},
}
