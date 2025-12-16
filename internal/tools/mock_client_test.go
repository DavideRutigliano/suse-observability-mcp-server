package tools

import (
	"context"
	"time"

	"suse-observability-mcp/client/suseobservability"

	"github.com/stretchr/testify/mock"
)

type MockSuseObservabilityClient struct {
	mock.Mock
}

func (m *MockSuseObservabilityClient) GetBoundMetricsWithData(ctx context.Context, componentID int64, start, end time.Time) (*suseobservability.BoundMetricsResponse, error) {
	args := m.Called(ctx, componentID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*suseobservability.BoundMetricsResponse), args.Error(1)
}

func (m *MockSuseObservabilityClient) QueryRangeMetric(ctx context.Context, query string, start time.Time, end time.Time, step, timeout string) (*suseobservability.MetricQueryResponse, error) {
	args := m.Called(ctx, query, start, end, step, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*suseobservability.MetricQueryResponse), args.Error(1)
}

func (m *MockSuseObservabilityClient) GetComponent(ctx context.Context, componentID int64) (*suseobservability.ComponentResponse, error) {
	args := m.Called(ctx, componentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*suseobservability.ComponentResponse), args.Error(1)
}

func (m *MockSuseObservabilityClient) SnapShotTopologyQuery(ctx context.Context, query string) ([]suseobservability.ViewComponent, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]suseobservability.ViewComponent), args.Error(1)
}
