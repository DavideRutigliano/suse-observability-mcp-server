package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"suse-observability-mcp/client/suseobservability"
)

type QueryTracesParams struct {
	Page        int    `json:"page" jsonschema:"which page from the query result to retrieve"`
	PageSize    int    `json:"pageSize" jsonschema:"the size of the page you want to retrieve"`
	Hours       int    `json:"hours" jsonschema:"How many hours (relative to now) we want to inspect"`
	ServiceName string `json:"serviceName" jsonschema:"The name of the service that you want to inspect the traces for"`
}

func (t tool) QueryTraces(ctx context.Context, request *mcp.CallToolRequest, params QueryTracesParams) (resp *mcp.CallToolResult, a any, err error) {
	now := time.Now()
	result, err := t.client.QueryTraces(ctx, &suseobservability.TraceQueryRequest{
		TraceQuery: suseobservability.TraceQuery{
			SpanFilter: suseobservability.SpanFilter{
				ServiceName: []string{params.ServiceName},
			},
		},
		Start:    now.Add(-time.Duration(params.Hours) * time.Hour),
		End:      now,
		Page:     params.Page,
		PageSize: params.PageSize,
	})
	if err != nil {
		return
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return
	}

	resp = &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(resultJSON),
			},
		},
	}
	return
}
