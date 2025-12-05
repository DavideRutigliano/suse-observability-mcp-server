package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListMonitorsParams struct {
	ComponentID int64 `json:"component_id" jsonschema:"required,The ID of the component to list monitors for"`
}

// ListMonitors lists monitors for a specific component using the Component API
func (t tool) ListMonitors(ctx context.Context, request *mcp.CallToolRequest, params ListMonitorsParams) (*mcp.CallToolResult, any, error) {
	// Get component with synced check states
	res, err := t.client.GetComponent(ctx, params.ComponentID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get component: %w", err)
	}

	// Check if component has synced check states
	if len(res.Node.SyncedCheckStates) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("No monitors found for component '%s' (ID: %d)", res.Node.Name, params.ComponentID),
				},
			},
		}, nil, nil
	}

	// Build output table
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d monitor(s) for component '%s' (ID: %d):\n\n", len(res.Node.SyncedCheckStates), res.Node.Name, params.ComponentID))
	sb.WriteString("| Monitor Name | Health | Query | Remediation Hint |\n")
	sb.WriteString("|---|---|---|---|\n")

	for _, checkStateData := range res.Node.SyncedCheckStates {
		// Extract monitor name from check state data
		name := ""
		if nameField, ok := checkStateData["name"].(string); ok {
			name = nameField
		}

		// Extract health
		health := ""
		if healthField, ok := checkStateData["health"].(string); ok {
			health = healthField
		}

		// Extract data.displayTimeSeries for queries
		query := "-"
		hint := "-"
		if dataField, ok := checkStateData["data"].(map[string]interface{}); ok {
			// Extract remediation hint
			if remediationHint, ok := dataField["remediationHint"].(string); ok {
				hint = remediationHint
				if len(hint) > 100 {
					hint = hint[:97] + "..."
				}
			}

			// Extract query from displayTimeSeries
			if displayTimeSeries, ok := dataField["displayTimeSeries"].([]interface{}); ok && len(displayTimeSeries) > 0 {
				if series, ok := displayTimeSeries[0].(map[string]interface{}); ok {
					if queries, ok := series["queries"].([]interface{}); ok && len(queries) > 0 {
						if queryData, ok := queries[0].(map[string]interface{}); ok {
							if q, ok := queryData["query"].(string); ok {
								query = fmt.Sprintf("`%s`", q)
								if len(query) > 80 {
									query = query[:77] + "...`"
								}
							}
						}
					}
				}
			}
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", name, health, query, hint))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: sb.String(),
			},
		},
	}, nil, nil
}
