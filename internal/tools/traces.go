package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"suse-observability-mcp/client/suseobservability"
)

type ListTracesParams struct {
	ComponentID int64 `json:"component_id" jsonschema:"required,The ID of the component to list bound traces for"`
}

type GetTraceParams struct {
	TraceID string `json:"trace_id" jsonschema:"the ID of the trace you want to retrieve and inspect"`
}

func (t tool) GetTrace(ctx context.Context, request *mcp.CallToolRequest, params GetTraceParams) (resp *mcp.CallToolResult, a any, err error) {
	traceData, err := t.client.GetTrace(ctx, params.TraceID)

	outputBytes, err := json.Marshal(traceData)
	if err != nil {
		return
	}

	resp = &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(outputBytes),
			},
		},
	}
	return
}

func (t tool) ListTraces(ctx context.Context, request *mcp.CallToolRequest, params ListTracesParams) (resp *mcp.CallToolResult, a any, err error) {
	query := "(label IN (\"stackpack:open-telemetry\") AND type IN (\"otel service\"))"
	components, err := t.client.SnapShotTopologyQuery(ctx, query)
	tags := make([]string, 0)

	for _, c := range components {
		if c.ID == params.ComponentID {
			tags = c.Tags
			break
		}
	}
	if len(tags) == 0 {
		err = errors.New("Component not found")
		return
	}

	var name, namespace string
	for _, tag := range tags {
		key, value := splitTag(tag)
		if key == "service.name" {
			name = value
		}
		if key == "service.namespace" {
			namespace = value
		}
	}
	if name == "" || namespace == "" {
		err = errors.New("Component has no service name and namespace defined")
		return
	}

	now := time.Now()
	result, err := t.client.QueryTraces(ctx, suseobservability.TracesRequest{
		Params: suseobservability.QueryParams{
			Start:    now.Add(-time.Hour),
			End:      now,
			Page:     0,
			PageSize: 20,
		},
		Body: suseobservability.TracesRequestBody{
			PrimarySpanFilter: suseobservability.PrimarySpanFilter{
				Attributes: suseobservability.ConstrainedAttributes{
					ServiceName:      []string{name},
					ServiceNamespace: []string{namespace},
				},
			},
		},
	})
	if err != nil {
		return
	}

	traceIDs := make([]string, len(result.Traces))
	for i := range result.Traces {
		traceIDs[i] = result.Traces[i].TraceID
	}

	outputBytes, err := json.Marshal(traceIDs)
	if err != nil {
		return
	}

	resp = &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(outputBytes),
			},
		},
	}
	return
}

func splitTag(input string) (key string, value string) {
	key, value, _ = strings.Cut(input, ":")
	return
}
