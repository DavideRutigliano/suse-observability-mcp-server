package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetMonitorsParams struct {
	State string `json:"state,omitempty" jsonschema:"Filter by state. Allowed values: 'CRITICAL' 'DEVIATING' 'UNKNOWN',default=CRITICAL"`
}

type MonitorData struct {
	Name          string
	Description   string
	Query         string
	CriticalCount int
	WarningCount  int
	ClearCount    int
}

// GetMonitors lists monitors filtered by health state with component details
func (t tool) GetMonitors(ctx context.Context, request *mcp.CallToolRequest, params GetMonitorsParams) (*mcp.CallToolResult, any, error) {
	// Default to CRITICAL if not specified
	state := params.State
	if state == "" {
		state = "CRITICAL"
	}

	// Validate state parameter
	validStates := map[string]bool{
		"CRITICAL":  true,
		"DEVIATING": true,
		"UNKNOWN":   true,
	}
	if !validStates[state] {
		return nil, nil, fmt.Errorf("invalid state '%s'. Allowed values: CRITICAL, DEVIATING, UNKNOWN", state)
	}

	// Get monitors overview
	overview, err := t.client.GetMonitorsOverview(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get monitors overview: %w", err)
	}

	// Collect monitor data
	type MonitorRow struct {
		Name              string
		Description       string
		AffectedCount     int
		AffectedComponent string
	}
	var rows []MonitorRow

	for _, monitorOverview := range overview.Monitors {
		monitor := monitorOverview.Monitor
		metrics := monitorOverview.RuntimeMetrics

		// Check if this monitor has components in the requested state
		var count int
		switch state {
		case "CRITICAL":
			count = metrics.CriticalCount
		case "DEVIATING":
			count = metrics.DeviatingCount
		case "UNKNOWN":
			count = metrics.UnknownCount
		}

		if count == 0 {
			continue
		}

		// Fetch check states to get component details
		checkStates, err := t.client.GetMonitorCheckStates(ctx, fmt.Sprintf("%d", monitor.Id), state, 10, 0)
		if err != nil || len(checkStates.States) == 0 {
			// Fallback: show monitor without component details
			rows = append(rows, MonitorRow{
				Name:              monitor.Name,
				Description:       monitor.Description,
				AffectedCount:     count,
				AffectedComponent: "-",
			})
			continue
		}

		// List affected components (show first few)
		componentsShown := 0
		maxComponents := 5
		for _, checkState := range checkStates.States {
			if componentsShown >= maxComponents {
				break
			}
			componentRef := fmt.Sprintf("ID:%d", checkState.TopologyElementId)
			if checkState.TopologyElementIdType == "identifier" {
				componentRef = fmt.Sprintf("URN:%d", checkState.TopologyElementId)
			}
			componentStr := fmt.Sprintf("%s (%s)", checkState.Name, componentRef)

			rows = append(rows, MonitorRow{
				Name:              monitor.Name,
				Description:       monitor.Description,
				AffectedCount:     count,
				AffectedComponent: componentStr,
			})
			componentsShown++
		}

		// Add "more" row if there are additional components
		if len(checkStates.States) > maxComponents {
			rows = append(rows, MonitorRow{
				Name:              monitor.Name,
				Description:       "-",
				AffectedCount:     count,
				AffectedComponent: fmt.Sprintf("... and %d more", len(checkStates.States)-maxComponents),
			})
		}
	}

	// Build output
	var sb strings.Builder

	if len(rows) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("No monitors in %s state found.", state),
				},
			},
		}, nil, nil
	}

	// Summary
	monitorCount := 0
	seenMonitors := make(map[string]bool)
	for _, row := range rows {
		if !seenMonitors[row.Name] {
			monitorCount++
			seenMonitors[row.Name] = true
		}
	}
	sb.WriteString(fmt.Sprintf("Found %d monitor(s) in %s state:\n\n", monitorCount, state))

	// Header
	sb.WriteString("| Monitor Name | Description | Affected Count | Affected Component |\n")
	sb.WriteString("|---|---|---|---|\n")

	// Data rows
	for _, row := range rows {
		desc := row.Description
		if desc == "" {
			desc = "-"
		}
		// Truncate long descriptions
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s |\n", row.Name, desc, row.AffectedCount, row.AffectedComponent))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: sb.String(),
			},
		},
	}, nil, nil
}
