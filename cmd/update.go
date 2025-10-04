package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/quicdb/quic-cli/releases"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		if isHomebrewInstall() {
			fmt.Println("Detected Homebrew installation.")
			fmt.Println("Please use: brew update && brew upgrade quic")
			os.Exit(1)
		}
		if err := selfUpdate(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func selfUpdate() error {
	latest, err := releases.GetLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to check latest version: %v", err)
	}

	if !releases.IsNewerVersion(releases.Version, latest) {
		return fmt.Errorf("already on latest version %s", releases.Version)
	}

	var binaryName string
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		binaryName = "quic-darwin-amd64"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		binaryName = "quic-darwin-arm64"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		binaryName = "quic-linux-amd64"
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		binaryName = "quic-linux-arm64"
	default:
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	downloadURL := fmt.Sprintf("https://github.com/quicdb/quic-cli/releases/latest/download/%s", binaryName)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download update: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download update: HTTP %d", resp.StatusCode)
	}

	// Get current executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	// Create temporary file
	tmpFile := executable + ".tmp"
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}

	// Copy downloaded content
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to write update: %v", err)
	}

	// Replace current executable
	if err := os.Rename(tmpFile, executable); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to replace executable: %v", err)
	}

	fmt.Println("Done")

	return nil
}
