package tools

import (
	"context"
	"fmt"
	"strings"

	"suse-observability-mcp/client/suseobservability"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetComponentsParams struct {
	// Raw STQL query (takes precedence if provided)
	Query string `json:"query,omitempty" jsonschema:"Raw STQL query for advanced filtering. If provided, other filters are ignored."`

	// Simple filters (used if query is not provided)
	NamePattern string `json:"name_pattern,omitempty" jsonschema:"Component name with wildcard support (e.g., 'checkout*', 'redis*')"`
	Type        string `json:"type,omitempty" jsonschema:"Component type filter (e.g., 'pod', 'service', 'deployment')"`
	Layer       string `json:"layer,omitempty" jsonschema:"Layer filter (e.g., 'Containers', 'Services')"`
	Domain      string `json:"domain,omitempty" jsonschema:"Domain filter (e.g., 'cluster.example.com')"`
	HealthState string `json:"healthstate,omitempty" jsonschema:"Health state filter (e.g., 'CRITICAL', 'DEVIATING', 'CLEAR')"`

	// withNeighborsOf parameters (only used with simple filters, not with raw query)
	WithNeighbors          bool   `json:"with_neighbors,omitempty" jsonschema:"Include connected components using withNeighborsOf function"`
	WithNeighborsLevels    string `json:"with_neighbors_levels,omitempty" jsonschema:"Number of levels (1-14) or 'all' for withNeighborsOf,default=1"`
	WithNeighborsDirection string `json:"with_neighbors_direction,omitempty" jsonschema:"Direction: 'up', 'down', or 'both' for withNeighborsOf,default=both"`
}

type Component struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Identifiers []string       `json:"identifiers,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	State       map[string]any `json:"state,omitempty"`
	Relations   []int64        `json:"relations,omitempty"` // Outgoing relation IDs
}

// GetComponents searches for topology components using STQL filters
func (t tool) GetComponents(ctx context.Context, request *mcp.CallToolRequest, params GetComponentsParams) (*mcp.CallToolResult, any, error) {
	var query string

	// If raw query is provided, use it directly
	if params.Query != "" {
		query = params.Query
	} else {
		// Build STQL query from parameters
		var queryParts []string

		// Add name filter with wildcard support
		if params.NamePattern != "" {
			queryParts = append(queryParts, fmt.Sprintf("name = \"%s\"", params.NamePattern))
		}

		// Add type filter
		if params.Type != "" {
			queryParts = append(queryParts, fmt.Sprintf("type = \"%s\"", params.Type))
		}

		// Add layer filter
		if params.Layer != "" {
			queryParts = append(queryParts, fmt.Sprintf("layer = \"%s\"", params.Layer))
		}

		// Add domain filter
		if params.Domain != "" {
			queryParts = append(queryParts, fmt.Sprintf("domain = \"%s\"", params.Domain))
		}

		// Add healthstate filter
		if params.HealthState != "" {
			queryParts = append(queryParts, fmt.Sprintf("healthstate = \"%s\"", params.HealthState))
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
	}

	if query == "" {
		return nil, nil, fmt.Errorf("either 'query' or at least one filter (name_pattern, type, layer, domain, healthstate) must be provided")
	}

	// Execute topology query
	components, err := t.client.SnapShotTopologyQuery(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query topology (STQL: %s): %w", query, err)
	}

	simplified := simplifyViewComponents(components)
	table := formatComponentsTable(simplified, params, query)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: table,
			},
		},
	}, nil, nil
}

func simplifyViewComponents(components []suseobservability.ViewComponent) []Component {
	var simplified []Component
	for _, c := range components {
		simplified = append(simplified, Component{
			ID:          c.ID,
			Name:        c.Name,
			Type:        c.InternalType,
			Identifiers: c.Identifiers,
			Tags:        c.Tags,
			Relations:   c.OutgoingRelations,
		})
	}
	return simplified
}

func formatComponentsTable(components []Component, params GetComponentsParams, query string) string {
	if len(components) == 0 {
		return fmt.Sprintf("No components found for query: %s", query)
	}

	var sb strings.Builder

	// Summary
	sb.WriteString(fmt.Sprintf("Found %d component(s)", len(components)))
	if params.Query != "" {
		sb.WriteString(fmt.Sprintf(" for query: %s", params.Query))
	} else {
		filters := []string{}
		if params.NamePattern != "" {
			filters = append(filters, fmt.Sprintf("name: %s", params.NamePattern))
		}
		if params.Type != "" {
			filters = append(filters, fmt.Sprintf("type: %s", params.Type))
		}
		if params.Layer != "" {
			filters = append(filters, fmt.Sprintf("layer: %s", params.Layer))
		}
		if params.Domain != "" {
			filters = append(filters, fmt.Sprintf("domain: %s", params.Domain))
		}
		if params.HealthState != "" {
			filters = append(filters, fmt.Sprintf("healthstate: %s", params.HealthState))
		}
		if len(filters) > 0 {
			sb.WriteString(" (" + strings.Join(filters, ", ") + ")")
		}
	}
	sb.WriteString(":\n\n")

	// Header
	sb.WriteString("| Component Name | Type | ID | Identifiers |\n")
	sb.WriteString("|---|---|---|---|\n")

	// Data rows
	for _, c := range components {
		identifiersStr := "-"
		if len(c.Identifiers) > 0 {
			// Show first 2 identifiers to keep table readable
			if len(c.Identifiers) > 2 {
				identifiersStr = strings.Join(c.Identifiers[:2], ", ") + "..."
			} else {
				identifiersStr = strings.Join(c.Identifiers, ", ")
			}
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s |\n", c.Name, c.Type, c.ID, identifiersStr))
	}

	return sb.String()
}
