package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"suse-observability-mcp/client/suseobservability"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type QueryMetricParams struct {
	Query string `json:"query" jsonschema:"The PromQL query to execute"`
	Start string `json:"start" jsonschema:"Start time: 'now' or duration (e.g. '1h')"`
	End   string `json:"end" jsonschema:"End time: 'now' or duration (e.g. '1h')"`
	Step  string `json:"step" jsonschema:"Query resolution step width in duration format or float number of seconds"`
}

type ListMetricsParams struct {
	ComponentID int64 `json:"component_id" jsonschema:"required,The ID of the component to list bound metrics for"`
}

// ListMetrics lists bound metrics for a specific component
func (t tool) ListMetrics(ctx context.Context, request *mcp.CallToolRequest, params ListMetricsParams) (*mcp.CallToolResult, any, error) {
	// Default time range: last 1 hour
	end := time.Now()
	start := end.Add(-1 * time.Hour)

	boundMetrics, err := t.client.GetBoundMetricsWithData(ctx, params.ComponentID, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list bound metrics: %w", err)
	}

	if len(boundMetrics.BoundMetrics) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("No bound metrics found for component ID %d.", params.ComponentID),
				},
			},
		}, nil, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d bound metrics for component ID %d:\n\n", len(boundMetrics.BoundMetrics), params.ComponentID))
	sb.WriteString("| Metric Name | Unit | Query Expression |\n")
	sb.WriteString("|---|---|---|\n")

	for _, bm := range boundMetrics.BoundMetrics {
		for _, bq := range bm.BoundQueries {
			sb.WriteString(fmt.Sprintf("| %s | %s | `%s` |\n", bm.Name, bm.Unit, bq.Expression))
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: sb.String(),
			},
		},
	}, nil, nil
}

// QueryMetric queries a metric over a range of time
func (t tool) QueryMetric(ctx context.Context, request *mcp.CallToolRequest, params QueryMetricParams) (*mcp.CallToolResult, any, error) {
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

	result, err := t.client.QueryRangeMetric(ctx, params.Query, start, end, step, timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query range metri c: %w", err)
	}

	output := formatMetrics(result.Data.Result, params.Query)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output,
			},
		},
	}, nil, nil
}

func formatMetrics(metricsResult []suseobservability.MetricResult, queryName string) string {
	if len(metricsResult) == 0 {
		return fmt.Sprintf("No data found for query: %s", queryName)
	}

	// Collect all unique label keys across all series
	labelKeys := make(map[string]bool)
	for _, res := range metricsResult {
		for k := range res.Labels {
			if k != "__name__" {
				labelKeys[k] = true
			}
		}
	}

	// Convert to sorted slice for consistent column order
	var sortedKeys []string
	for k := range labelKeys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	var sb strings.Builder

	// Header
	sb.WriteString("| Timestamp | Value |")
	for _, k := range sortedKeys {
		sb.WriteString(fmt.Sprintf(" %s |", k))
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString("|---|---|")
	for range sortedKeys {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")

	// Data rows
	for _, res := range metricsResult {
		for _, p := range res.Points {
			ts := time.Unix(p.Timestamp, 0).Format(time.RFC3339)
			sb.WriteString(fmt.Sprintf("| %s | %.4f |", ts, p.Value))

			for _, k := range sortedKeys {
				val := res.Labels[k]
				if val == "" {
					val = "-"
				}
				sb.WriteString(fmt.Sprintf(" %s |", val))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
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
