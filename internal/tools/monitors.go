package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetMonitorDetailsParams struct {
	MonitorIdOrUrn string `json:"monitorIdOrUrn" jsonschema:"required,The monitor identifier (ID or URN)"`
}

type GetMonitorCheckStatesParams struct {
	MonitorIdOrUrn string `json:"monitorIdOrUrn" jsonschema:"required,The monitor identifier (ID or URN)"`
	HealthState    string `json:"healthState,omitempty" jsonschema:"Filter by health state (e.g., CRITICAL, DEVIATING, CLEAR, UNKNOWN)"`
	Limit          int    `json:"limit,omitempty" jsonschema:"Maximum number of states to return"`
	Timestamp      int64  `json:"timestamp,omitempty" jsonschema:"Timestamp for the query in milliseconds"`
}

type GetMonitorCheckStatusParams struct {
	CheckStatusId int64 `json:"checkStatusId" jsonschema:"required,The check status ID"`
	TopologyTime  int64 `json:"topologyTime,omitempty" jsonschema:"Timestamp for topology query in milliseconds"`
}

type ListMonitorsParams struct{}

// ListMonitors lists all available monitors
func (t tool) ListMonitors(ctx context.Context, request *mcp.CallToolRequest, params ListMonitorsParams) (*mcp.CallToolResult, any, error) {
	res, err := t.client.GetMonitors(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list monitors: %w", err)
	}

	jsonRes, err := json.Marshal(res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal monitors result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonRes),
			},
		},
	}, nil, nil
}

// GetMonitorDetails retrieves a specific monitor by its identifier
func (t tool) GetMonitorDetails(ctx context.Context, request *mcp.CallToolRequest, params GetMonitorDetailsParams) (*mcp.CallToolResult, any, error) {
	res, err := t.client.GetMonitor(ctx, params.MonitorIdOrUrn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get monitor details: %w", err)
	}

	jsonRes, err := json.Marshal(res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal monitor result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonRes),
			},
		},
	}, nil, nil
}

// GetMonitorCheckStates returns the check states that a monitor generated
func (t *tool) GetMonitorCheckStates(ctx context.Context, request *mcp.CallToolRequest, params GetMonitorCheckStatesParams) (*mcp.CallToolResult, any, error) {
	res, err := t.client.GetMonitorCheckStates(ctx, params.MonitorIdOrUrn, params.HealthState, params.Limit, params.Timestamp)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get monitor check states: %w", err)
	}

	jsonRes, err := json.Marshal(res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal monitor check states result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(jsonRes),
			},
		},
	}, nil, nil
}

// GetMonitorCheckStatus returns a monitor check status by check state id
func (t *tool) GetMonitorCheckStatus(ctx context.Context, request *mcp.CallToolRequest, params GetMonitorCheckStatusParams) (*mcp.CallToolResult, any, error) {
	res, err := t.client.GetMonitorCheckStatus(ctx, params.CheckStatusId, params.TopologyTime)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get monitor check status: %w", err)
	}

	jsonRes, err := json.Marshal(res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal monitor check status result: %w", err)
	}

	// Add helpful context
	triggeredTime := time.UnixMilli(res.TriggeredTimestamp).Format(time.RFC3339)
	context := fmt.Sprintf("Check Status for Monitor '%s' (Health: %s)\nTriggered at: %s\nMessage: %s",
		res.MonitorName, res.Health, triggeredTime, res.Message)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: context + "\n\n" + string(jsonRes),
			},
		},
	}, nil, nil
}
