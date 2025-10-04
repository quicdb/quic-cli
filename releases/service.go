package releases

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Version is set at build time via ldflags
var Version = "dev"

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func GetLatestVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/quicdb/quic-cli/releases/latest", nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to check version: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse JSON response to get tag_name
	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("failed to parse release info: %v", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("no release tag found")
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

func IsNewerVersion(current, latest string) bool {
	if current == "dev" {
		return false
	}

	currentParts := parseVersion(strings.TrimPrefix(current, "v"))
	latestParts := parseVersion(strings.TrimPrefix(latest, "v"))

	maxLen := max(len(latestParts), len(currentParts))

	for len(currentParts) < maxLen {
		currentParts = append(currentParts, 0)
	}
	for len(latestParts) < maxLen {
		latestParts = append(latestParts, 0)
	}

	for i := range maxLen {
		if latestParts[i] > currentParts[i] {
			return true
		} else if latestParts[i] < currentParts[i] {
			return false
		}
	}

	return false
}

func parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		// Ignore errors, default to 0
		num := 0
		fmt.Sscanf(p, "%d", &num)
		nums[i] = num
	}
	return nums
}
