package tools

import (
	"context"
	"time"

	"suse-observability-mcp/client/suseobservability"
)

type SuseObservabilityClient interface {
	GetBoundMetricsWithData(ctx context.Context, componentID int64, start, end time.Time) (*suseobservability.BoundMetricsResponse, error)
	QueryRangeMetric(ctx context.Context, query string, start time.Time, end time.Time, step, timeout string) (*suseobservability.MetricQueryResponse, error)
	GetComponent(ctx context.Context, componentID int64) (*suseobservability.ComponentResponse, error)
	SnapShotTopologyQuery(ctx context.Context, query string) ([]suseobservability.ViewComponent, error)
}

type tool struct {
	client SuseObservabilityClient
}

// NewBaseTool returns a tool factory
func NewBaseTool(c SuseObservabilityClient) (t *tool) {
	t = new(tool)
	t.client = c
	return
}
