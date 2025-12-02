package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListMetricsParams struct{}

type QueryMetricParams struct {
	Query string `json:"query" jsonschema:"The PromQL query to execute"`
}

type QueryRangeMetricParams struct {
	Query string `json:"query" jsonschema:"The PromQL query to execute"`
	Start string `json:"start" jsonschema:"Start time: 'now' or duration (e.g. '1h')"`
	End   string `json:"end" jsonschema:"End time: 'now' or duration (e.g. '1h')"`
	Step  string `json:"step" jsonschema:"Query resolution step width in duration format or float number of seconds"`
}

// ListMetrics lists all available metrics
func (t *Tools) ListMetrics(ctx context.Context, request *mcp.CallToolRequest, params ListMetricsParams) (*mcp.CallToolResult, any, error) {
	end := time.Now()
	start := end.Add(-1 * time.Hour)
	metrics, err := t.client.ListMetrics(start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list metrics: %w", err)
	}

	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal metrics: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(metricsJSON),
			},
		},
	}, nil, nil
}

// QueryMetric queries a single metric
func (t *Tools) QueryMetric(ctx context.Context, request *mcp.CallToolRequest, params QueryMetricParams) (*mcp.CallToolResult, any, error) {
	// Default to now if time is not provided or invalid
	at := time.Now()
	timeout := "30s"

	result, err := t.client.QueryMetric(params.Query, at, timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query metric: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(resultJSON),
			},
		},
	}, nil, nil
}

// QueryRangeMetric queries a metric over a range of time
func (t *Tools) QueryRangeMetric(ctx context.Context, request *mcp.CallToolRequest, params QueryRangeMetricParams) (*mcp.CallToolResult, any, error) {
	start, err := parseTime(params.Start)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse start time: %w", err)
	}

	end, err := parseTime(params.End)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse end time: %w", err)
	}

	step := params.Step
	if step == "" {
		step = "1m"
	}
	timeout := "30s"

	result, err := t.client.QueryRangeMetric(params.Query, start, end, step, timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query range metric: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(resultJSON),
			},
		},
	}, nil, nil
}

func parseTime(s string) (time.Time, error) {
	if s == "now" {
		return time.Now(), nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("invalid time format: %s (expected 'now' or duration like '1h')", s)
}
