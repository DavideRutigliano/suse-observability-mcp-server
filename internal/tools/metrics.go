package tools

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"suse-observability-mcp/client/suseobservability"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListMetricsParams struct {
	SearchPattern string `json:"search_pattern" jsonschema:"required,A regex pattern to search for specific metrics (e.g. 'cpu' 'memory' 'redis')"`
}

type QueryRangeMetricParams struct {
	Query string `json:"query" jsonschema:"The PromQL query to execute"`
	Start string `json:"start" jsonschema:"Start time: 'now' or duration (e.g. '1h')"`
	End   string `json:"end" jsonschema:"End time: 'now' or duration (e.g. '1h')"`
	Step  string `json:"step" jsonschema:"Query resolution step width in duration format or float number of seconds"`
}

func (t tool) ListMetrics(ctx context.Context, request *mcp.CallToolRequest, params ListMetricsParams) (*mcp.CallToolResult, any, error) {
	// Validate required parameter
	if params.SearchPattern == "" {
		return nil, nil, fmt.Errorf("search_pattern is required")
	}

	end := time.Now()
	start := end.Add(-1 * time.Hour)
	metrics, err := t.client.ListMetrics(ctx, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list metrics: %w", err)
	}

	// Filter metrics by search_pattern
	re, err := regexp.Compile(params.SearchPattern)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var filteredMetrics []string
	for _, m := range metrics {
		if re.MatchString(m) {
			filteredMetrics = append(filteredMetrics, m)
		}
	}

	if len(filteredMetrics) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("No metrics found matching '%s'.", params.SearchPattern),
				},
			},
		}, nil, nil
	}

	// Limit to 50 metrics to avoid timeouts
	const maxMetrics = 50
	metricsToProcess := filteredMetrics
	truncated := false
	if len(filteredMetrics) > maxMetrics {
		metricsToProcess = filteredMetrics[:maxMetrics]
		truncated = true
	}

	// Build table with labels
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d metrics matching '%s'", len(filteredMetrics), params.SearchPattern))
	if truncated {
		sb.WriteString(fmt.Sprintf(" (showing first %d)", maxMetrics))
	}
	sb.WriteString(":\n\n")
	sb.WriteString("| Metric Name | Labels |\n")
	sb.WriteString("|---|---|\n")

	for _, metricName := range metricsToProcess {
		labels, err := t.client.GetMetricLabels(ctx, metricName, start, end)
		if err != nil {
			// If we can't get labels, just show the metric with no labels
			sb.WriteString(fmt.Sprintf("| %s | - |\n", metricName))
			continue
		}

		labelsStr := "-"
		if len(labels) > 0 {
			labelsStr = strings.Join(labels, ", ")
		}
		sb.WriteString(fmt.Sprintf("| %s | %s |\n", metricName, labelsStr))
	}

	if truncated {
		sb.WriteString(fmt.Sprintf("\n_Note: Showing first %d of %d metrics. Use a more specific search pattern to narrow results._\n", maxMetrics, len(filteredMetrics)))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: sb.String(),
			},
		},
	}, nil, nil
}

// QueryRangeMetric queries a metric over a range of time
func (t tool) QueryRangeMetric(ctx context.Context, request *mcp.CallToolRequest, params QueryRangeMetricParams) (*mcp.CallToolResult, any, error) {
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
		return "No data found."
	}

	// Collect all unique label keys across all series
	labelKeys := make(map[string]bool)
	for _, res := range metricsResult {
		for k := range res.Labels {
			if k != "__name__" { // Skip __name__ as it's often the query itself
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
