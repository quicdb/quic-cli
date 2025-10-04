package cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/quicdb/quic-cli/internal/api"
	"github.com/quicdb/quic-cli/internal/userconfig"
)

// ResolveCluster resolves which cluster ID to use based on:
// 1. Explicit flag value (returns immediately if provided)
// 2. Single cluster auto-selection
// 3. Config file selection (if valid)
// 4. Returns error with cluster list for user selection
func ResolveCluster(ctx context.Context, flagClusterID string, client *api.Client) (string, error) {
	// If cluster ID provided via flag, use it directly
	if flagClusterID != "" {
		return flagClusterID, nil
	}

	// Fetch all clusters
	clusters, err := client.ListClusters(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch clusters: %w", err)
	}

	if len(clusters) == 0 {
		return "", fmt.Errorf("no clusters found. Please create a cluster in the dashboard first")
	}

	// If only one cluster, auto-select it
	if len(clusters) == 1 {
		return clusters[0].ID, nil
	}

	// Multiple clusters - check config for selected cluster
	selectedCluster, err := userconfig.GetSelectedCluster()
	if err == nil && selectedCluster != "" {
		// Validate that selected cluster exists in the list
		if isValidCluster(selectedCluster, clusters) {
			return selectedCluster, nil
		}
	}

	// No valid selected cluster - return error with formatted table
	table := FormatClusterTable(clusters)
	return "", fmt.Errorf("please select a cluster:\n\n" +
		"  Use --cluster flag:         --cluster=<cluster-id>\n" +
		"  Or set default cluster:     quic config cluster <cluster-id>\n\n" +
		"Available clusters:\n\n%s", table)
}

// isValidCluster checks if the given cluster ID exists in the list
func isValidCluster(clusterID string, clusters []api.Cluster) bool {
	for _, c := range clusters {
		if c.ID == clusterID {
			return true
		}
	}
	return false
}

// FormatClusterTable formats a list of clusters as a table string
func FormatClusterTable(clusters []api.Cluster) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("  %-36s %-25s %-15s %-20s\n",
		"Cluster ID", "Name", "Region", "Database"))
	b.WriteString(fmt.Sprintf("  %-36s %-25s %-15s %-20s\n",
		"------------------------------------",
		"-------------------------",
		"---------------",
		"--------------------"))

	// Rows
	for _, cluster := range clusters {
		name := cluster.Name
		if name == "" {
			name = "-"
		}
		database := cluster.SelectedDatabase
		if database == "" {
			database = "-"
		}
		b.WriteString(fmt.Sprintf("  %-36s %-25s %-15s %-20s\n",
			cluster.ID, name, cluster.Region, database))
	}

	return b.String()
}
