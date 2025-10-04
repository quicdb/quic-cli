package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var dashCmd = &cobra.Command{
	Use:   "dash",
	Short: "Open QuicDB dashboard in browser",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		url := "https://dash.quicdb.com"

		var err error
		switch runtime.GOOS {
		case "linux":
			err = exec.Command("xdg-open", url).Start()
		case "windows":
			err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		case "darwin":
			err = exec.Command("open", url).Start()
		default:
			err = fmt.Errorf("unsupported platform")
		}

		if err != nil {
			fmt.Printf("Failed to open browser: %v\n", err)
			fmt.Printf("Please visit: %s\n", url)
			return
		}

		fmt.Printf("Opening %s in your browser...\n", url)
	},
}
