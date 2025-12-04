package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"suse-observability-mcp/client/suseobservability"
)

type QueryTracesCriteria struct {
	ServiceName      []string `json:"serviceName" jsonschema:"The name(s) of the service(s) that you want to search for tracing data"`
	ServiceNamespace []string `json:"serviceNamespace" jsonschema:"The namespace(s) of the service(s) that you want to search for tracing data"`
}

func (t tool) QueryTraces(ctx context.Context, request *mcp.CallToolRequest, criteria QueryTracesCriteria) (resp *mcp.CallToolResult, a any, err error) {
	now := time.Now()
	result, err := t.client.RetrieveTraces(ctx, suseobservability.TracesRequest{
		Params: suseobservability.QueryParams{
			Start:    now.Add(-time.Hour),
			End:      now,
			Page:     0,
			PageSize: 100,
		},
		Body: suseobservability.TracesRequestBody{
			PrimarySpanFilter: suseobservability.PrimarySpanFilter{
				Attributes: suseobservability.ConstrainedAttributes{
					ServiceName:      criteria.ServiceName,
					ServiceNamespace: criteria.ServiceNamespace,
				},
			},
		},
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
