package tools

import (
	"context"
	"fmt"
	"strings"

	"suse-observability-mcp/client/suseobservability"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetComponentsParams struct {
	// Filters - all support multiple comma-separated values
	Names        string `json:"names,omitempty" jsonschema:"Component names to match (comma-separated for multiple values, e.g., 'checkout-service,redis-master')"`
	Types        string `json:"types,omitempty" jsonschema:"Component types to filter (comma-separated, e.g., 'pod,service,deployment')"`
	HealthStates string `json:"healthstates,omitempty" jsonschema:"Health states to filter (comma-separated, e.g., 'CRITICAL,DEVIATING')"`
	Domains      string `json:"domains,omitempty" jsonschema:"Cluster names to filter (comma-separated, e.g., 'prod-cluster,staging-cluster'). Domain represents the cluster name."`
	Namespace    string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace to filter (e.g., 'default', 'kube-system')"`

	// withNeighborsOf parameters
	WithNeighbors          bool   `json:"with_neighbors,omitempty" jsonschema:"Include connected components using withNeighborsOf function"`
	WithNeighborsLevels    string `json:"with_neighbors_levels,omitempty" jsonschema:"Number of levels (1-14) or 'all' for withNeighborsOf,default=1"`
	WithNeighborsDirection string `json:"with_neighbors_direction,omitempty" jsonschema:"Direction: 'up', 'down', or 'both' for withNeighborsOf,default=both"`
}

type Component struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	State string `json:"state,omitempty"`
}

// GetComponents searches for topology components using STQL filters
func (t tool) GetComponents(ctx context.Context, request *mcp.CallToolRequest, params GetComponentsParams) (*mcp.CallToolResult, any, error) {
	var query string

	// Build STQL query from parameters using IN/NOT IN operators
	var queryParts []string

	// Helper function to parse comma-separated values and build IN clause
	buildInClause := func(fieldName, values string) string {
		if values == "" {
			return ""
		}
		parts := strings.Split(values, ",")
		quoted := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				quoted = append(quoted, fmt.Sprintf("\"%s\"", p))
			}
		}
		if len(quoted) == 0 {
			return ""
		}
		return fmt.Sprintf("%s IN (%s)", fieldName, strings.Join(quoted, ", "))
	}

	// Add names filter
	if clause := buildInClause("name", params.Names); clause != "" {
		queryParts = append(queryParts, clause)
	}

	// Add types filter
	if clause := buildInClause("type", params.Types); clause != "" {
		queryParts = append(queryParts, clause)
	}

	// Add healthstates filter
	if clause := buildInClause("healthstate", params.HealthStates); clause != "" {
		queryParts = append(queryParts, clause)
	}

	// Add domains filter (cluster names)
	if clause := buildInClause("domain", params.Domains); clause != "" {
		queryParts = append(queryParts, clause)
	}

	// Add namespace filter
	if params.Namespace != "" {
		queryParts = append(queryParts, fmt.Sprintf("namespace = \"%s\"", params.Namespace))
	}

	// Combine basic filters with AND
	if len(queryParts) > 0 {
		query = strings.Join(queryParts, " AND ")
	}

	// Add withNeighborsOf if requested
	if params.WithNeighbors {
		if query == "" {
			return nil, nil, fmt.Errorf("with_neighbors requires at least one filter to define the components")
		}

		// Set defaults for levels and direction
		levels := params.WithNeighborsLevels
		if levels == "" {
			levels = "1"
		}
		direction := params.WithNeighborsDirection
		if direction == "" {
			direction = "both"
		}

		// Validate direction
		validDirections := map[string]bool{"up": true, "down": true, "both": true}
		if !validDirections[direction] {
			return nil, nil, fmt.Errorf("invalid with_neighbors_direction '%s'. Must be 'up', 'down', or 'both'", direction)
		}

		// Build withNeighborsOf function
		// According to STQL spec, combine the base filters with OR when using withNeighborsOf
		neighborsQuery := fmt.Sprintf("withNeighborsOf(components = (%s), levels = \"%s\", direction = \"%s\")", query, levels, direction)
		query = fmt.Sprintf("%s OR %s", query, neighborsQuery)
	}

	if query == "" {
		return nil, nil, fmt.Errorf("at least one filter (names, types, healthstates, domains, namespace) must be provided")
	}

	// Execute topology query
	components, err := t.client.SnapShotTopologyQuery(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query topology (STQL: %s): %w", query, err)
	}

	table := formatComponentsTable(components, params, query)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: table,
			},
		},
	}, nil, nil
}

func formatComponentsTable(components []suseobservability.ViewComponent, params GetComponentsParams, query string) string {
	if len(components) == 0 {
		return fmt.Sprintf("No components found for query: %s", query)
	}

	var sb strings.Builder

	// Summary
	sb.WriteString(fmt.Sprintf("Found %d component(s)", len(components)))

	filters := []string{}
	if params.Names != "" {
		filters = append(filters, fmt.Sprintf("names: %s", params.Names))
	}
	if params.Types != "" {
		filters = append(filters, fmt.Sprintf("types: %s", params.Types))
	}
	if params.HealthStates != "" {
		filters = append(filters, fmt.Sprintf("healthstates: %s", params.HealthStates))
	}
	if params.Domains != "" {
		filters = append(filters, fmt.Sprintf("domains: %s", params.Domains))
	}
	if params.Namespace != "" {
		filters = append(filters, fmt.Sprintf("namespace: %s", params.Namespace))
	}
	if len(filters) > 0 {
		sb.WriteString(" (" + strings.Join(filters, ", ") + ")")
	}
	sb.WriteString(":\n\n")

	// Header
	sb.WriteString("| Component Name | ID | State |\n")
	sb.WriteString("|---|---|---|\n")

	// Data rows
	for _, c := range components {
		sb.WriteString(fmt.Sprintf("| %s | %d | %s |\n", c.Name, c.ID, c.State.HealthState))
	}

	return sb.String()
}
