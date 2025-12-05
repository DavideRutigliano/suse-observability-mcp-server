package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"suse-observability-mcp/client/suseobservability"
)

type QueryTracesCriteria struct {
	AttributeFilters map[string][]string `json:"attributes" jsonschema:"Optional attributes to filter. The key is the attribute name, the value is a list of valid values for that key. Leave it empty to ignore this filter. Exemple attributes are: service.name, service.namespace, component among others."`
}

type AttributeCriteria struct {
	Attribute string `json:"attribute" jsonschema:"The attribute for which you want to discover possible values."`
}

func (t tool) GetValuesForAttributeFilters(ctx context.Context, request *mcp.CallToolRequest, att AttributeCriteria) (resp *mcp.CallToolResult, a any, err error) {
	result, err := t.client.RetrievePossibleValuesForAttributeFilters(ctx, att.Attribute)
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

func (t tool) GetAttributeFilters(ctx context.Context, request *mcp.CallToolRequest, criteria QueryTracesCriteria) (resp *mcp.CallToolResult, a any, err error) {
	result, err := t.client.RetrieveAllAttributeFilters(ctx)
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
				Attributes: suseobservability.ConstrainedAttributes(criteria.AttributeFilters),
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
