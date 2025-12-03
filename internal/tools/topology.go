package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type QueryTopologyParams struct {
	Query string `json:"query" jsonschema:"The topology query to execute"`
	Time  string `json:"time,omitempty" jsonschema:"Optional time to execute the query at"`
}

// QueryTopology queries the StackState topology
func (t tool) QueryTopology(ctx context.Context, request *mcp.CallToolRequest, params QueryTopologyParams) (*mcp.CallToolResult, any, error) {
	res, err := t.client.TopologyQuery(ctx, params.Query, params.Time, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query topology: %w", err)
	}

	jsonRes, err := json.Marshal(res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal topology result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonRes),
			},
		},
	}, nil, nil
}
