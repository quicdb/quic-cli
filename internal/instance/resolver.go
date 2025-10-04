package instance

import (
	"context"
	"fmt"
	"strings"

	"github.com/quicdb/quic-cli/internal/api"
	"github.com/quicdb/quic-cli/internal/userconfig"
)

// ResolveInstance resolves which instance ID to use based on:
// 1. Explicit flag value (returns immediately if provided)
// 2. Single instance auto-selection
// 3. Config file selection (if valid)
// 4. Returns error with instance list for user selection
func ResolveInstance(ctx context.Context, flagInstanceID string, client *api.Client) (string, error) {
	// If instance ID provided via flag, use it directly
	if flagInstanceID != "" {
		return flagInstanceID, nil
	}

	// Fetch all instances
	instances, err := client.ListInstances(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch clusters: %w", err)
	}

	if len(instances) == 0 {
		return "", fmt.Errorf("no clusters found. Please create a cluster in the dashboard first")
	}

	// If only one instance, auto-select it
	if len(instances) == 1 {
		return instances[0].ID, nil
	}

	// Multiple instances - check config for selected instance
	selectedInstance, err := userconfig.GetSelectedInstance()
	if err == nil && selectedInstance != "" {
		// Validate that selected instance exists in the list
		if isValidInstance(selectedInstance, instances) {
			return selectedInstance, nil
		}
	}

	// No valid selected instance - return error with formatted table
	table := FormatInstanceTable(instances)
	return "", fmt.Errorf("please select a cluster:\n\n" +
		"  Use --instance flag:        --instance=<cluster-id>\n" +
		"  Or set default cluster:     quic config instance <cluster-id>\n\n" +
		"Available clusters:\n\n%s", table)
}

// isValidInstance checks if the given instance ID exists in the list
func isValidInstance(instanceID string, instances []api.Instance) bool {
	for _, inst := range instances {
		if inst.ID == instanceID {
			return true
		}
	}
	return false
}

// FormatInstanceTable formats a list of instances as a table string
func FormatInstanceTable(instances []api.Instance) string {
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
	for _, instance := range instances {
		name := instance.Name
		if name == "" {
			name = "-"
		}
		database := instance.SelectedDatabase
		if database == "" {
			database = "-"
		}
		b.WriteString(fmt.Sprintf("  %-36s %-25s %-15s %-20s\n",
			instance.ID, name, instance.Region, database))
	}

	return b.String()
}
